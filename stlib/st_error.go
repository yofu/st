package st

import (
	"fmt"
)

type ParallelError struct {
	funcname string
}

func (p ParallelError) Error() string {
	return fmt.Sprintf("%s: Not Parallel", p.funcname)
}

func NotParallel(fn string) ParallelError {
	return ParallelError{fn}
}

type ElemDivisionError struct {
	funcname    string
	description string
}

func (e ElemDivisionError) Error() string {
	return fmt.Sprintf("%s: %s", e.funcname, e.description)
}

func DivideAtEnod(fn string) ElemDivisionError {
	return ElemDivisionError{fn, "Divide At Enod"}
}

type EtypeError struct {
	shouldbe string
	funcname string
}

func (e EtypeError) Error() string {
	return fmt.Sprintf("%s: Not %sElem", e.funcname, e.shouldbe)
}

func NotLineElem(fn string) EtypeError {
	return EtypeError{"Line", fn}
}

func NotPlateElem(fn string) EtypeError {
	return EtypeError{"Plate", fn}
}

type ZeroAllowableError struct {
	Name string
}

func (z ZeroAllowableError) Error() string {
	return fmt.Sprintf("Rate: %s == 0.0", z.Name)
}

type BrittleFailureError struct {
	elem  *Elem
	index int
}

func BrittleFailure(elem *Elem, index int) BrittleFailureError {
	return BrittleFailureError{elem, index}
}

func (b BrittleFailureError) Error() string {
	return fmt.Sprintf("BRITTLE FAILURE: ELEM %d NODE %d", b.elem.Num, b.elem.Enod[b.index].Num)
}

type YieldedError struct {
	elem  *Elem
	index int
}

func Yielded(elem *Elem, index int) YieldedError {
	return YieldedError{elem, index}
}

func (b YieldedError) Error() string {
	return fmt.Sprintf("YIELDED: ELEM %d NODE %d", b.elem.Num, b.elem.Enod[b.index].Num)
}

type ArgumentError struct {
	name string
	desc string
}

func (ne ArgumentError) Error() string {
	return fmt.Sprintf("%s: %s", ne.name, ne.desc)
}

func NotEnoughArgs(name string) ArgumentError {
	return ArgumentError{name, "not enough argument"}
}
