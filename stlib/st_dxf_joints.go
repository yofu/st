package st

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/yofu/dxf/drawing"
	"github.com/yofu/dxf/entity"
)

type dxfXY [2]float64

func xyAdd(a, b dxfXY) dxfXY         { return dxfXY{a[0] + b[0], a[1] + b[1]} }
func xySub(a, b dxfXY) dxfXY         { return dxfXY{a[0] - b[0], a[1] - b[1]} }
func xyMul(a dxfXY, s float64) dxfXY { return dxfXY{a[0] * s, a[1] * s} }
func xyDot(a, b dxfXY) float64       { return a[0]*b[0] + a[1]*b[1] }
func xyCross(a, b dxfXY) float64     { return a[0]*b[1] - a[1]*b[0] }
func xyLength(a dxfXY) float64       { return math.Hypot(a[0], a[1]) }

const dxfJointTol = 1e-8 // model metres, not display pixels

// A pin is an input release, not a finite spring or a material-yield flag.
// Compare bond numbers: snapshots need not retain the global Pin pointer.
func dxfPinEnd(e *Elem, end int) bool {
	for _, j := range []int{4, 5} {
		if b := e.Bonds[6*end+j]; b != nil && b.Num == 1 {
			return true
		}
	}
	return false
}
func dxfRigidEnd(e *Elem, end int) bool {
	for _, j := range []int{4, 5} {
		if b := e.Bonds[6*end+j]; b != nil && b.Num != 0 {
			return false
		}
	}
	return true
}

type dxfMemberStroke struct {
	id, etype int
	sect      *Sect
	a, b      dxfXY
	u, n      dxfXY
	length    float64
	lo, hi    float64 // signed transverse offsets, metres
	nodes     map[int]*Node
	pin       [2]bool
	rigid     [2]bool
	ends      [2][2]dxfXY // end, positive/negative rail
	clips     [][]dxfXY
}
type dxfJointRef struct {
	stroke int
	end    int
	at     float64
}

func dxfSectionShape(s *Sect) ([][]float64, float64) {
	if s == nil || s.Allow == nil {
		return nil, 0
	}
	var shape Shape
	switch a := s.Allow.(type) {
	case *SColumn:
		shape = a.Shape
	case *SGirder:
		shape = a.Shape
	case *SBrace:
		shape = a.Shape
	case *RCColumn:
		if a.CShape != nil {
			return a.CShape.Vertices(), 0
		}
	case *RCGirder:
		if a.CShape != nil {
			return a.CShape.Vertices(), 0
		}
	}
	if shape != nil {
		switch sh := shape.(type) {
		case CPIPE:
			return nil, sh.D * .005
		case SAREA, THICK:
			return nil, 0
		}
		return shape.Vertices(), 0
	}
	return nil, 0
}

