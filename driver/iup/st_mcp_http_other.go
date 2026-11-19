//go:build !windows

package stgui

import (
	"context"
	"os/exec"
	"sync"
)

var mcpHTTPStartupMutex sync.Mutex

func configureMCPHTTPProcess(cmd *exec.Cmd) {}
func withMCPHTTPStartupLock(ctx context.Context, address string, action func() error) error {
	mcpHTTPStartupMutex.Lock()
	defer mcpHTTPStartupMutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	return action()
}
