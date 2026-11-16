package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	st "github.com/yofu/st/stlib"
)

type liveAnalysisOutput struct {
	root   *os.Root
	recent []string
}

func newLiveAnalysisOutput(root *os.Root, dir string, model []byte) (*liveAnalysisOutput, error) {
	if !filepath.IsLocal(dir) || filepath.Clean(dir) == "." {
		return nil, fmt.Errorf("preview_dir must name a new directory within the model root")
	}
	if err := root.Mkdir(dir, 0700); err != nil {
		return nil, fmt.Errorf("create preview_dir: %w", err)
	}
	dst, err := root.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	output := &liveAnalysisOutput{root: dst}
	if err = output.write(st.LiveAnalysisModel, func(w io.Writer) error {
		_, err := w.Write(model)
		return err
	}); err != nil {
		dst.Close()
		return nil, err
	}
	return output, nil
}

// Publish immutable files. The GUI only consumes complete JSON documents;
// if a reader has the previous file open, retain it for the next cleanup.
func (out *liveAnalysisOutput) write(name string, write func(io.Writer) error) error {
	file, err := out.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	err = write(file)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = out.root.Remove(name)
	}
	return err
}

func (out *liveAnalysisOutput) publish(step *st.LiveAnalysisStep) error {
	name := st.LiveAnalysisStepFile(step.Step)
	if err := out.write(name, func(w io.Writer) error { return json.NewEncoder(w).Encode(step) }); err != nil {
		return err
	}
	out.recent = append(out.recent, name)
	for len(out.recent) > 2 {
		if err := out.root.Remove(out.recent[0]); err != nil && !os.IsNotExist(err) {
			break
		}
		out.recent = out.recent[1:]
	}
	return nil
}

func (out *liveAnalysisOutput) finish(state, message string) error {
	defer out.root.Close()
	return out.write(st.LiveAnalysisStatusFile, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(st.LiveAnalysisStatus{State: state, Error: message})
	})
}