func dxfTransverse(e *Elem, x, y int, n dxfXY) (float64, float64) {
	verts, radius := dxfSectionShape(e.Sect)
	s := n[0]*e.Strong[x] + n[1]*e.Strong[y]
	w := n[0]*e.Weak[x] + n[1]*e.Weak[y]
	if radius > 0 {
		h := radius * math.Hypot(s, w)
		return -h, h
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range verts {
		if v != nil {
			z := .01 * (v[0]*s + v[1]*w)
			lo = math.Min(lo, z)
			hi = math.Max(hi, z)
		}
	}
	if len(verts) > 0 && finiteDXF(lo) && finiteDXF(hi) {
		return lo, hi
	}
	return 0, 0 // no fabricated width for area-only or undefined sections
}

// prepareDXFStrokes joins only geometrically straight, equally shaped members
// whose common input ends are both rigid. Node IDs, not projected crossings,
// identify actual connectivity; internal junction nodes survive aggregation.
func (f *Frame) prepareDXFStrokes(axis int, min, max float64, x, y int) []dxfMemberStroke {
	var members []*Elem
	for _, e := range f.Elems {
		if !e.IsLineElem() || e.Etype == WBRACE || e.Etype == SBRACE {
			continue
		}
		inside := true
		for _, n := range e.Enod {
			inside = inside && n.Coord[axis] >= min && n.Coord[axis] <= max
		}
		if inside && math.Hypot(e.Enod[1].Coord[x]-e.Enod[0].Coord[x], e.Enod[1].Coord[y]-e.Enod[0].Coord[y]) > dxfJointTol {
			members = append(members, e)
		}
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Num < members[j].Num })
	parent := make([]int, len(members))
	for i := range parent {
		parent[i] = i
	}
	var root func(int) int
	root = func(i int) int {
		if parent[i] != i {
			parent[i] = root(parent[i])
		}
		return parent[i]
	}
	byNode := map[int][][2]int{}
	for i, e := range members {
		for end, n := range e.Enod {
			byNode[n.Num] = append(byNode[n.Num], [2]int{i, end})
		}
	}
	for _, refs := range byNode {
		matches := map[int]int{}
		for _, ref := range refs {
			e, end := members[ref[0]], ref[1]
			if !dxfRigidEnd(e, end) {
				continue
			}
			match := -1
			ambiguous := false
			v := []float64{e.Enod[1-end].Coord[0] - e.Enod[end].Coord[0], e.Enod[1-end].Coord[1] - e.Enod[end].Coord[1], e.Enod[1-end].Coord[2] - e.Enod[end].Coord[2]}
			lenV := e.Length()
			for _, other := range refs {
				if other[0] == ref[0] {
					continue
				}
				q, qe := members[other[0]], other[1]
				if q.Sect.Num != e.Sect.Num || q.Etype != e.Etype || !dxfRigidEnd(q, qe) {
					continue
				}
				dot := 0.
				for j := 0; j < 3; j++ {
					dot += v[j] * (q.Enod[1-qe].Coord[j] - q.Enod[qe].Coord[j])
				}
				if math.Abs(dot/(lenV*q.Length())+1) > 1e-9 {
					continue
				}
				u := dxfXY{v[x], v[y]}
				u = xyMul(u, 1/xyLength(u))
				n := dxfXY{-u[1], u[0]}
				el, eh := dxfTransverse(e, x, y, n)
				ql, qh := dxfTransverse(q, x, y, n)
				if math.Abs(el-ql) > dxfJointTol || math.Abs(eh-qh) > dxfJointTol {
					continue
				}
				if match != -1 {
					ambiguous = true
					break
				}
				match = other[0]
			}
			if match >= 0 && !ambiguous {
				matches[ref[0]] = match
			}
		}
		for i, j := range matches {
			if other, ok := matches[j]; !ok || other != i {
				continue
			}
			a, b := root(i), root(j)
			if a > b {
				a, b = b, a
			}
			parent[b] = a
		}
	}

	groups := map[int][]*Elem{}
	for i, e := range members {
		groups[root(i)] = append(groups[root(i)], e)
	}
	var keys []int
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var strokes []dxfMemberStroke
	for _, key := range keys {
		es := groups[key]
		first := es[0]
		nodes := map[int]*Node{}
		for _, e := range es {
			for _, n := range e.Enod {
				nodes[n.Num] = n
			}
		}
		var start, end *Node
		for _, n := range nodes {
			if start == nil || n.Coord[x] < start.Coord[x] || n.Coord[x] == start.Coord[x] && n.Coord[y] < start.Coord[y] {
				start = n
			}
			if end == nil || n.Coord[x] > end.Coord[x] || n.Coord[x] == end.Coord[x] && n.Coord[y] > end.Coord[y] {
				end = n
			}
		}
		a, b := dxfXY{start.Coord[x], start.Coord[y]}, dxfXY{end.Coord[x], end.Coord[y]}
		length := xyLength(xySub(b, a))
		u := xyMul(xySub(b, a), 1/length)
		n := dxfXY{-u[1], u[0]}
		lo, hi := dxfTransverse(first, x, y, n)
		s := dxfMemberStroke{id: first.Num, etype: first.Etype, sect: first.Sect, a: a, b: b, u: u, n: n, length: length, lo: lo, hi: hi, nodes: nodes}
		for _, e := range es {
			for i, node := range e.Enod {
				if node.Num == start.Num {
					s.pin[0], s.rigid[0] = dxfPinEnd(e, i), dxfRigidEnd(e, i)
				}
				if node.Num == end.Num {
					s.pin[1], s.rigid[1] = dxfPinEnd(e, i), dxfRigidEnd(e, i)
				}
			}
		}
		s.ends = [2][2]dxfXY{{xyAdd(a, xyMul(n, hi)), xyAdd(a, xyMul(n, lo))}, {xyAdd(b, xyMul(n, hi)), xyAdd(b, xyMul(n, lo))}}
		strokes = append(strokes, s)
	}
	return strokes
}

