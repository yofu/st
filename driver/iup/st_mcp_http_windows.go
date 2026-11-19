package stgui

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os/exec"
	"runtime"
	"syscall"

	"golang.org/x/sys/windows"
)

func configureMCPHTTPProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

func withMCPHTTPStartupLock(ctx context.Context, address string, action func() error) error {
	// Windows mutex ownership belongs to the calling OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	key := sha256.Sum256([]byte(address))
	name, err := windows.UTF16PtrFromString(fmt.Sprintf(`Local\st-mcp-http-%x`, key[:16]))
	if err != nil {
		return err
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil && err != windows.ERROR_ALREADY_EXISTS {
		return err
	}
	defer windows.CloseHandle(handle)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		state, err := windows.WaitForSingleObject(handle, 100)
		if err != nil {
			return err
		}
		if state == windows.WAIT_OBJECT_0 || state == windows.WAIT_ABANDONED {
			defer windows.ReleaseMutex(handle)
			return action()
		}
	}
}
