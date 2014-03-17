package st

import (
    "fmt"
)

type ParallelError struct {
    funcname string
}

func (p ParallelError) Error () string {
    return fmt.Sprintf("%s: Not Parallel", p.funcname)
}

func NotParallel(fn string) ParallelError {
    return ParallelError{fn}
}

type EtypeError struct {
    shouldbe string
    funcname string
}

func (e EtypeError) Error () string {
    return fmt.Sprintf("%s: Not %sElem", e.funcname, e.shouldbe)
}

func NotLineElem(fn string) EtypeError {
    return EtypeError{"Line", fn}
}

func NotPlateElem(fn string) EtypeError {
    return EtypeError{"Plate", fn}
}
