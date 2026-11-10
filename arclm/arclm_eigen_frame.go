package arclm

import (
	"fmt"

	"github.com/yofu/st/matrix"
)

// bucklingPreloadGeometric recovers the equilibrated local end forces from
// the reference load. The geometry remains that used to assemble KE.
func (f *Frame) bucklingPreloadGeometric(elems []linearElement, load linearLoads, u []float64) (*matrix.COOMatrix, [][][]float64, error) {
	kg := matrix.NewCOOMatrix(len(u))
	local := make([][][]float64, len(elems))
	for i, el := range elems {
		member := f.Elems[i]
		if !member.IsValid {
			continue
		}
		e := *member
		e.Stress = matrix.MatrixVector(el.local, el.localDisplacement(u))
		for j, q := range load.fixed[i] {
			e.Stress[j] += q
		}
		g, err := e.bucklingGeometricStiffness()
		if err != nil {
			return nil, nil, fmt.Errorf("ELEM %d buckling geometric stiffness: %w", e.Num, err)
		}
		local[i] = g
		global := Transformation(g, el.transform)
		for j, row := range el.dofs {
			for k, col := range el.dofs {
				if v := global[j][k]; v != 0 {
					kg.Add(row, col, v)
				}
			}
		}
	}
	return kg, local, nil
}

// previewEigenMode supplies a single mode to the LAP consumer. OTP output
// uses EigenVector directly and the enclosing analysis restores the frame.
func (f *Frame) previewEigenMode(elems []linearElement, geometric [][][]float64, conf []bool, mode []float64) {
	stress, _ := f.linearResult(elems, linearLoads{}, 0, conf, mode)
	for i, e := range f.Elems {
		e.Energy, e.Energyb = 0, 0
		clear(e.Phinge)
		if !e.IsValid {
			continue
		}
		u := elems[i].localDisplacement(mode)
		e.Energy = Dot(u, stress[i], len(u))
		if geometric != nil {
			e.Energyb = Dot(u, matrix.MatrixVector(geometric[i], u), len(u))
		}
	}
}
