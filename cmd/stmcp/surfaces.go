package main

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"

	st "github.com/yofu/st/stlib"
)

type workflowConstruction struct {
	st.GUIConstruction
	PlateSections []plateSectionRequest `json:"plate_sections,omitzero"`
	Plates        []st.BuildElement     `json:"plates,omitzero"`
	PanelGrid     *panelGrid            `json:"panel_grid,omitempty"`
}

type plateSectionRequest struct {
	ID   int       `json:"id"`
	Dead *float64  `json:"dead_load_kgf_m2,omitempty"`
	Live []float64 `json:"lload_tf_m2,omitzero"`
}

type panelGrid struct {
	Plane      string   `json:"plane"`
	Coordinate *float64 `json:"coordinate_m"`
	Section    int      `json:"section"`
	Type       string   `json:"type"`
	Expected   *int     `json:"expected_cells,omitempty"`
}

func (q workflowConstruction) hasSurfaces() bool {
	return q.PlateSections != nil || q.Plates != nil || q.PanelGrid != nil
}

// The caller owns this private GUI snapshot. Only a successful batch can
// reach the IUP revision-checked apply and create an Undo entry.
func (q workflowConstruction) onPrivateFrame(f *st.Frame) (*st.ConstructionReceipt, error) {
	base := q.GUIConstruction
	if !q.hasSurfaces() {
		return f.Construct(base, nil)
	}
	if q.Plates != nil && len(q.Plates) == 0 || q.PlateSections != nil && len(q.PlateSections) == 0 {
		return nil, fmt.Errorf("plates and plate_sections must be non-empty when supplied")
	}
	beforeProps, beforeSects, beforeNodes, beforeElems := len(f.Props), len(f.Sects), len(f.Nodes), len(f.Elems)
	r := &st.ConstructionReceipt{MemberTypes: make(map[string]int)}
	if base.Grid != nil || len(base.Nodes)+len(base.Elements)+len(base.Sections)+len(base.Properties) > 0 {
		var err error
		r, err = f.Construct(base, nil)
		if err != nil {
			return nil, err
		}
	}
	r.PlateSections = make(map[int]st.PlateLoadReceipt)
	for _, p := range q.PlateSections {
		if _, ok := r.PlateSections[p.ID]; ok {
			return nil, fmt.Errorf("duplicate plate section request %d", p.ID)
		}
		s, err := p.apply(f)
		if err != nil {
			return nil, err
		}
		r.PlateSections[p.ID] = plateLoads(s)
	}
	plates := q.Plates
	if q.PanelGrid != nil {
		generated, err := q.PanelGrid.cells(f)
		if err != nil {
			return nil, err
		}
		plates = append(slices.Clone(plates), generated...)
	}
	if len(plates) > 100000 {
		return nil, fmt.Errorf("too many plates in one construction")
	}
	r.PlateArea = make(map[int]float64)
	faces := make(map[[4]int]int)
	for id, e := range f.Elems {
		f.Maxenum = max(f.Maxenum, id)
		if !e.IsLineElem() && (e.Enods == 3 || e.Enods == 4) {
			faces[faceKey(e.Enod[:e.Enods])] = id
		}
	}
	for _, in := range plates {
		f.Maxenum = max(f.Maxenum, in.ID)
	}
	for _, in := range plates {
		et := slices.Index(st.ETYPES, in.Type)
		sec := f.Sects[in.Section]
		if et != st.SLAB && et != st.WALL || sec == nil || !sec.HasThick(0) {
			return nil, fmt.Errorf("plate requires WALL or SLAB and a plate section, got %s SECT %d", in.Type, in.Section)
		}
		if len(in.Nodes) < 3 || len(in.Nodes) > 4 || in.ID < 0 || f.Elems[in.ID] != nil {
			return nil, fmt.Errorf("invalid or occupied plate ID %d or invalid number of corners", in.ID)
		}
		ns := make([]*st.Node, len(in.Nodes))
		for j, id := range in.Nodes {
			ns[j] = f.Nodes[id]
			if ns[j] == nil {
				return nil, fmt.Errorf("plate references unknown NODE %d", id)
			}
		}
		key := faceKey(ns)
		if id, ok := faces[key]; ok {
			return nil, fmt.Errorf("face %v already occupied by ELEM %d", in.Nodes, id)
		}
		e := st.NewPlateElem(ns, sec, et)
		if ok, err := e.IsValidElem(); !ok {
			return nil, err
		}
		if len(ns) == 4 && !convexPlate(ns) {
			return nil, fmt.Errorf("plate corners must be in convex perimeter order: %v", in.Nodes)
		}
		id := in.ID
		if id == 0 {
			if f.Maxenum == math.MaxInt {
				return nil, fmt.Errorf("plate numbering overflow")
			}
			f.Maxenum++
			id = f.Maxenum
		}
		f.AddElem(id, e)
		faces[key] = id
		r.MemberTypes[in.Type]++
		r.PlateArea[sec.Num] += e.Area()
		if _, ok := r.PlateSections[sec.Num]; !ok {
			r.PlateSections[sec.Num] = plateLoads(sec)
		}
	}
	r.Total = st.ConstructionCounts{Properties: len(f.Props), Sections: len(f.Sects), Nodes: len(f.Nodes), Elements: len(f.Elems)}
	r.Added = st.ConstructionCounts{Properties: r.Total.Properties - beforeProps, Sections: r.Total.Sections - beforeSects, Nodes: r.Total.Nodes - beforeNodes, Elements: r.Total.Elements - beforeElems}
	return r, nil
}

