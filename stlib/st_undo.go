package st

import (
	"fmt"
)

type UndoStack struct {
	size int
	position int
	stack []*Frame
	enabled bool
}

func NewUndoStack(size int) *UndoStack {
	return &UndoStack{
		size: size,
		position: 0,
		stack: make([]*Frame, size),
		enabled: true,
	}
}

func (u *UndoStack) Redo() (*Frame, error) {
	if !u.enabled {
		return nil, nil
	}
	u.position--
	if u.position < 0 {
		u.position = 0
		return nil, fmt.Errorf("cannot redo any more")
	}
	return u.stack[u.position].Snapshot(), nil
}

func (u *UndoStack) Undo() (*Frame, error) {
	if !u.enabled {
		return nil, nil
	}
	u.position++
	if u.position >= u.size {
		u.position = u.size - 1
		return nil, fmt.Errorf("cannot undo any more")
	}
	if u.stack[u.position] == nil {
		u.position--
		return nil, fmt.Errorf("cannot undo any more")
	}
	return u.stack[u.position].Snapshot(), nil
}

func (u *UndoStack) UseUndo(yes bool) {
	u.enabled = yes
}

func (u *UndoStack) EnableUndo() {
	u.enabled = true
}

func (u *UndoStack) DisableUndo() {
	u.enabled = false
}

func (u *UndoStack) UndoEnabled() bool {
	return u.enabled
}

func (u *UndoStack) PushUndo(frame *Frame) {
	tmp := make([]*Frame, u.size)
	tmp[0] = frame.Snapshot()
	for i := 0; i < u.size-1-u.position; i++ {
		tmp[i+1] = u.stack[i+u.position]
	}
	u.stack = tmp
	u.position = 0
}
