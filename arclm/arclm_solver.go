package arclm

import (
	"sync"

	"github.com/yofu/st/matrix"
)

type Solver struct {
	name  string
	solve func(*matrix.COOMatrix, int, []bool, ...[]float64) ([][]float64, error)
}

func (s Solver) Solve(gmtx *matrix.COOMatrix, csize int, conf []bool, vecs ...[]float64) ([][]float64, error) {
	return s.solve(gmtx, csize, conf, vecs...)
}

func CRS(laptime func(string)) Solver {
	return Solver{
		name: "CRS",
		solve: func(gmtx *matrix.COOMatrix, csize int, conf []bool, vecs ...[]float64) ([][]float64, error) {
			mtx := gmtx.ToCRS(csize, conf)
			laptime("ToCRS")
			answers := mtx.Solve(vecs...)
			laptime("Solve")
			return answers, nil
		},
	}
}

func CRS_CG(eps float64, laptime func(string)) Solver {
	return Solver{
		name: "CRS_CG",
		solve: func(gmtx *matrix.COOMatrix, csize int, conf []bool, vecs ...[]float64) ([][]float64, error) {
			mtx := gmtx.ToCRS(csize, conf)
			laptime("ToCRS")
			answers := make([][]float64, len(vecs))
			for i, vec := range vecs {
				answers[i] = mtx.CG(vec, eps)
			}
			laptime("Solve")
			return answers, nil
		},
	}
}

func LLS(frame *Frame, laptime func(string)) Solver {
	return Solver{
		name: "LLS",
		solve: func(gmtx *matrix.COOMatrix, csize int, conf []bool, vecs ...[]float64) ([][]float64, error) {
			mtx := gmtx.ToLLS(csize, conf)
			laptime("ToLLS")
			answers, err := mtx.Solve(frame.Pivot, vecs...)
			laptime("Solve")
			if err != nil {
				return nil, err
			}
			return answers, nil
		},
	}
}

func LLS_CG(eps float64, laptime func(string)) Solver {
	return Solver{
		name: "LLS_CG",
		solve: func(gmtx *matrix.COOMatrix, csize int, conf []bool, vecs ...[]float64) ([][]float64, error) {
			mtx := gmtx.ToLLS(csize, conf)
			mtx.DiagUp()
			laptime("ToLLS")
			answers := make([][]float64, len(vecs))
			var wg sync.WaitGroup
			for i, vec := range vecs {
				wg.Add(1)
				go func(ind int, v []float64) {
					answers[ind] = mtx.CG(v, eps)
					wg.Done()
				}(i, vec)
			}
			wg.Wait()
			laptime("Solve")
			return answers, nil
		},
	}
}

func LLS_PCG(eps float64, laptime func(string)) Solver {
	return Solver{
		name: "LLS_PCG",
		solve: func(gmtx *matrix.COOMatrix, csize int, conf []bool, vecs ...[]float64) ([][]float64, error) {
			mtx := gmtx.ToLLS(csize, conf)
			C := gmtx.ToLLS(csize, conf)
			mtx.DiagUp()
			laptime("ToLLS")
			answers := make([][]float64, len(vecs))
			for i, vec := range vecs {
				answers[i] = mtx.PCG(C, vec)
			}
			laptime("Solve")
			return answers, nil
		},
	}
}
