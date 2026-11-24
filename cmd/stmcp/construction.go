package main

import (
	"errors"
	"fmt"
	st "github.com/yofu/st/stlib"
	"os"
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

// Stage extended construction against an isolated snapshot and commit once.
// Legacy line construction also uses this path after an explicit unknown
// action reply, never after a timeout that could follow a successful commit.
func (s *modelSession) constructFromSnapshot(id string, q st.GUIRequest, build *workflowConstruction) (st.GUIResponse, error) {
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
	receipt, updatedLST, err := build.onPrivateFrameWithLST(temp.frame, base.Snapshot.LST, temp.lstPath)
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
