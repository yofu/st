package st

import (
	"fmt"
)

type TagFrame struct {
	dict map[string]*Frame
}

func NewTagFrame() *TagFrame {
	return &TagFrame{
		dict: make(map[string]*Frame),
	}
}

func (t *TagFrame) Checkout(name string) (*Frame, error) {
	f, exists := t.dict[name]
	if !exists {
		return nil, fmt.Errorf("tag %s doesn't exist", name)
	}
	t.dict[name] = f.Snapshot()
	return f, nil
}

func (t *TagFrame) AddTag(frame *Frame, name string, bang bool) error {
	if !bang {
		if _, exists := t.dict[name]; exists {
			return fmt.Errorf("tag %s already exists", name)
		}
	}
	t.dict[name] = frame.Snapshot()
	return nil
}
