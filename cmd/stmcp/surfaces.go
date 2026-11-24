package main

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	st "github.com/yofu/st/stlib"
)

type workflowConstruction struct {
	st.GUIConstruction
	UpdateSections []updateSectionRequest `json:"update_sections,omitzero"`
	UpdateNodes    []st.BuildNode         `json:"update_nodes,omitzero"`
	PlateSections  []plateSectionRequest  `json:"plate_sections,omitzero"`
	Plates         []st.BuildElement      `json:"plates,omitzero"`
	PanelGrid      *panelGrid             `json:"panel_grid,omitempty"`
}

type updateSectionRequest struct {
	ID         int       `json:"id"`
	Name       string    `json:"name,omitempty"`
	Property   *int      `json:"property,omitempty"`
	Shape      string    `json:"shape"`
	Dimensions []float64 `json:"dimensions_mm"`
	SyncLST    *bool     `json:"sync_lst,omitempty"`
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

func (q workflowConstruction) usesSnapshot() bool {
	return q.hasSurfaces() || len(q.UpdateSections) > 0 || len(q.UpdateNodes) > 0
}

// The caller owns this private GUI snapshot. Only a successful batch can
// reach the IUP revision-checked apply and create an Undo entry.
func (q workflowConstruction) onPrivateFrame(f *st.Frame) (*st.ConstructionReceipt, error) {
	r, _, err := q.onPrivateFrameWithLST(f, nil, "")
	return r, err
}

func (q workflowConstruction) onPrivateFrameWithLST(f *st.Frame, lst []byte, lstPath string) (*st.ConstructionReceipt, []byte, error) {
	base := q.GUIConstruction
	if !q.usesSnapshot() {
		r, err := f.Construct(base, nil)
		return r, lst, err
	}
	if q.Plates != nil && len(q.Plates) == 0 || q.PlateSections != nil && len(q.PlateSections) == 0 {
		return nil, nil, fmt.Errorf("plates and plate_sections must be non-empty when supplied")
	}
	if len(q.UpdateSections) > 1000 {
		return nil, nil, fmt.Errorf("too many update_sections in one construction")
	}
	if len(q.UpdateNodes) > 100000 {
		return nil, nil, fmt.Errorf("too many update_nodes in one construction")
	}
	beforeProps, beforeSects, beforeNodes, beforeElems := len(f.Props), len(f.Sects), len(f.Nodes), len(f.Elems)
	r := &st.ConstructionReceipt{
		MemberTypes: make(map[string]int),
		Sections:    make(map[int][]st.ConstructedFig),
		Properties:  make(map[int]string),
	}
	if base.Grid != nil || len(base.Nodes)+len(base.Elements)+len(base.Sections)+len(base.Properties) > 0 {
		var err error
		r, err = f.Construct(base, nil)
		if err != nil {
			return nil, nil, err
		}
		if r.Sections == nil {
			r.Sections = make(map[int][]st.ConstructedFig)
		}
		if r.Properties == nil {
			r.Properties = make(map[int]string)
		}
	}
	updatedLST := lst
	if len(q.UpdateSections) > 0 {
		var err error
		updatedLST, err = applySectionUpdates(f, q.UpdateSections, lst, lstPath, r)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(q.PlateSections) > 0 || len(q.Plates) > 0 || q.PanelGrid != nil {
		r.PlateSections = make(map[int]st.PlateLoadReceipt)
	}
	for _, p := range q.PlateSections {
		if _, ok := r.PlateSections[p.ID]; ok {
			return nil, nil, fmt.Errorf("duplicate plate section request %d", p.ID)
		}
		s, err := p.apply(f)
		if err != nil {
			return nil, nil, err
		}
		r.PlateSections[p.ID] = plateLoads(s)
	}
	plates := q.Plates
	if q.PanelGrid != nil {
		generated, err := q.PanelGrid.cells(f)
		if err != nil {
			return nil, nil, err
		}
		plates = append(slices.Clone(plates), generated...)
	}
	if len(plates) > 100000 {
		return nil, nil, fmt.Errorf("too many plates in one construction")
	}
	var addedPlates []*st.Elem
	if len(plates) > 0 {
		r.PlateArea = make(map[int]float64)
	}
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
			return nil, nil, fmt.Errorf("plate requires WALL or SLAB and a plate section, got %s SECT %d", in.Type, in.Section)
		}
		if len(in.Nodes) < 3 || len(in.Nodes) > 4 || in.ID < 0 || f.Elems[in.ID] != nil {
			return nil, nil, fmt.Errorf("invalid or occupied plate ID %d or invalid number of corners", in.ID)
		}
		ns := make([]*st.Node, len(in.Nodes))
		for j, id := range in.Nodes {
			ns[j] = f.Nodes[id]
			if ns[j] == nil {
				return nil, nil, fmt.Errorf("plate references unknown NODE %d", id)
			}
		}
		key := faceKey(ns)
		if id, ok := faces[key]; ok {
			return nil, nil, fmt.Errorf("face %v already occupied by ELEM %d", in.Nodes, id)
		}
		e := st.NewPlateElem(ns, sec, et)
		if ok, err := e.IsValidElem(); !ok {
			return nil, nil, err
		}
		if len(ns) == 4 && !convexPlate(ns) {
			return nil, nil, fmt.Errorf("plate corners must be in convex perimeter order: %v", in.Nodes)
		}
		id := in.ID
		if id == 0 {
			if f.Maxenum == math.MaxInt {
				return nil, nil, fmt.Errorf("plate numbering overflow")
			}
			f.Maxenum++
			id = f.Maxenum
		}
		f.AddElem(id, e)
		faces[key] = id
		addedPlates = append(addedPlates, e)
		r.MemberTypes[in.Type]++
		if _, ok := r.PlateSections[sec.Num]; !ok {
			r.PlateSections[sec.Num] = plateLoads(sec)
		}
	}
	if len(q.UpdateNodes) > 0 {
		if err := applyNodeUpdates(f, q.UpdateNodes, r); err != nil {
			return nil, nil, err
		}
	}
	for _, e := range addedPlates {
		r.PlateArea[e.Sect.Num] += e.Area()
	}
	if len(r.Sections) == 0 {
		r.Sections = nil
	}
	if len(r.Properties) == 0 {
		r.Properties = nil
	}
	r.Total = st.ConstructionCounts{Properties: len(f.Props), Sections: len(f.Sects), Nodes: len(f.Nodes), Elements: len(f.Elems)}
	r.Added = st.ConstructionCounts{Properties: r.Total.Properties - beforeProps, Sections: r.Total.Sections - beforeSects, Nodes: r.Total.Nodes - beforeNodes, Elements: r.Total.Elements - beforeElems}
	return r, updatedLST, nil
}

