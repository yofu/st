package arclm

import (
	"io"
	"math"
	"testing"
)

// Three axial bars give Ke={{101,1},{1,2}} and, after the horizontal
// bar yields, Kp={{1,1},{1,2}}. For f=(.099,-.003), the elastic Newton
// direction at first yield INCREASES the residual for every line-search
// length. The directional plastic tangent instead gives the exact solution.
func TestNewtonDirectionalBoundaryDescent(t *testing.T) {
	for _, solver := range []string{"LLS", "LDLT"} {
		t.Run(solver, func(t *testing.T) {
			f := NewFrame()
			f.Output, f.Pivot, f.Lapch = io.Discard, nil, nil
			coords := [][2]float64{{-1, 0}, {-1 / math.Sqrt2, -1 / math.Sqrt2}, {0, -1}, {0, 0}}
			for i, xy := range coords {
				n := NewNode()
				n.Num, n.Index = i+1, i
				n.Coord[0], n.Coord[1] = xy[0], xy[1]
				for j := range n.Conf {
					n.Conf[j] = i != 3 || j >= 2
				}
				f.Nodes = append(f.Nodes, n)
			}
			f.Nodes[3].Force[0], f.Nodes[3].Force[1] = .099, -.003
			for i, stiffness := range []float64{100, 2, 1} {
				s := NewSect()
				s.Num, s.E, s.Exp, s.Exq = i+1, stiffness, 2, 2
				s.Value[0] = 1
				for j := 0; j < 6; j++ {
					s.Yield[2*j], s.Yield[2*j+1] = 1000, -1000
				}
				if i == 0 {
					s.Yield[0], s.Yield[1] = .1/RADIUS, -.1/RADIUS
				}
				e := NewElem()
				e.Num, e.Sect, e.Enod = i+1, s, []*Node{f.Nodes[i], f.Nodes[3]}
				f.Sects = append(f.Sects, s)
				f.Elems = append(f.Elems, e)
			}
			for i, tc := range []struct{ factor, ux, uy float64 }{
				{1, .001, -.002},
				{1.01, .00301, -.00302},
				{1, .003, -.003},
				{1.01, .00301, -.00302},
				{1.02, .00502, -.00404},
			} {
				c := newtonTestCondition(t)
				c.SetInit(i == 0)
				c.SetSolver(solver)
				c.SetDelta(tc.factor - f.plasticLoad)
				if err := f.Arclm101(nil, c); err != nil {
					t.Fatalf("target=%g: %v", tc.factor, err)
				}
				nearNewton(t, "analytic Ux", f.Nodes[3].Disp[0], tc.ux, 1e-8)
				nearNewton(t, "analytic Uy", f.Nodes[3].Disp[1], tc.uy, 1e-8)
			}
		})
	}
}