func dxfRayExit(origin, direction dxfXY, poly []dxfXY) float64 {
	exit := 0.
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		edge := xySub(b, a)
		den := xyCross(direction, edge)
		if math.Abs(den) < 1e-12 {
			continue
		}
		t := xyCross(xySub(a, origin), edge) / den
		u := xyCross(xySub(a, origin), direction) / den
		if t >= -dxfJointTol && u >= -dxfJointTol && u <= 1+dxfJointTol {
			exit = math.Max(exit, t)
		}
	}
	return exit
}
func strokePolygon(s *dxfMemberStroke) []dxfXY {
	return []dxfXY{s.ends[0][0], s.ends[1][0], s.ends[1][1], s.ends[0][1]}
}
func dxfRank(s dxfMemberStroke) int {
	switch s.etype {
	case COLUMN:
		return 3
	case GIRDER:
		return 2
	default:
		return 1
	}
}
func dxfRefRay(s *dxfMemberStroke, ref dxfJointRef) dxfXY {
	if ref.end == 1 {
		return xyMul(s.u, -1)
	}
	return s.u
}

func dxfMiter(a *dxfMemberStroke, ar dxfJointRef, b *dxfMemberStroke, br dxfJointRef) bool {
	if ar.end < 0 || br.end < 0 || !a.rigid[ar.end] || !b.rigid[br.end] || a.etype != b.etype || a.hi-a.lo < dxfJointTol || b.hi-b.lo < dxfJointTol {
		return false
	}
	ra, rb := dxfRefRay(a, ar), dxfRefRay(b, br)
	den := xyCross(ra, rb)
	if math.Abs(den) < .05 {
		return false
	}
	var ae, be [2]dxfXY
	for i, p := range a.ends[ar.end] {
		best := -1
		distance := math.Inf(1)
		var point dxfXY
		for j, q := range b.ends[br.end] {
			// Corresponding outer/inner rails have opposite normals measured
			// from the junction into each member.
			asign := i
			if ar.end == 1 {
				asign = 1 - i
			}
			bsign := j
			if br.end == 1 {
				bsign = 1 - j
			}
			if asign == bsign {
				continue
			}
			t := xyCross(xySub(q, p), rb) / den
			v := xyAdd(p, xyMul(ra, t))
			if math.Abs(t) < distance {
				best = j
				distance = math.Abs(t)
				point = v
			}
		}
		limit := 4 * math.Max(a.hi-a.lo, b.hi-b.lo)
		if best < 0 || distance > limit || xyLength(xySub(point, b.ends[br.end][best])) > limit {
			return false
		}
		ae[i], be[best] = point, point
	}
	a.ends[ar.end], b.ends[br.end] = ae, be
	return true
}