func applySectionUpdates(f *st.Frame, updates []updateSectionRequest, lst []byte, lstPath string, r *st.ConstructionReceipt) ([]byte, error) {
	seen := make(map[int]bool, len(updates))
	blocksByID := make(map[int]string)
	for _, b := range lstBlocks(lst) {
		if b.id > 0 {
			blocksByID[b.id] = b.text
		}
	}
	var lstEdits []lstEdit
	for _, u := range updates {
		if u.ID <= 0 {
			return nil, fmt.Errorf("invalid section ID %d", u.ID)
		}
		if seen[u.ID] {
			return nil, fmt.Errorf("duplicate update_sections entry for SECT %d", u.ID)
		}
		seen[u.ID] = true
		sec := f.Sects[u.ID]
		if sec == nil || !sec.HasArea(0) || len(sec.Figs) == 0 {
			return nil, fmt.Errorf("SECT %d does not exist or is not a line section", u.ID)
		}
		sh, err := (st.BuildSection{ID: u.ID, Shape: u.Shape, Dimensions: u.Dimensions}).ModelShape()
		if err != nil {
			return nil, fmt.Errorf("SECT %d: %w", u.ID, err)
		}
		if u.Property != nil {
			prop := f.Props[*u.Property]
			if prop == nil {
				return nil, fmt.Errorf("SECT %d: unknown PROP %d", u.ID, *u.Property)
			}
			sec.Figs[0].Prop = prop
		}
		sec.Figs[0].SetShapeProperty(sh)
		name := u.Name
		if name == "" {
			name = sh.Description()
		}
		if len(strings.Fields(name)) != 1 || strings.TrimSpace(name) != name {
			return nil, fmt.Errorf("SECT %d name must be one INP token", u.ID)
		}
		sec.Name = name
		figs, props := st.ConstructedFigs(sec)
		r.Sections[u.ID] = figs
		for pid, mat := range props {
			r.Properties[pid] = mat
		}
		r.UpdatedSections++

		syncLST := u.SyncLST == nil || *u.SyncLST
		explicitSync := u.SyncLST != nil && *u.SyncLST
		if !syncLST {
			continue
		}
		if len(lst) == 0 {
			if explicitSync {
				return nil, fmt.Errorf("SECT %d: sync_lst requested but no LST is loaded", u.ID)
			}
			continue
		}
		blockText, exists := blocksByID[u.ID]
		if !exists {
			if explicitSync {
				return nil, fmt.Errorf("SECT %d: no CODE block in loaded LST", u.ID)
			}
			continue
		}
		overrideMat := ""
		if u.Property != nil && sec.Figs[0].Prop != nil && sec.Figs[0].Prop.Material != nil {
			overrideMat = sec.Figs[0].Prop.Material.Name()
		}
		newBlock, synced, err := syncSteelLSTBlock(u.ID, blockText, sh, overrideMat, explicitSync)
		if err != nil {
			return nil, err
		}
		if synced {
			lstEdits = append(lstEdits, lstEdit{Section: u.ID, Definition: newBlock})
			r.SyncedLST = append(r.SyncedLST, u.ID)
		}
	}
	if len(lstEdits) == 0 {
		return lst, nil
	}
	updatedLST, _, err := amendLst(lst, lstEdits, f)
	if err != nil {
		return nil, err
	}
	if err := readLst(f, updatedLST, lstPath); err != nil {
		return nil, err
	}
	slices.Sort(r.SyncedLST)
	return updatedLST, nil
}

