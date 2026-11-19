package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	st "github.com/yofu/st/stlib"
)

type saveFile struct {
	path           string
	data, original []byte
	stage, backup  string
}

func writeNew(root *os.Root, path string, data []byte) error {
	f, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		_ = root.Remove(path)
	}
	return err
}

func (s *modelSession) verifyOriginal(file saveFile) error {
	if err := s.requireRoot(); err != nil {
		return err
	}
	f, err := s.root.Open(file.path)
	if os.IsNotExist(err) && file.original == nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s changed since opening: %w", file.path, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, int64(len(file.original))+1))
	if err != nil {
		return err
	}
	if file.original == nil || !bytes.Equal(data, file.original) {
		return fmt.Errorf("%s changed since opening; reopen and preview again", file.path)
	}
	return nil
}

// Paths are opened through os.Root; reject symlinks before the final rename,
// which Go 1.24 exposes only outside os.Root.
func (s *modelSession) renameModelFile(from, to string) error {
	abs, err := filepath.Abs(s.modelRootPath())
	if err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}
	for _, name := range []string{from, to} {
		if !filepath.IsLocal(name) {
			return fmt.Errorf("invalid save path")
		}
		for p := name; p != "."; p = filepath.Dir(p) {
			info, err := s.root.Lstat(p)
			if os.IsNotExist(err) && p == to {
				continue
			}
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("cannot replace a symlink: %s", p)
			}
		}
	}
	return os.Rename(filepath.Join(root, from), filepath.Join(root, to))
}

func (s *modelSession) saveFiles(id uint64, path string, overwrite bool) (any, error) {
	if err := s.requireRoot(); err != nil {
		return nil, err
	}
	p := s.pending
	if p == nil || p.ID != id {
		return nil, fmt.Errorf("preview the changes and use the current proposal_id")
	}
	if overwrite {
		path = s.source
	}
	path, err := inpPath(path)
	if err != nil {
		return nil, err
	}
	if !overwrite && strings.EqualFold(path, s.source) {
		return nil, fmt.Errorf("choose a different output filename")
	}
	files := []saveFile{{path: path, data: p.data, original: s.raw}}
	lst := s.proposedLst(p)
	lstPath := st.Ce(path, ".lst")
	if overwrite {
		lstPath = s.lstPath
		if lstPath == "" {
			lstPath = st.Ce(path, ".lst")
		}
	}
	if lst != nil && (!overwrite || !bytes.Equal(s.lst, lst)) {
		files = append(files, saveFile{path: lstPath, data: lst, original: s.lst})
	}
	result := map[string]any{"path": path, "proposal_id": id, "changes": len(p.Changes)}
	if lst != nil {
		result["lst_path"] = lstPath
	}
	if !overwrite {
		for i, f := range files {
			if err := writeNew(s.root, f.path, f.data); err != nil {
				for _, written := range files[:i] {
					err = errors.Join(err, s.root.Remove(written.path))
				}
				return nil, err
			}
		}
		return result, nil
	}
	next, err := readModel(p.data, s.parsePath)
	if err != nil {
		return nil, err
	}
	if lst != nil {
		if err = readLst(next, lst, lstPath); err != nil {
			return nil, err
		}
	}
	if lst != nil && len(files) == 1 {
		if err = s.verifyOriginal(saveFile{path: lstPath, original: s.lst}); err != nil {
			return nil, err
		}
	}
	var key [8]byte
	if _, err = rand.Read(key[:]); err != nil {
		return nil, err
	}
	suffix := hex.EncodeToString(key[:])
	for i := range files {
		f := &files[i]
		if err = s.verifyOriginal(*f); err != nil {
			return nil, err
		}
		stage := f.path + ".stage-" + suffix
		if err = writeNew(s.root, stage, f.data); err != nil {
			return nil, err
		}
		f.stage = stage
		defer s.root.Remove(stage)
		if f.original != nil {
			f.backup = f.path + ".bak-" + suffix
			if err = writeNew(s.root, f.backup, f.original); err != nil {
				return nil, err
			}
		}
	}
	for _, f := range files {
		if err = s.verifyOriginal(f); err != nil {
			return nil, err
		}
	}
	for i, f := range files {
		if err = s.renameModelFile(f.stage, f.path); err == nil {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			prev := files[j]
			if verifyErr := s.verifyOriginal(saveFile{path: prev.path, original: prev.data}); verifyErr != nil {
				err = errors.Join(err, fmt.Errorf("could not roll back %s from %s: %w", prev.path, prev.backup, verifyErr))
				continue
			}
			if prev.original == nil {
				err = errors.Join(err, s.root.Remove(prev.path))
				continue
			}
			rollback := prev.stage + ".restore"
			if restoreErr := writeNew(s.root, rollback, prev.original); restoreErr != nil {
				err = errors.Join(err, restoreErr)
				continue
			}
			restoreErr := s.renameModelFile(rollback, prev.path)
			_ = s.root.Remove(rollback)
			if restoreErr != nil {
				err = errors.Join(err, fmt.Errorf("restore %s from %s: %w", prev.path, prev.backup, restoreErr))
			}
		}
		return nil, err
	}
	backups := make([]string, 0, len(files))
	for _, f := range files {
		if f.backup != "" {
			backups = append(backups, f.backup)
		}
	}
	result["backups"] = backups
	s.frame, s.raw, s.lst, s.lstPath, s.pending = next, p.data, lst, lstPath, nil
	return result, nil
}