// A cut member (e.g. a column in a plan) supplies an outside envelope only
// to members actually sharing one of its input nodes.
func (f *Frame) dxfCutFaces(axis int, cut float64, x, y int) map[int][][]dxfXY {
	faces := map[int][][]dxfXY{}
	for _, e := range f.Fence(axis, cut, false) {
		verts, radius := dxfSectionShape(e.Sect)
		if len(verts) == 0 && radius == 0 {
			continue
		}
		lo, hi := dxfXY{math.Inf(1), math.Inf(1)}, dxfXY{math.Inf(-1), math.Inf(-1)}
		if radius > 0 {
			lo, hi = dxfXY{-radius, -radius}, dxfXY{radius, radius}
		} else {
			for _, v := range verts {
				if v != nil {
					for j := 0; j < 2; j++ {
						lo[j] = math.Min(lo[j], v[j]*.01)
						hi[j] = math.Max(hi[j], v[j]*.01)
					}
				}
			}
		}
		angle := e.Cang
		if axis != 2 {
			angle -= math.Pi / 2
		}
		s, w, err := PrincipalAxis([]float64{0, 0, 1}, angle)
		if err != nil {
			continue
		}
		for _, node := range e.Enod {
			center := dxfXY{node.Coord[x], node.Coord[y]}
			var poly []dxfXY
			if radius > 0 {
				for i := 0; i < 64; i++ {
					a := float64(i) * 2 * math.Pi / 64
					poly = append(poly, xyAdd(center, dxfXY{radius * math.Cos(a), radius * math.Sin(a)}))
				}
			} else {
				for _, q := range []dxfXY{{lo[0], lo[1]}, {hi[0], lo[1]}, {hi[0], hi[1]}, {lo[0], hi[1]}} {
					poly = append(poly, xyAdd(center, dxfXY{q[0]*s[0] + q[1]*w[0], q[0]*s[1] + q[1]*w[1]}))
				}
			}
			faces[node.Num] = append(faces[node.Num], poly)
		}
	}
	return faces
}

func (f *Frame) jointDXFStrokes(strokes []dxfMemberStroke, axis int, cut float64, x, y int, gap float64) {
	byNode := map[int][]dxfJointRef{}
	for i := range strokes {
		s := &strokes[i]
		for id, n := range s.nodes {
			at := xyDot(xySub(dxfXY{n.Coord[x], n.Coord[y]}, s.a), s.u)
			end := -1
			if at < dxfJointTol {
				end = 0
			}
			if at > s.length-dxfJointTol {
				end = 1
			}
			byNode[id] = append(byNode[id], dxfJointRef{i, end, at})
		}
	}
	faces := f.dxfCutFaces(axis, cut, x, y)
	var ids []int
	for id := range byNode {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		refs := byNode[id]
		external := faces[id]
		if len(external) == 0 && len(refs) == 2 && dxfMiter(&strokes[refs[0].stroke], refs[0], &strokes[refs[1].stroke], refs[1]) {
			continue
		}
		host := refs[0]
		for _, ref := range refs[1:] {
			s, h := strokes[ref.stroke], strokes[host.stroke]
			better := ref.end < 0 && host.end >= 0
			if (ref.end < 0) == (host.end < 0) {
				sp, hp := ref.end >= 0 && s.pin[ref.end], host.end >= 0 && h.pin[host.end]
				better = !sp && hp || sp == hp && (dxfRank(s) > dxfRank(h) || dxfRank(s) == dxfRank(h) && (s.hi-s.lo > h.hi-h.lo+dxfJointTol || math.Abs(s.hi-s.lo-h.hi+h.lo) < dxfJointTol && s.id < h.id))
			}
			if better {
				host = ref
			}
		}
		h := &strokes[host.stroke]
		if len(external) == 0 && host.end >= 0 && h.rigid[host.end] {
			out := xyMul(dxfRefRay(h, host), -1)
			extend := 0.
			for _, ref := range refs {
				if ref.stroke == host.stroke {
					continue
				}
				s := &strokes[ref.stroke]
				if math.Abs(xyCross(s.u, h.u)) < 1e-8 {
					continue
				}
				extend = math.Max(extend, math.Max(xyDot(xyMul(s.n, s.hi), out), xyDot(xyMul(s.n, s.lo), out)))
			}
			for j := 0; j < 2; j++ {
				h.ends[host.end][j] = xyAdd(h.ends[host.end][j], xyMul(out, extend))
			}
		}
		for _, ref := range refs {
			s := &strokes[ref.stroke]
			obstacles := external
			if len(obstacles) == 0 && ref.stroke != host.stroke {
				obstacles = [][]dxfXY{strokePolygon(h)}
			}
			if ref.end >= 0 && s.pin[ref.end] {
				ray := dxfRefRay(s, ref)
				center := s.a
				if ref.end == 1 {
					center = s.b
				}
				retreat := 0.
				for _, poly := range obstacles {
					// A flat pin end must also clear tips between its two rails
					// (e.g. a column rotated 45 degrees).
					for _, p := range poly {
						transverse := xyDot(xySub(p, center), s.n)
						if transverse >= s.lo-dxfJointTol && transverse <= s.hi+dxfJointTol {
							retreat = math.Max(retreat, xyDot(xySub(p, center), ray))
						}
					}
					for _, p := range s.ends[ref.end] {
						retreat = math.Max(retreat, dxfRayExit(p, ray, poly))
					}
				}
				for j := 0; j < 2; j++ {
					s.ends[ref.end][j] = xyAdd(s.ends[ref.end][j], xyMul(ray, retreat+gap))
				}
			} else {
				s.clips = append(s.clips, obstacles...)
			}
		}
	}
}

