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

type Messager interface {
	Message() string
}

type Message string

func (m Message) Error() string {
	return fmt.Sprintf("message: %s", string(m))
}

func (m Message) Message() string {
	return string(m)
}
