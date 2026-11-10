package arclm

import "fmt"

// condenseConnections eliminates the member-side rotations one at a time.
// Each elimination uses the current stiffness, including earlier eliminations.
// k is a newly assembled local stiffness and q, when provided, is a copy of CMQ.
func (elem *Elem) condenseConnections(k [][]float64, q []float64) error {
	return elem.condenseConnectionOperators(k, q, nil)
}

// g is projected through the elastic displacement recovery, one rotation at
// a time. A geometric operator alone need not have a usable release pivot.
func (elem *Elem) condenseConnectionOperators(k [][]float64, q []float64, g [][]float64) error {
	for j, bond := range elem.Bonds {
		if bond == nil {
			return fmt.Errorf("ELEM %d: missing connection at DOF %d", elem.Num, j)
		}
		if bond.Num == 0 {
			continue
		}
		if j%6 < 3 {
			return fmt.Errorf("ELEM %d: translational connection release at DOF %d is not supported", elem.Num, j)
		}
		spring := 0.0
		if bond.Num != 1 {
			if len(bond.Value) < 3 {
				return fmt.Errorf("ELEM %d: missing spring stiffness at DOF %d", elem.Num, j)
			}
			spring = bond.Value[j%6-3]
			if spring < 0 || !finitePlastic(spring) {
				return fmt.Errorf("ELEM %d: invalid spring stiffness at DOF %d", elem.Num, j)
			}
		}
		var col [12]float64
		var force float64
		if q != nil {
			force = q[j]
		}
		den := k[j][j] + spring
		if den == 0 {
			empty := force == 0
			for i := range col {
				empty = empty && k[i][j] == 0 && (g == nil || g[i][j] == 0)
			}
			if empty {
				continue
			}
		}
		if den <= 0 || !finitePlastic(den) {
			return fmt.Errorf("ELEM %d: invalid connection pivot at DOF %d: %g", elem.Num, j, den)
		}
		for i := range col {
			col[i] = k[i][j]
		}
		if g != nil {
			// For P = I - e_j k[:,j]^T / den, compute P^T g P.
			var gcol [12]float64
			for i := range gcol {
				gcol[i] = g[i][j]
			}
			for i := range gcol {
				ci := col[i] / den
				for m := i; m < len(gcol); m++ {
					cm := col[m] / den
					g[i][m] += ci*(gcol[j]*cm-gcol[m]) - gcol[i]*cm
					g[m][i] = g[i][m]
				}
			}
		}
		for i, v := range col {
			factor := v / den
			if q != nil {
				q[i] -= factor * force
			}
			for m := i; m < len(col); m++ {
				k[i][m] -= factor * col[m]
				k[m][i] = k[i][m]
			}
		}
		if spring == 0 {
			// A released rotation transmits exactly zero moment.
			clear(k[j])
			for i := range col {
				k[i][j] = 0
			}
			if g != nil {
				clear(g[j])
				for i := range col {
					g[i][j] = 0
				}
			}
			if q != nil {
				q[j] = 0
			}
		}
	}
	return nil
}

func (elem *Elem) bucklingGeometricStiffness() ([][]float64, error) {
	g, err := elem.GeoStiffMatrix()
	if err != nil {
		return nil, err
	}
	// Use the same elastic release kinematics for axial, bending and shear
	// contributions. Condensing KG by itself would divide by a geometric
	// pivot that can vanish as the axial force tends to zero.
	released := false
	for _, bond := range elem.Bonds {
		released = released || bond.Num != 0
	}
	if !released {
		return g, nil
	}
	for _, row := range g {
		for _, v := range row {
			if v != 0 {
				k, err := elem.StiffMatrix()
				if err != nil {
					return nil, err
				}
				if err := elem.condenseConnectionOperators(k, nil, g); err != nil {
					return nil, err
				}
				return g, nil
			}
		}
	}
	return g, nil
}