// dxfPolygonRemainder traces the exposed boundary of subject minus the union
// of clips. It preserves the actual sloping/curved envelope instead of joining
// only the two rail intersections with a chord through the host section.
func dxfPolygonRemainder(subject []dxfXY, clips [][]dxfXY) [][]dxfXY {
	ccw := func(p []dxfXY) []dxfXY {
		q := append([]dxfXY(nil), p...)
		area := 0.
		for i, a := range q {
			area += xyCross(a, q[(i+1)%len(q)])
		}
		if area < 0 {
			for i, j := 0, len(q)-1; i < j; i, j = i+1, j-1 {
				q[i], q[j] = q[j], q[i]
			}
		}
		return q
	}
	subject = ccw(subject)
	polygons := [][]dxfXY{subject}
	for _, p := range clips {
		area := 0.
		for i, a := range p {
			area += xyCross(a, p[(i+1)%len(p)])
		}
		if math.Abs(area) > dxfJointTol*dxfJointTol {
			clip := ccw(p)
			if dxfConvexOverlapArea(subject, clip) > dxfJointTol*dxfJointTol {
				polygons = append(polygons, clip)
			}
		}
	}
	type segment struct{ a, b dxfXY }
	var segments []segment
	for pi, poly := range polygons {
		for i, a := range poly {
			b := poly[(i+1)%len(poly)]
			v := xySub(b, a)
			length := xyLength(v)
			if length < dxfJointTol {
				continue
			}
			ts := []float64{0, 1}
			for qi, other := range polygons {
				if qi == pi {
					continue
				}
				for j, c := range other {
					d := other[(j+1)%len(other)]
					w := xySub(d, c)
					den := xyCross(v, w)
					if math.Abs(den) > 1e-12*length*xyLength(w) {
						t := xyCross(xySub(c, a), w) / den
						u := xyCross(xySub(c, a), v) / den
						if t > 0 && t < 1 && u >= -1e-9 && u <= 1+1e-9 {
							ts = append(ts, t)
						}
					} else if math.Abs(xyCross(xySub(c, a), v)) <= dxfJointTol*length {
						for _, p := range []dxfXY{c, d} {
							t := xyDot(xySub(p, a), v) / (length * length)
							if t > 0 && t < 1 {
								ts = append(ts, t)
							}
						}
					}
				}
			}
			sort.Float64s(ts)
			for j := 1; j < len(ts); j++ {
				if (ts[j]-ts[j-1])*length < dxfJointTol {
					continue
				}
				mid := xyAdd(a, xyMul(v, (ts[j]+ts[j-1])*.5))
				keep := true
				if pi == 0 {
					for _, clip := range polygons[1:] {
						if dxfPointLocation(mid, clip) >= 0 {
							keep = false
							break
						}
					}
				} else {
					keep = dxfPointLocation(mid, subject) == 1
					if keep {
						for qi, clip := range polygons[1:] {
							qi++
							if qi == pi {
								continue
							}
							loc := dxfPointLocation(mid, clip)
							if loc == 1 {
								keep = false
								break
							}
							if loc == 0 {
								outward := xyAdd(mid, xyMul(dxfXY{v[1], -v[0]}, 4*dxfJointTol/length))
								if dxfPointLocation(outward, clip) == 1 || qi < pi {
									keep = false
									break
								}
							}
						}
					}
				}
				if keep {
					p, q := xyAdd(a, xyMul(v, ts[j-1])), xyAdd(a, xyMul(v, ts[j]))
					if pi > 0 {
						p, q = q, p
					}
					segments = append(segments, segment{p, q})
				}
			}
		}
	}
	type key [2]int64
	keyOf := func(p dxfXY) key {
		return key{int64(math.Round(p[0] / dxfJointTol)), int64(math.Round(p[1] / dxfJointTol))}
	}
	starts := map[key][]int{}
	for i, e := range segments {
		starts[keyOf(e.a)] = append(starts[keyOf(e.a)], i)
	}
	used := make([]bool, len(segments))
	var result [][]dxfXY
	for i, e := range segments {
		if used[i] {
			continue
		}
		path := []dxfXY{e.a}
		current := i
		for {
			used[current] = true
			end := segments[current].b
			if xyLength(xySub(end, path[0])) < 2*dxfJointTol {
				break
			}
			path = append(path, end)
			next := -1
			k := keyOf(end)
			for dx := int64(-1); dx <= 1; dx++ {
				for dy := int64(-1); dy <= 1; dy++ {
					for _, candidate := range starts[key{k[0] + dx, k[1] + dy}] {
						if !used[candidate] && xyLength(xySub(segments[candidate].a, end)) < 2*dxfJointTol {
							next = candidate
							break
						}
					}
				}
			}
			if next < 0 {
				break
			}
			current = next
		}
		if len(path) >= 3 {
			result = append(result, path)
		}
	}
	return result
}

