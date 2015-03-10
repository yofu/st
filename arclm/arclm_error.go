package arclm

import (
	"fmt"
)

type Elemer interface {
	Number() int
	Enode(int) int
}

type BrittleFailureError struct {
	elem  Elemer
	index int
}

func BrittleFailure(elem Elemer, index int) BrittleFailureError {
	return BrittleFailureError{elem, index}
}

func (b BrittleFailureError) Error() string {
	return fmt.Sprintf("BRITTLE FAILURE: ELEM %d NODE %d", b.elem.Number(), b.elem.Enode(b.index))
}

type YieldedError struct {
	elem  Elemer
	index int
}

func Yielded(elem Elemer, index int) YieldedError {
	return YieldedError{elem, index}
}

func (b YieldedError) Error() string {
	return fmt.Sprintf("YIELDED: ELEM %d NODE %d", b.elem.Number(), b.elem.Enode(b.index))
}
