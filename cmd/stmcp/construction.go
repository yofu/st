package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	st "github.com/yofu/st/stlib"
)

func (s *modelSession) saveGUICopy(path string) error {
	if err := s.requireRoot(); err != nil {
		return err
	}
	lst := st.Ce(path, ".lst")
	if _, err := s.root.Lstat(lst); err == nil {
		return fmt.Errorf("companion LST already exists: %s", lst)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := writeNew(s.root, path, s.raw); err != nil {
		return err
	}
	if len(s.lst) > 0 {
		if err := writeNew(s.root, lst, s.lst); err != nil {
			return errors.Join(err, s.root.Remove(path))
		}
	}
	return nil
}

func (s *modelSession) loadNodeUpdatesFile(path string) ([]st.BuildNode, error) {
	if err := s.requireRoot(); err != nil {
		return nil, err
	}
	if !filepath.IsLocal(path) {
		return nil, fmt.Errorf("update_nodes_file must be a relative path within the server root")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv", ".json", ".jsonl":
	default:
		return nil, fmt.Errorf("update_nodes_file supports .csv, .json or .jsonl, got %q", ext)
	}
	clean := filepath.Clean(path)
	f, err := s.root.Open(clean)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("update_nodes_file must be a regular file")
	}
	const maxBytes = 32 << 20
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("update_nodes_file exceeds %d bytes", maxBytes)
	}
	return parseNodeUpdatesData(ext, data)
}

func parseNodeUpdatesData(ext string, data []byte) ([]st.BuildNode, error) {
	const maxNodes = 100000
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	switch strings.ToLower(ext) {
	case ".json":
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		var nodes []st.BuildNode
		if err := dec.Decode(&nodes); err != nil {
			return nil, fmt.Errorf("parse update_nodes_file JSON: %w", err)
		}
		var trailing any
		if err := dec.Decode(&trailing); err != io.EOF {
			return nil, fmt.Errorf("update_nodes_file JSON has trailing data")
		}
		if len(nodes) == 0 {
			return nil, fmt.Errorf("update_nodes_file contains no node updates")
		}
		if len(nodes) > maxNodes {
			return nil, fmt.Errorf("update_nodes_file exceeds %d nodes", maxNodes)
		}
		return nodes, nil
	case ".jsonl":
		sc := bufio.NewScanner(bytes.NewReader(data))
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		var nodes []st.BuildNode
		lineNo := 0
		for sc.Scan() {
			lineNo++
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			dec := json.NewDecoder(strings.NewReader(line))
			dec.DisallowUnknownFields()
			var n st.BuildNode
			if err := dec.Decode(&n); err != nil {
				return nil, fmt.Errorf("update_nodes_file line %d: %w", lineNo, err)
			}
			var trailing any
			if err := dec.Decode(&trailing); err != io.EOF {
				return nil, fmt.Errorf("update_nodes_file line %d has trailing data", lineNo)
			}
			nodes = append(nodes, n)
			if len(nodes) > maxNodes {
				return nil, fmt.Errorf("update_nodes_file exceeds %d nodes", maxNodes)
			}
		}
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("read update_nodes_file JSONL: %w", err)
		}
		if len(nodes) == 0 {
			return nil, fmt.Errorf("update_nodes_file contains no node updates")
		}
		return nodes, nil
	case ".csv":
		r := csv.NewReader(bytes.NewReader(data))
		r.FieldsPerRecord = -1
		r.TrimLeadingSpace = true
		var nodes []st.BuildNode
		rowNum := 0
		sawFirstRow := false
		for {
			rec, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("parse update_nodes_file CSV: %w", err)
			}
			rowNum++
			if len(rec) == 0 || (len(rec) == 1 && strings.TrimSpace(rec[0]) == "") {
				continue
			}
			first := strings.TrimSpace(rec[0])
			if strings.HasPrefix(first, "#") {
				continue
			}
			if len(rec) != 4 {
				return nil, fmt.Errorf("update_nodes_file CSV row %d: expected 4 columns (id,x,y,z), got %d", rowNum, len(rec))
			}
			id, err := strconv.Atoi(first)
			if err != nil {
				if !sawFirstRow {
					sawFirstRow = true
					continue
				}
				return nil, fmt.Errorf("update_nodes_file CSV row %d: invalid node ID %q", rowNum, first)
			}
			sawFirstRow = true
			xyz := make([]float64, 3)
			for i := 0; i < 3; i++ {
				v, err := strconv.ParseFloat(strings.TrimSpace(rec[i+1]), 64)
				if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
					return nil, fmt.Errorf("update_nodes_file CSV row %d col %d: invalid coordinate %q", rowNum, i+2, rec[i+1])
				}
				xyz[i] = v
			}
			nodes = append(nodes, st.BuildNode{ID: id, XYZ: xyz})
			if len(nodes) > maxNodes {
				return nil, fmt.Errorf("update_nodes_file exceeds %d nodes", maxNodes)
			}
		}
		if len(nodes) == 0 {
			return nil, fmt.Errorf("update_nodes_file contains no node updates")
		}
		return nodes, nil
	default:
		return nil, fmt.Errorf("unsupported update_nodes_file format %q", ext)
	}
}

// Stage extended construction against an isolated snapshot and commit once.
// Legacy line construction also uses this path after an explicit unknown
// action reply, never after a timeout that could follow a successful commit.
func (s *modelSession) constructFromSnapshot(id string, q st.GUIRequest, build *workflowConstruction) (st.GUIResponse, error) {
	resolved := *build
	if resolved.UpdateNodesFile != "" {
		loaded, err := s.loadNodeUpdatesFile(resolved.UpdateNodesFile)
		if err != nil {
			return st.GUIResponse{}, err
		}
		resolved.fileUpdateNodes = loaded
	}
	base, err := exchangeGUI(id, st.GUIRequest{Action: "snapshot", Revision: q.Revision})
	if err != nil {
		return base, err
	}
	if base.Path == "" && !base.DeltaSupported {
		return st.GUIResponse{}, fmt.Errorf("this untitled window uses an older st.exe; reopen ST with the updated binary before constructing")
	}
	temp := *s
	if err := temp.importGUI(id, base); err != nil {
		return base, err
	}
	digest := temp.frame.GUIModelDigest()
	receipt, updatedLST, err := resolved.onPrivateFrameWithLST(temp.frame, base.Snapshot.LST, temp.lstPath)
	if err != nil {
		return st.GUIResponse{}, err
	}
	reply, err := exchangeGUI(id, st.GUIRequest{Action: "apply", BasePath: base.Path, Revision: base.Revision, BaseDigest: digest, INP: temp.frame.GUIInpBytes(), LST: updatedLST})
	if err != nil {
		return reply, err
	}
	reply.Construction = receipt
	if reply.Revision != temp.frame.GUIRevision() || reply.Snapshot == nil {
		reply.SyncError = fmt.Sprintf("GUI applied the edit but verification failed (revision %s)", reply.Revision)
	}
	return reply, nil
}