// Clip a convex member footprint against a convex host envelope. Touching
// outlines have no area and must not remove an otherwise valid end cap.
func dxfConvexOverlapArea(subject, clip []dxfXY) float64 {
	out := append([]dxfXY(nil), subject...)
	area := 0.
	for i, a := range clip {
		area += xyCross(a, clip[(i+1)%len(clip)])
	}
	orientation := 1.
	if area < 0 {
		orientation = -1
	}
	for i, a := range clip {
		b := clip[(i+1)%len(clip)]
		edge := xySub(b, a)
		input := out
		out = nil
		if len(input) == 0 {
			return 0
		}
		prev := input[len(input)-1]
		pd := orientation * xyCross(edge, xySub(prev, a))
		for _, cur := range input {
			cd := orientation * xyCross(edge, xySub(cur, a))
			if (cd >= 0) != (pd >= 0) {
				out = append(out, xyAdd(prev, xyMul(xySub(cur, prev), pd/(pd-cd))))
			}
			if cd >= 0 {
				out = append(out, cur)
			}
			prev, pd = cur, cd
		}
	}
	area = 0.
	for i, a := range out {
		area += xyCross(a, out[(i+1)%len(out)])
	}
	return math.Abs(area) * .5
}

// -1 outside, 0 boundary, +1 inside.
func dxfPointLocation(p dxfXY, poly []dxfXY) int {
	inside := false
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		v := xySub(b, a)
		w := xySub(p, a)
		length := xyLength(v)
		if length > dxfJointTol && math.Abs(xyCross(v, w)) <= dxfJointTol*length && xyDot(w, v) >= -dxfJointTol*length && xyDot(xySub(p, b), v) <= dxfJointTol*length {
			return 0
		}
		if (a[1] > p[1]) != (b[1] > p[1]) && p[0] < a[0]+(p[1]-a[1])*(b[0]-a[0])/(b[1]-a[1]) {
			inside = !inside
		}
	}
	if inside {
		return 1
	}
	return -1
}

