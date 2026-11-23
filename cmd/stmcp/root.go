package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func (s *modelSession) modelRootPath() string {
	if s.root != nil {
		return s.root.Name()
	}
	return s.rootPath
}

func (s *modelSession) guiModelPath() string {
	if s.source == "" {
		return ""
	}
	return filepath.Join(s.modelRootPath(), s.source)
}

// GUI requests operate on the active desktop model without opening its files
// in the MCP process. Retry direct access only for tools which need it.
func (s *modelSession) requireRoot() error {
	if s.root != nil {
		return nil
	}
	root, err := os.OpenRoot(s.rootPath)
	if err != nil {
		return fmt.Errorf("direct access to model root %q failed: %w; start the local HTTP MCP server in the desktop user session and reconnect to its URL", s.rootPath, err)
	}
	s.root = root
	return nil
}
