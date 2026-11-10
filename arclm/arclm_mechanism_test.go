package arclm

import (
	"bytes"
	"errors"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewtonBeamSimultaneousEndYield(t *testing.T) {
	// For a fixed-fixed beam with a point load at x=1 of L=3, the
	// three-hinge mechanism gives P=2 Mp (1/1 + 1/2).
	const mp = 10 * RADIUS
	const collapse = 3 * mp
	for _, solver := range []string{"LLS", "LDLT"} {
		t.Run(solver, func(t *testing.T) {
			f := plasticTestFrame()
			f.Pivot, f.Lapch = nil, nil
			var log bytes.Buffer
			f.Output = &log
			middle := f.Nodes[1]
			for j := range middle.Conf {
				middle.Conf[j] = j != 1 && j != 5
			}
			middle.Force[0], middle.Force[1] = 0, -collapse
			right := NewNode()
			right.Num, right.Index, right.Coord[0] = 3, 2, 3
			for j := range right.Conf {
				right.Conf[j] = true
			}
			f.Nodes = append(f.Nodes, right)
			sect := f.Elems[0].Sect
			for j := range 6 {
				sect.Yield[2*j], sect.Yield[2*j+1] = 1e8, -1e8
			}
			sect.Yield[10], sect.Yield[11] = 10, -10
			e := NewElem()
			e.Num, e.Enod, e.Sect = 2, []*Node{middle, right}, sect
			f.Elems = append(f.Elems, e)

			c := newtonTestCondition(t)
			c.SetSolver(solver)
			c.SetNlap(101)
			c.SetDelta(.01)
			err := f.Arclm101(nil, c)
			var failed *PlasticConvergenceError
			if !errors.As(err, &failed) || math.Abs(failed.Target-1.01) > 1e-12 {
				tail := log.String()
				if len(tail) > 4000 {
					tail = tail[len(tail)-4000:]
				}
				t.Fatalf("failed at load factor %.12g: %v\n%s", f.plasticLoad, err, tail)
			}
			if !strings.Contains(log.String(), "ARCLM101 NEWTON REGULARIZATION:") {
				t.Fatal("simultaneous yielding did not exercise correction regularization")
			}
			nearNewton(t, "collapse load", collapse*f.plasticLoad, collapse, 1e-9)
			nearNewton(t, "vertical equilibrium", f.Nodes[0].Reaction[1]+right.Reaction[1], collapse, 1e-8)
			nearNewton(t, "left member moment at the loaded node", math.Abs(f.Elems[0].Stress[11]), mp, 1e-7)
			nearNewton(t, "right member moment at the loaded node", math.Abs(f.Elems[1].Stress[5]), mp, 1e-7)
			t.Logf("last equilibrium load %.9g, first rejected load %.9g", collapse*f.plasticLoad, collapse*failed.Target)
		})
	}
}

func TestBucklingWithUnloadedPinnedMember(t *testing.T) {
	for _, name := range []string{"beam", "axial-only brace"} {
		t.Run(name, func(t *testing.T) {
			axialOnly := name == "axial-only brace"
			f := diagnosticCantilever()
			column := f.Elems[0]
			// Subdivide the column to converge to Euler's fixed-free solution.
			const segments = 8
			f.Nodes = f.Nodes[:1]
			f.Elems = f.Elems[:0]
			for i := 1; i <= segments; i++ {
				node := NewNode()
				node.Num, node.Index, node.Coord[0] = i+1, i, float64(i)/segments
				for j := range node.Conf {
					node.Conf[j] = j != 0 && j != 1 && j != 5
				}
				f.Nodes = append(f.Nodes, node)
				elem := NewElem()
				elem.Num, elem.Sect, elem.Enod = i, column.Sect, f.Nodes[i-1:i+1]
				f.Elems = append(f.Elems, elem)
			}
			f.Nodes[segments].Force[0] = -1
			f.Pivot = nil
			// Connect the fixed base to a second fixed node with a pinned tie.
			anchor := NewNode()
			anchor.Num, anchor.Index, anchor.Coord[1] = segments+2, segments+1, 1
			for j := range anchor.Conf {
				anchor.Conf[j] = true
			}
			f.Nodes = append(f.Nodes, anchor)
			tie := NewElem()
			tie.Num, tie.Sect, tie.Enod = segments+1, column.Sect, []*Node{f.Nodes[0], anchor}
			if axialOnly {
				tie.Sect = column.Sect.Snapshot()
				clear(tie.Sect.Value[1:])
			}
			for _, j := range []int{4, 5, 10, 11} {
				tie.Bonds[j] = NewSect()
				tie.Bonds[j].Num = 1
			}
			f.Elems = append(f.Elems, tie)
			output := filepath.Join(t.TempDir(), "buckling.otp")
			err := runTorsionOrPCGTest(t, f, func() error { return f.Bclng001(output, true, 1, 1e-12, 1) })
			if err != nil {
				t.Fatal(err)
			}
			want := math.Pi * math.Pi * column.Sect.E * column.Sect.Value[2] / 4
			nearNewton(t, "Euler fixed-free load", f.EigenValue[0], want, 5e-6*want)
			nearNewton(t, "tie axial force", tie.Stress[0], 0, 0)
			nearNewton(t, "unloaded tie geometric energy", tie.Energyb, 0, 0)
			t.Logf("eigenvalue %.10g, Euler %.10g", f.EigenValue[0], want)
		})
	}
}

func TestNewtonPortalSwayCollapse(t *testing.T) {
	// Four column hinges in the sway mechanism give H = 4 Mp / h.
	const height, span, mp = 1.0, 2.0, 10 * RADIUS
	const collapse = 4 * mp / height
	for _, solver := range []string{"LLS", "LDLT"} {
		t.Run(solver, func(t *testing.T) {
			f := plasticTestFrame()
			f.Pivot, f.Lapch = nil, nil
			var log bytes.Buffer
			f.Output = &log
			f.Nodes = nil
			for i, xy := range [][2]float64{{0, 0}, {0, height}, {span, height}, {span, 0}} {
				n := NewNode()
				n.Num, n.Index = i+1, i
				copy(n.Coord, xy[:])
				for j := range n.Conf {
					n.Conf[j] = i == 0 || i == 3 || (j != 0 && j != 1 && j != 5)
				}
				if i == 1 || i == 2 {
					n.Force[0], n.Force[1] = collapse/2, -collapse/8
				}
				f.Nodes = append(f.Nodes, n)
			}
			sect := f.Elems[0].Sect
			for j := range 6 {
				sect.Yield[2*j], sect.Yield[2*j+1] = 1e8, -1e8
			}
			sect.Yield[10], sect.Yield[11] = 10, -10
			f.Elems = nil
			for i, ends := range [][2]int{{0, 1}, {1, 2}, {3, 2}} {
				e := NewElem()
				e.Num, e.Sect, e.Enod = i+1, sect, []*Node{f.Nodes[ends[0]], f.Nodes[ends[1]]}
				f.Elems = append(f.Elems, e)
			}
			c := newtonTestCondition(t)
			c.SetSolver(solver)
			c.SetNlap(201)
			c.SetDelta(.005)
			err := f.Arclm101(nil, c)
			var failed *PlasticConvergenceError
			if !errors.As(err, &failed) || math.Abs(failed.Target-1.005) > 1e-12 {
				t.Fatalf("failed at load factor %.12g: %v", f.plasticLoad, err)
			}
			nearNewton(t, "sway collapse load", collapse*f.plasticLoad, collapse, 1e-9)
			nearNewton(t, "horizontal equilibrium", f.Nodes[0].Reaction[0]+f.Nodes[3].Reaction[0], -collapse, 1e-8)
			for _, e := range f.Elems {
				for _, dof := range []int{5, 11} {
					nearNewton(t, "hinge moment", math.Abs(e.Stress[dof]), mp, 1e-7)
				}
			}
			t.Logf("last equilibrium load %.9g, first rejected load %.9g", collapse*f.plasticLoad, collapse*failed.Target)
		})
	}
}

func TestBucklingZeroAxialWithShearAtReleasedRotation(t *testing.T) {
	e := diagnosticCantilever().Elems[0]
	e.Bonds[5] = NewSect()
	e.Bonds[5].Num = 1
	e.Stress[2] = 6
	g, err := e.bucklingGeometricStiffness()
	if err != nil {
		t.Fatal(err)
	}
	// Before release, G[3][5] = Qy L / 6. A unit local displacement at
	// DOF 1 induces a released rotation of -3/(2L) at DOF 5.
	want := -e.Stress[2] / 4
	nearNewton(t, "retained shear coupling", g[3][1], want, 1e-12)
	nearNewton(t, "symmetric shear coupling", g[1][3], want, 1e-12)
	for i := range g {
		nearNewton(t, "released row", g[5][i], 0, 0)
		nearNewton(t, "released column", g[i][5], 0, 0)
	}
}