// Area-only sections have no drawable width, but their centerlines still
// terminate at the connected host face.
func dxfLineRemainder(a, b dxfXY, clips [][]dxfXY) [][]dxfXY {
	v := xySub(b, a)
	ts := []float64{0, 1}
	for _, poly := range clips {
		for i, p := range poly {
			edge := xySub(poly[(i+1)%len(poly)], p)
			den := xyCross(v, edge)
			if math.Abs(den) < 1e-12 {
				continue
			}
			t := xyCross(xySub(p, a), edge) / den
			u := xyCross(xySub(p, a), v) / den
			if t > 0 && t < 1 && u >= -1e-9 && u <= 1+1e-9 {
				ts = append(ts, t)
			}
		}
	}
	sort.Float64s(ts)
	var result [][]dxfXY
	for i := 1; i < len(ts); i++ {
		if (ts[i]-ts[i-1])*xyLength(v) < dxfJointTol {
			continue
		}
		mid := xyAdd(a, xyMul(v, (ts[i]+ts[i-1])*.5))
		keep := true
		for _, poly := range clips {
			if dxfPointLocation(mid, poly) == 1 {
				keep = false
				break
			}
		}
		if keep {
			p, q := xyAdd(a, xyMul(v, ts[i-1])), xyAdd(a, xyMul(v, ts[i]))
			result = append(result, []dxfXY{p, q})
		}
	}
	return result
}

func dxfStrokePieces(s dxfMemberStroke) [][]dxfXY {
	if xyDot(xySub(s.ends[1][0], s.ends[0][0]), s.u) <= dxfJointTol || xyDot(xySub(s.ends[1][1], s.ends[0][1]), s.u) <= dxfJointTol {
		return nil
	}
	poly := strokePolygon(&s)
	if s.hi-s.lo <= dxfJointTol {
		return dxfLineRemainder(poly[0], poly[1], s.clips)
	}
	return dxfPolygonRemainder(poly, s.clips)
}

func (f *Frame) drawDXFJoinedMembers(d *drawing.Drawing, axis int, min, max float64, side int, x, y int, scale, textheight float64, setLayer, setTextLayer func(int)) error {
	strokes := f.prepareDXFStrokes(axis, min, max, x, y)
	f.jointDXFStrokes(strokes, axis, []float64{min, max}[side], x, y, .2*textheight/scale)
	for _, s := range strokes {
		pieces := dxfStrokePieces(s)
		if len(pieces) == 0 {
			continue
		}
		setLayer(s.etype)
		for _, p := range pieces {
			if s.hi-s.lo > dxfJointTol {
				var points [][]float64
				for _, v := range p {
					points = append(points, []float64{v[0] * scale, v[1] * scale})
				}
				if _, err := d.LwPolyline(true, points...); err != nil {
					return err
				}
			} else {
				if _, err := d.Line(p[0][0]*scale, p[0][1]*scale, 0, p[1][0]*scale, p[1][1]*scale, 0); err != nil {
					return err
				}
			}
		}
		name := strings.TrimSpace(strings.Split(s.sect.Name, ":")[0])
		if s.sect.Allow != nil && s.sect.Allow.Name() != "" {
			name = s.sect.Allow.Name()
		}
		if name == "" {
			name = fmt.Sprintf("S%d", s.sect.Num)
		}
		// Use the longest exposed outer rail, with text outside the member.
		mid := xyMul(xyAdd(s.a, s.b), .5)
		longest := -1.
		bestOffset := math.Inf(-1)
		for _, p := range pieces {
			for i, a := range p {
				b := p[(i+1)%len(p)]
				delta := xySub(b, a)
				if math.Abs(xyCross(delta, s.u)) > dxfJointTol {
					continue
				}
				length := xyLength(delta)
				offset := xyDot(xySub(a, s.a), s.n)
				if offset > bestOffset+dxfJointTol || math.Abs(offset-bestOffset) < dxfJointTol && length > longest {
					longest, bestOffset = length, offset
					mid = xyMul(xyAdd(a, b), .5)
				}
			}
		}
		mid = xyAdd(mid, xyMul(s.n, .5*textheight/scale))

		setTextLayer(s.etype)
		t, err := d.Text(name, mid[0]*scale, mid[1]*scale, 0, textheight)
		if err != nil {
			return err
		}
		t.Anchor(entity.CENTER_BOTTOM)
		t.Rotation = math.Atan2(s.u[1], s.u[0]) * 180 / math.Pi
	}
	return nil
}
