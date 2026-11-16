package st

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var guiPipeID = regexp.MustCompile("^[0-9a-f]{32}$")

func guiPipeName(id string) (*uint16, error) {
	if !guiPipeID.MatchString(id) {
		return nil, fmt.Errorf("invalid ST GUI connection ID")
	}
	return windows.UTF16PtrFromString(`\\.\pipe\LOCAL\st-mcp-` + id)
}

type guiPipe struct {
	ctx    context.Context
	handle windows.Handle
}

func (p guiPipe) operation(start func(*uint32, *windows.Overlapped) error) (int, error) {
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(event)
	ov := windows.Overlapped{HEvent: event}
	var count uint32
	err = start(&count, &ov)
	if !errors.Is(err, windows.ERROR_IO_PENDING) {
		return int(count), err
	}
	for {
		if err := p.ctx.Err(); err != nil {
			_ = windows.CancelIoEx(p.handle, &ov)
			_ = windows.GetOverlappedResult(p.handle, &ov, &count, true)
			return 0, err
		}
		eventState, err := windows.WaitForSingleObject(event, 100)
		if err != nil || eventState == windows.WAIT_OBJECT_0 {
			if err != nil {
				_ = windows.CancelIoEx(p.handle, &ov)
				_ = windows.GetOverlappedResult(p.handle, &ov, &count, true)
				return 0, err
			}
			err = windows.GetOverlappedResult(p.handle, &ov, &count, false)
			return int(count), err
		}
	}
}

func (p guiPipe) Read(data []byte) (int, error) {
	n, err := p.operation(func(n *uint32, ov *windows.Overlapped) error { return windows.ReadFile(p.handle, data, n, ov) })
	runtime.KeepAlive(data)
	if errors.Is(err, windows.ERROR_BROKEN_PIPE) {
		return n, io.EOF
	}
	return n, err
}

func (p guiPipe) Write(data []byte) (int, error) {
	n, err := p.operation(func(n *uint32, ov *windows.Overlapped) error { return windows.WriteFile(p.handle, data, n, ov) })
	runtime.KeepAlive(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return n, err
}

func RunGUIPipe(ctx context.Context, id string, action func(GUIRequest) GUIResponse, ready chan<- error) {
	name, err := guiPipeName(id)
	if err != nil {
		ready <- err
		return
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		ready <- err
		return
	}
	// Restricted desktop MCP processes also need the restricting SIDs.
	// The pipe is local and its unpredictable connection ID is shared by the user.
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;GA;;;" + user.User.Sid.String() + ")(A;;GRGW;;;RC)(A;;GRGW;;;AC)")
	if err != nil {
		ready <- err
		return
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	h, err := windows.CreateNamedPipe(name, windows.PIPE_ACCESS_DUPLEX|windows.FILE_FLAG_OVERLAPPED|windows.FILE_FLAG_FIRST_PIPE_INSTANCE,
		windows.PIPE_TYPE_BYTE|windows.PIPE_REJECT_REMOTE_CLIENTS, 1, 65536, 65536, 0, &sa)
	ready <- err
	if err != nil {
		return
	}
	defer windows.CloseHandle(h)
	for ctx.Err() == nil {
		pipe := guiPipe{ctx, h}
		_, err := pipe.operation(func(_ *uint32, ov *windows.Overlapped) error { return windows.ConnectNamedPipe(h, ov) })
		if errors.Is(err, windows.ERROR_NO_DATA) {
			_ = windows.DisconnectNamedPipe(h)
			continue
		}
		if err != nil && !errors.Is(err, windows.ERROR_PIPE_CONNECTED) {
			return
		}
		func() {
			exchange, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			pipe.ctx = exchange
			var q GUIRequest
			var response GUIResponse
			if err := json.NewDecoder(io.LimitReader(pipe, 180<<20)).Decode(&q); err != nil {
				response.Error = "invalid GUI request"
			} else {
				response = action(q)
			}
			if err := json.NewEncoder(pipe).Encode(response); err != nil {
				return
			}
			// Wait for the client to close after reading, without blocking shutdown.
			var ack [1]byte
			_, _ = pipe.Read(ack[:])
		}()
		_ = windows.DisconnectNamedPipe(h)
	}
}

func ExchangeGUI(ctx context.Context, id string, q GUIRequest) (GUIResponse, error) {
	var result GUIResponse
	name, err := guiPipeName(id)
	if err != nil {
		return result, err
	}
	var handle windows.Handle
	for {
		handle, err = windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED, 0)
		if err == nil {
			break
		}
		if !errors.Is(err, windows.ERROR_PIPE_BUSY) {
			return result, fmt.Errorf("connect to ST GUI: %w", err)
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	defer windows.CloseHandle(handle)
	pipe := guiPipe{ctx, handle}
	if err = json.NewEncoder(pipe).Encode(q); err != nil {
		return result, err
	}
	if err = json.NewDecoder(io.LimitReader(pipe, 16<<20)).Decode(&result); err != nil {
		return result, err
	}
	if result.Error != "" {
		return result, fmt.Errorf("ST GUI: %s", result.Error)
	}
	return result, nil
}