func syncSteelLSTBlock(id int, rawBlock string, sh st.Shape, overrideMaterial string, explicit bool) (string, bool, error) {
	decoded := lstText(rawBlock)
	lines := strings.Split(strings.ReplaceAll(decoded, "\r\n", "\n"), "\n")
	headerIdx, shapeIdx := -1, -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if headerIdx < 0 {
			headerIdx = i
		} else {
			shapeIdx = i
			break
		}
	}
	if headerIdx < 0 || shapeIdx < 0 {
		if explicit {
			return "", false, fmt.Errorf("SECT %d: malformed LST CODE block", id)
		}
		return "", false, nil
	}
	headerFields := strings.Fields(lines[headerIdx])
	if len(headerFields) < 4 || !strings.EqualFold(headerFields[0], "CODE") {
		if explicit {
			return "", false, fmt.Errorf("SECT %d: malformed LST CODE header", id)
		}
		return "", false, nil
	}
	if !strings.EqualFold(headerFields[2], "S") {
		if explicit {
			return "", false, fmt.Errorf("SECT %d: LST block is %s, not steel (S); use preview_lst_edits", id, headerFields[2])
		}
		return "", false, nil
	}
	shapeFields := strings.Fields(lines[shapeIdx])
	_, size, err := st.ParseShape(shapeFields)
	if err != nil || len(shapeFields) <= 1+size {
		if explicit {
			return "", false, fmt.Errorf("SECT %d: cannot parse existing LST shape line", id)
		}
		return "", false, nil
	}
	material := shapeFields[1+size]
	if overrideMaterial != "" {
		material = overrideMaterial
	}
	lines[shapeIdx] = fmt.Sprintf("%s %s \"%s\"", strings.Join(strings.Fields(sh.String()), " "), material, sh.Description())
	return strings.TrimSpace(strings.Join(lines, "\n")), true, nil
}

func applyNodeUpdates(f *st.Frame, updates []st.BuildNode, r *st.ConstructionReceipt) error {
	moved := make(map[int]bool, len(updates))
	for _, in := range updates {
		if in.ID <= 0 || len(in.XYZ) != 3 {
			return fmt.Errorf("invalid update_nodes entry for NODE %d", in.ID)
		}
		if moved[in.ID] {
			return fmt.Errorf("duplicate update_nodes entry for NODE %d", in.ID)
		}
		node := f.Nodes[in.ID]
		if node == nil {
			return fmt.Errorf("update_nodes references unknown NODE %d", in.ID)
		}
		for _, v := range in.XYZ {
			if math.IsInf(v, 0) || math.IsNaN(v) {
				return fmt.Errorf("NODE %d has nonfinite coordinates", in.ID)
			}
		}
		copy(node.Coord, in.XYZ)
		moved[in.ID] = true
	}
	if err := f.ValidateNodePositions(); err != nil {
		return err
	}
	for _, e := range f.Elems {
		touched := false
		for _, n := range e.Enod {
			if n != nil && moved[n.Num] {
				touched = true
				break
			}
		}
		if !touched {
			continue
		}
		if e.IsLineElem() {
			length := e.Length()
			if !(length > 0) || math.IsInf(length, 0) {
				return fmt.Errorf("ELEM %d has invalid length after node update", e.Num)
			}
		} else {
			area := e.Area()
			if !(area > 0) || math.IsInf(area, 0) || math.IsNaN(area) {
				return fmt.Errorf("ELEM %d has invalid area after node update", e.Num)
			}
		}
	}
	r.UpdatedNodes = len(updates)
	return nil
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
