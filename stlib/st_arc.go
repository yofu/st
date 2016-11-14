package st

import (
	"fmt"
	"math"
	"sort"

	"github.com/yofu/dxf/geometry"
)

type Arc struct {
	Frame     *Frame
	Enod      []*Node
	Center    []float64
	Radius    float64
	Direction []float64
	Start     float64
	End       float64
	Pcenter   []float64
}

func NewArc(nodes []*Node) *Arc {
	if len(nodes) < 3 {
		return nil
	}
	c, r, d, s, e, err := ArcProperty([][]float64{
		nodes[0].Coord,
		nodes[1].Coord,
		nodes[2].Coord,
	})
	if err != nil {
		return nil
	}
	return &Arc{
		Frame:     nil,
		Enod:      nodes,
		Center:    c,
		Radius:    r,
		Direction: d,
		Start:     s,
		End:       e,
	}
}

func ArcProperty(coords [][]float64) ([]float64, float64, []float64, float64, float64, error) {
	if len(coords) < 3 || OnTheSameLine(coords[0], coords[1], coords[2], 1e-4) {
		return nil, 0.0, nil, 0.0, 0.0, fmt.Errorf("ArcProperty: invalid data")
	}
	d1 := make([]float64, 3)
	d2 := make([]float64, 3)
	m1 := make([]float64, 3)
	m2 := make([]float64, 3)
	l := 0.0
	for i := 0; i < 3; i++ {
		d1[i] = coords[1][i] - coords[0][i]
		d2[i] = coords[2][i] - coords[1][i]
		m1[i] = 0.5 * (coords[0][i] + coords[1][i])
		m2[i] = 0.5 * (coords[1][i] + coords[2][i])
		l += d1[i] * d1[i] * 0.25
	}
	normal := Normalize(Cross(d1, d2))
	c1 := Normalize(Cross(d1, normal))
	c2 := Normalize(Cross(d2, normal))
	k1, _, _, err := DistLineLine(m1, c1, m2, c2)
	if err != nil {
		return nil, 0.0, nil, 0.0, 0.0, err
	}
	center := make([]float64, 3)
	vec1 := make([]float64, 3)
	vec2 := make([]float64, 3)
	h := 0.0
	for i := 0; i < 3; i++ {
		val := k1 * c1[i]
		center[i] = m1[i] + val
		h += val * val
		vec1[i] = coords[0][i] - center[i]
		vec2[i] = coords[2][i] - center[i]
	}
	vec1 = Normalize(vec1)
	vec2 = Normalize(vec2)
	ax, ay, err := geometry.ArbitraryAxis(normal)
	if err != nil {
		return nil, 0.0, nil, 0.0, 0.0, err
	}
	sum1x := 0.0
	sum1y := 0.0
	sum2x := 0.0
	sum2y := 0.0
	for i := 0; i < 3; i++ {
		sum1x += ax[i] * vec1[i]
		sum1y += ay[i] * vec1[i]
		sum2x += ax[i] * vec2[i]
		sum2y += ay[i] * vec2[i]
	}
	start := math.Atan2(sum1y, sum1x)
	if start < 0.0 {
		start += 2.0 * math.Pi
	}
	end := math.Atan2(sum2y, sum2x)
	if end < 0.0 {
		end += 2.0 * math.Pi
	}
	return center, math.Sqrt(h + l), normal, start, end, nil
}

func ArcPoints(center []float64, r float64, ex []float64, start, mid, end float64) ([][]float64, error) {
	ax, ay, err := geometry.ArbitraryAxis(ex)
	if err != nil {
		return nil, err
	}
	z := []float64{0.0, 0.0, 1.0}
	bx, by, err := geometry.ArbitraryAxis(z)
	if err != nil {
		return nil, err
	}
	c0 := make([]float64, 3)
	before := [][]float64{ax, ay, ex}
	after := [][]float64{bx, by, z}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				c0[i] += center[j] * before[j][k] * after[i][k]
			}
		}
	}
	rtn := make([][]float64, 3)
	angle := []float64{start, mid, end}
	for i := 0; i < 3; i++ {
		rtn[i] = make([]float64, 3)
		c := math.Cos(angle[i] * math.Pi / 180.0)
		s := math.Sin(angle[i] * math.Pi / 180.0)
		for j := 0; j < 3; j++ {
			rtn[i][j] = c0[j] + r*(ax[j]*c+ay[j]*s)
		}
	}
	return rtn, nil
}

func (arc *Arc) IsHidden(show *Show) bool {
	for _, en := range arc.Enod {
		if en.IsHidden(show) {
			return true
		}
	}
	return false
}

func (arc *Arc) DividingPoints(n int) [][]float64 {
	if n < 2 {
		return [][]float64{}
	}
	rtn := make([][]float64, n-1)
	dtheta := (arc.End - arc.Start) / float64(n)
	angle := dtheta
	for i := 0; i < n-1; i++ {
		rtn[i] = RotateVector(arc.Enod[0].Coord, arc.Center, arc.Direction, angle)
		angle += dtheta
	}
	return rtn
}

func (arc *Arc) DivideAtAngles(angles []float64, eps float64) ([]*Node, []*Elem, error) {
	n1 := arc.Enod[0]
	ns := make([]*Node, 0)
	els := make([]*Elem, 0)
	for _, angle := range angles {
		c := RotateVector(arc.Enod[0].Coord, arc.Center, arc.Direction, angle)
		n2, _ := arc.Frame.CoordNode(c[0], c[1], c[2], eps)
		el := arc.Frame.AddLineElem(-1, []*Node{n1, n2}, arc.Frame.DefaultSect(), NULL)
		ns = append(ns, n2)
		els = append(els, el)
		n1 = n2
	}
	el := arc.Frame.AddLineElem(-1, []*Node{n1, arc.Enod[2]}, arc.Frame.DefaultSect(), NULL)
	return ns, append(els, el), nil
}

func (arc *Arc) DivideAtLocalAxis(axis int, coords []float64, eps float64) ([]*Node, []*Elem, error) {
	var f, g func(float64) float64
	switch axis {
	default:
		return nil, nil, fmt.Errorf("unknown axis")
	case 0:
		f = math.Acos
		g = func(t float64) float64 {
			return 2*math.Pi - t
		}
	case 1:
		f = math.Asin
		g = func(t float64) float64 {
			if t < math.Pi {
				return math.Pi - t
			} else {
				return 3*math.Pi - t
			}
		}
	}
	angles := make([]float64, len(coords))
	num := 0
	end := arc.End
	if arc.Start > end {
		end += 2 * math.Pi
	}
	for _, c := range coords {
		if c < -arc.Radius || c > arc.Radius {
			continue
		}
		theta := f(c / arc.Radius)
		if arc.Start < theta && theta < end {
			angles[num] = theta - arc.Start
			num++
		} else if phi := g(theta); arc.Start < phi && phi < end {
			angles[num] = phi - arc.Start
			num++
		}
	}
	angles = angles[:num]
	sort.Float64s(angles)
	return arc.DivideAtAngles(angles, eps)
}