func faceKey(ns []*st.Node) [4]int {
	var key [4]int
	for i, n := range ns {
		key[i] = n.Num
	}
	slices.Sort(key[:len(ns)])
	return key
}

func convexPlate(ns []*st.Node) bool {
	var normal []float64
	for i, n := range ns {
		a, b := ns[(i+1)%4], ns[(i+2)%4]
		next := st.Cross(st.Direction(n, a, false), st.Direction(a, b, false))
		if i == 0 {
			normal = next
		} else if st.Dot(normal, next, 3) <= 0 {
			return false
		}
	}
	return true
}

func plateLoads(s *st.Sect) st.PlateLoadReceipt {
	var dead float64
	for _, fig := range s.Figs {
		dead += fig.Weight()
	}
	return st.PlateLoadReceipt{Dead: 1000 * dead, Live: slices.Clone(s.Lload)}
}

const loadPropName = "ST_AREA_LOAD"

func loadProp(p *st.Prop) bool {
	return p != nil && p.Name == loadPropName && p.Material == nil && p.Hiju() == 1 && p.EL() == 0 && p.ES() == 0
}

func (q plateSectionRequest) apply(f *st.Frame) (*st.Sect, error) {
	if q.ID < 1 || q.Dead == nil && q.Live == nil || q.Live != nil && len(q.Live) != 3 {
		return nil, fmt.Errorf("SECT %d: supply a dead load or three LLOAD values", q.ID)
	}
	sec := f.Sects[q.ID]
	if sec == nil {
		if q.Dead == nil {
			return nil, fmt.Errorf("new load-only SECT %d requires dead_load_kgf_m2", q.ID)
		}
		sec = f.AddSect(q.ID)
		sec.Frame, sec.Original, sec.Name = f, q.ID, fmt.Sprintf("LOAD-%d", q.ID)
		f.Maxsnum = max(f.Maxsnum, q.ID)
	} else if !sec.HasThick(0) {
		return nil, fmt.Errorf("SECT %d is not a plate section", q.ID)
	}
	if q.Dead != nil {
		if *q.Dead < 0 || math.IsNaN(*q.Dead) || math.IsInf(*q.Dead, 0) {
			return nil, fmt.Errorf("invalid dead load for SECT %d", q.ID)
		}
		var fixed float64
		var layer *st.Fig
		for _, fig := range sec.Figs {
			if loadProp(fig.Prop) && fig.Name == loadPropName {
				if layer != nil {
					return nil, fmt.Errorf("SECT %d has multiple managed load layers", q.ID)
				}
				layer = fig
			} else {
				fixed += fig.Weight()
			}
		}
		extra := *q.Dead/1000 - fixed
		if extra < -1e-12 {
			return nil, fmt.Errorf("SECT %d already has %.6g kgf/m2 in its existing layers; requested total %.6g is lower", q.ID, 1000*fixed, *q.Dead)
		}
		if layer == nil && (extra > 1e-12 || len(sec.Figs) == 0) {
			var prop *st.Prop
			for _, p := range f.Props {
				if loadProp(p) && (prop == nil || p.Num < prop.Num) {
					prop = p
				}
			}
			if prop == nil {
				id := 401
				for f.Props[id] != nil {
					id++
				}
				var err error
				prop, err = f.ParseProp([]string{"PROP", strconv.Itoa(id), "PNAME", loadPropName, "HIJU", "1", "E", "0", "POI", "0"}, false)
				if err != nil {
					return nil, err
				}
			}
			layer = st.NewFig()
			for _, fig := range sec.Figs {
				layer.Num = max(layer.Num, fig.Num+1)
			}
			layer.Name, layer.Prop = loadPropName, prop
			sec.Figs = append(sec.Figs, layer)
		}
		if layer != nil {
			layer.Value["THICK"] = max(0, extra)
		}
	}
	if q.Live != nil {
		copy(sec.Lload, q.Live)
	}
	return sec, nil
}

