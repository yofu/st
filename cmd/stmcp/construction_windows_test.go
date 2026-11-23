package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	st "github.com/yofu/st/stlib"
)

func TestConstructOnOlderNamedWindow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.inp")
	start, err := readModel([]byte("NODE 11 CORD 0 0 0\n"), path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	id := hex.EncodeToString(nonce[:])
	ready := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		f := start
		st.RunGUIPipe(ctx, id, func(q st.GUIRequest) st.GUIResponse {
			switch q.Action {
			case "snapshot":
			case "apply":
				if q.BasePath != path || q.BaseDigest != f.GUIModelDigest() || q.Revision != f.GUIRevision() {
					return st.GUIResponse{Error: "stale model"}
				}
				next, err := readModel(q.INP, path)
				if err != nil {
					return st.GUIResponse{Error: err.Error()}
				}
				f = next
			default:
				return st.GUIResponse{Error: "unknown GUI action"}
			}
			snap, revision := f.GUIInputSnapshot()
			return st.GUIResponse{Snapshot: snap, Revision: revision, Path: path}
		}, ready)
	}()
	if err := <-ready; err != nil {
		t.Fatal(err)
	}
	defer func() { cancel(); <-done }()
	s := modelSession{rootPath: dir}
	defer s.closeWorkflow()
	result, err := s.call("run_model_workflow", []byte(`{"connection_id":"`+id+`","construct":{"nodes":[{"id":12,"xyz_m":[1,0,0]}]},"wait_ms":5000}`))
	if err != nil {
		t.Fatal(err)
	}
	r := result.(workflowResult)
	if r.State != "completed" || r.GUI == nil || r.ConstructionRevision == "" || r.ConstructionRevision != r.GUI.Revision || r.Construction == nil || r.Construction.Total.Nodes != 2 || s.frame.Nodes[12] == nil {
		t.Fatalf("failed older-window verification: %+v", r)
	}
	// The receipt remains available even if the ST window is already closed.
	cancel()
	<-done
	recovered, err := s.call("model_workflow_status", []byte(`{}`))
	if err != nil || recovered.(workflowResult).ConstructionRevision != r.ConstructionRevision || recovered.(workflowResult).Construction.Added.Nodes != 1 {
		t.Fatalf("recover completed workflow: %v (%v)", recovered, err)
	}
	for _, invalid := range []string{`{"workflow_id":0}`, `{"workflow_id":2}`} {
		if _, err := s.call("model_workflow_status", []byte(invalid)); err == nil {
			t.Fatalf("wrong ID accepted: %s", invalid)
		}
	}
	if _, err := s.call("cancel_model_workflow", []byte(`{}`)); err == nil {
		t.Fatal("cancellation accepted without a workflow ID")
	}
	if err := s.saveGUICopy("new.inp"); err != nil {
		t.Fatal(err)
	}
	if err := s.saveGUICopy("new.inp"); err == nil {
		t.Fatal("overwrote saved copy")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("unexpected write to the live source")
	}
	if s.root != nil {
		s.root.Close()
	}
}