// Follow existing horizontal/vertical edges of the requested plane. A missing
// edge cannot produce a face. The cyclic axis order gives a positive normal.
func (g panelGrid) cells(f *st.Frame) ([]st.BuildElement, error) {
	normal, ok := map[string]int{"XY": 2, "YZ": 0, "ZX": 1}[g.Plane]
	if !ok || g.Coordinate == nil || (g.Expected != nil && *g.Expected < 1) {
		return nil, fmt.Errorf("panel_grid requires plane XY, YZ or ZX, coordinate_m and a positive expected_cells if supplied")
	}
	const tol = 1e-6
	u, v := (normal+1)%3, (normal+2)%3
	next := make(map[int][2]int)
	for _, e := range f.Elems {
		if !e.IsLineElem() {
			continue
		}
		a, b := e.Enod[0], e.Enod[1]
		if math.Abs(a.Coord[normal]-*g.Coordinate) > tol || math.Abs(b.Coord[normal]-*g.Coordinate) > tol {
			continue
		}
		du, dv := b.Coord[u]-a.Coord[u], b.Coord[v]-a.Coord[v]
		axis, delta := 0, du
		switch {
		case math.Abs(dv) <= tol && math.Abs(du) > tol:
		case math.Abs(du) <= tol && math.Abs(dv) > tol:
			axis, delta = 1, dv
		default:
			continue
		}
		if delta < 0 {
			a, b = b, a
		}
		links := next[a.Num]
		if links[axis] != 0 && links[axis] != b.Num {
			return nil, fmt.Errorf("ambiguous collinear edges at NODE %d", a.Num)
		}
		links[axis] = b.Num
		next[a.Num] = links
	}
	var plates []st.BuildElement
	for _, a := range slices.Sorted(maps.Keys(next)) {
		b, d := next[a][0], next[a][1]
		c := next[b][1]
		if b != 0 && d != 0 && c != 0 && next[d][0] == c {
			plates = append(plates, st.BuildElement{Nodes: []int{a, b, c, d}, Type: g.Type, Section: g.Section})
		}
	}
	if len(plates) == 0 || g.Expected != nil && len(plates) != *g.Expected {
		return nil, fmt.Errorf("panel_grid found %d closed cells in %s at %g m; check boundary members and expected_cells", len(plates), g.Plane, *g.Coordinate)
	}
	return plates, nil
}
