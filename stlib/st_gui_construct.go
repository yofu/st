package st

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
)

// Distances in the model are metres. Section dimensions supplied by MCP are mm.
type GUIConstruction struct {
	Properties []BuildProperty `json:"properties,omitempty"`
	Sections   []BuildSection  `json:"sections,omitempty"`
	Nodes      []BuildNode     `json:"nodes,omitempty"`
	Elements   []BuildElement  `json:"elements,omitempty"`
	Grid       *BuildGrid      `json:"grid,omitempty"`
}
type BuildProperty struct {
	ID       int    `json:"id"`
	Material string `json:"material"`
}
type BuildSection struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Property   int       `json:"property"`
	Shape      string    `json:"shape"`
	Dimensions []float64 `json:"dimensions_mm"`
}
type BuildNode struct {
	ID  int       `json:"id"`
	XYZ []float64 `json:"xyz_m"`
}
type BuildElement struct {
	ID      int    `json:"id"`
	Nodes   []int  `json:"nodes"`
	Type    string `json:"type"`
	Section int    `json:"section"`
}
type BuildGrid struct {
	Origin       []float64 `json:"origin_m,omitempty"`
	Cells        []int     `json:"cells"`
	Pitch        []float64 `json:"pitch_m"`
	FirstNode    int       `json:"first_node,omitempty"`
	FirstElement int       `json:"first_element,omitempty"`
	Type         string    `json:"type"`
	Section      int       `json:"section"`
}
type ConstructionCounts struct {
	Properties int `json:"properties"`
	Sections   int `json:"sections"`
	Nodes      int `json:"nodes"`
	Elements   int `json:"elements"`
}
type ConstructedFig struct {
	Property   int       `json:"property"`
	Material   string    `json:"material"`
	Shape      string    `json:"shape_cm,omitempty"`
	Dimensions []float64 `json:"dimensions_mm,omitempty"`
}
type ConstructionReceipt struct {
	PlateSections   map[int]PlateLoadReceipt `json:"plate_loads,omitempty"`
	PlateArea       map[int]float64          `json:"added_plate_area_m2,omitempty"`
	Added           ConstructionCounts       `json:"added"`
	Total           ConstructionCounts       `json:"total"`
	UpdatedNodes    int                      `json:"updated_nodes,omitempty"`
	BoundsMin       []float64                `json:"bounds_min_m,omitempty"`
	BoundsMax       []float64                `json:"bounds_max_m,omitempty"`
	UpdatedSections int                      `json:"updated_sections,omitempty"`
	SyncedLST       []int                    `json:"synced_lst,omitempty"`
	MemberTypes     map[string]int           `json:"member_types"`
	Directions      map[string]int           `json:"directions"`
	MinLength       float64                  `json:"min_length_m,omitempty"`
	MaxLength       float64                  `json:"max_length_m,omitempty"`
	Sections        map[int][]ConstructedFig `json:"section_figures"`
	Properties      map[int]string           `json:"materials"`
}

type PlateLoadReceipt struct {
	Dead float64   `json:"dead_load_kgf_m2"`
	Live []float64 `json:"lload_tf_m2"`
}

const constructionLimit = 100000

func (g BuildGrid) expand(q *GUIConstruction, firstNode, firstElem int) error {
	if len(g.Cells) != 3 || len(g.Pitch) != 3 || (g.Origin != nil && len(g.Origin) != 3) {
		return fmt.Errorf("grid cells, pitch_m and origin_m need three coordinates")
	}
	var origin [3]float64
	copy(origin[:], g.Origin)
	count := 1
	for i, n := range g.Cells {
		if n < 0 || n >= constructionLimit || count > constructionLimit/(n+1) || n > 0 && !(g.Pitch[i] > 0) {
			return fmt.Errorf("invalid grid cells or pitch_m on axis %d", i)
		}
		count *= n + 1
	}
	edges := 0
	for _, n := range g.Cells {
		edges += count / (n + 1) * n
	}
	if edges == 0 || count+len(q.Nodes) > constructionLimit || edges+len(q.Elements) > constructionLimit {
		return fmt.Errorf("empty or oversized grid")
	}
	if g.FirstNode != 0 {
		firstNode = g.FirstNode
	}
	if g.FirstElement != 0 {
		firstElem = g.FirstElement
	}
	if firstNode <= 0 || firstElem <= 0 || firstNode > math.MaxInt-count || firstElem > math.MaxInt-edges {
		return fmt.Errorf("invalid grid numbering")
	}
	strides := [3]int{1, g.Cells[0] + 1, (g.Cells[0] + 1) * (g.Cells[1] + 1)}
	for i := 0; i < count; i++ {
		xyz := origin
		for axis, stride := range strides {
			k := i / stride % (g.Cells[axis] + 1)
			xyz[axis] += float64(k) * g.Pitch[axis]
			if k < g.Cells[axis] {
				q.Elements = append(q.Elements, BuildElement{firstElem, []int{firstNode + i, firstNode + i + stride}, g.Type, g.Section})
				firstElem++
			}
		}
		q.Nodes = append(q.Nodes, BuildNode{firstNode + i, xyz[:]})
	}
	return nil
}

// ModelShape validates and parses a construction section shape (dimensions in mm).
func (s BuildSection) ModelShape() (Shape, error) {
	return s.modelShape()
}

// ConstructedFigs summarizes a section's figures in MCP receipt format.
func ConstructedFigs(sec *Sect) ([]ConstructedFig, map[int]string) {
	if sec == nil {
		return nil, nil
	}
	figs := make([]ConstructedFig, 0, len(sec.Figs))
	props := make(map[int]string, len(sec.Figs))
	for _, fig := range sec.Figs {
		p := fig.Prop
		material := p.Name
		if p.Material != nil {
			material = p.Material.Name()
		}
		r := ConstructedFig{Property: p.Num, Material: material}
		if fig.Shape != nil {
			r.Shape = fig.Shape.String()
		}
		switch sh := fig.Shape.(type) {
		case HKYOU:
			r.Dimensions = []float64{10 * sh.H, 10 * sh.B, 10 * sh.Tw, 10 * sh.Tf}
		case HWEAK:
			r.Dimensions = []float64{10 * sh.H, 10 * sh.B, 10 * sh.Tw, 10 * sh.Tf}
		case RPIPE:
			r.Dimensions = []float64{10 * sh.H, 10 * sh.B, 10 * sh.Tw, 10 * sh.Tf}
		case CPIPE:
			r.Dimensions = []float64{10 * sh.D, 10 * sh.T}
		case PLATE:
			r.Dimensions = []float64{10 * sh.H, 10 * sh.B}
		}
		figs = append(figs, r)
		props[p.Num] = material
	}
	return figs, props
}

// ValidateNodePositions rejects any two nodes within inpNodeTolerance.
func (f *Frame) ValidateNodePositions() error {
	cells := make(inpNodeCells, len(f.Nodes))
	for _, id := range slices.Sorted(maps.Keys(f.Nodes)) {
		n := f.Nodes[id]
		if other := cells.find(n.Coord); other != nil {
			return fmt.Errorf("NODE %d coincides with NODE %d", n.Num, other.Num)
		}
		cells.add(n)
	}
	return nil
}

func (s BuildSection) modelShape() (Shape, error) {
	n := 0
	switch s.Shape {
	case "HKYOU", "HWEAK", "RPIPE":
		n = 4
	case "CPIPE", "PLATE":
		n = 2
	default:
		return nil, fmt.Errorf("unknown construction shape %q", s.Shape)
	}
	d := s.Dimensions
	if len(d) != n {
		return nil, fmt.Errorf("%s requires %d dimensions_mm", s.Shape, n)
	}
	words := []string{s.Shape}
	for _, v := range d {
		if !(v > 0) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("dimensions_mm must be finite and positive")
		}
		words = append(words, strconv.FormatFloat(v/10, 'g', -1, 64))
	}
	switch s.Shape {
	case "HKYOU", "HWEAK", "RPIPE":
		web := d[2]
		if s.Shape == "RPIPE" {
			web *= 2
		}
		if web >= d[1] || 2*d[3] >= d[0] {
			return nil, fmt.Errorf("thickness exceeds section envelope")
		}
	case "CPIPE":
		if 2*d[1] >= d[0] {
			return nil, fmt.Errorf("thickness exceeds pipe radius")
		}
	}
	sh, _, err := ParseShape(words)
	if err != nil {
		return nil, err
	}
	for _, v := range []float64{sh.A(), sh.Ix(), sh.Iy(), sh.J()} {
		if !(v > 0) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("invalid section properties")
		}
	}
	// Native INP stores a shape through its String method. Catch precision
	// loss before committing dimensions that would change when reopened.
	round, _, err := ParseShape(strings.Fields(sh.String()))
	if err != nil || round != sh {
		return nil, fmt.Errorf("dimensions exceed native INP shape precision")
	}
	return sh, nil
}

// Stage all new records privately. Existing nodes, members and sections are
// only referenced; an error never leaves a partially constructed model.
func (f *Frame) Construct(q GUIConstruction, beforeCommit func()) (*ConstructionReceipt, error) {
	maxNode, maxElem := f.Maxnnum, f.Maxenum
	for _, n := range q.Nodes {
		maxNode = max(maxNode, n.ID)
	}
	for _, e := range q.Elements {
		maxElem = max(maxElem, e.ID)
	}
	if q.Grid != nil {
		if err := q.Grid.expand(&q, maxNode+1, maxElem+1); err != nil {
			return nil, err
		}
	}
	if len(q.Nodes) > constructionLimit || len(q.Elements) > constructionLimit || len(q.Nodes)+len(q.Elements)+len(q.Properties)+len(q.Sections) == 0 {
		return nil, fmt.Errorf("empty or oversized construction")
	}
	props, sects := maps.Clone(f.Props), maps.Clone(f.Sects)
	nodes, elems := maps.Clone(f.Nodes), maps.Clone(f.Elems)
	receipt := &ConstructionReceipt{MemberTypes: map[string]int{}, Directions: map[string]int{}, Properties: map[int]string{}, Sections: map[int][]ConstructedFig{}}
	for _, in := range q.Properties {
		if in.ID <= 0 || props[in.ID] != nil {
			return nil, fmt.Errorf("invalid or occupied PROP %d", in.ID)
		}
		m, err := materialname(in.Material)
		if err != nil {
			return nil, err
		}
		props[in.ID] = &Prop{Num: in.ID, Name: m.Name(), Material: m, HFactor: 1, EFactor: 1, Color: 16711935}
		receipt.Properties[in.ID] = props[in.ID].Material.Name()
	}
	for _, in := range q.Sections {
		if in.ID <= 0 || sects[in.ID] != nil || props[in.Property] == nil {
			return nil, fmt.Errorf("invalid SECT %d or referenced PROP %d", in.ID, in.Property)
		}
		sh, err := in.modelShape()
		if err != nil {
			return nil, fmt.Errorf("SECT %d: %w", in.ID, err)
		}
		name := in.Name
		if name == "" {
			name = sh.Description()
		}
		if len(strings.Fields(name)) != 1 || strings.TrimSpace(name) != name {
			return nil, fmt.Errorf("SECT %d name must be one INP token", in.ID)
		}
		sec := NewSect()
		sec.Num, sec.Original, sec.Frame, sec.Name = in.ID, in.ID, f, name
		fig := NewFig()
		fig.Prop = props[in.Property]
		fig.SetShapeProperty(sh)
		sec.Figs = []*Fig{fig}
		sects[in.ID] = sec
		receipt.Sections[in.ID] = nil
	}
	cells := make(inpNodeCells)
	if len(q.Nodes) > 0 {
		for _, n := range nodes {
			cells.add(n)
		}
	}
	for _, in := range q.Nodes {
		if in.ID <= 0 || nodes[in.ID] != nil || len(in.XYZ) != 3 {
			return nil, fmt.Errorf("invalid or occupied NODE %d", in.ID)
		}
		for _, v := range in.XYZ {
			if math.IsInf(v, 0) || math.IsNaN(v) {
				return nil, fmt.Errorf("NODE %d has nonfinite coordinates", in.ID)
			}
		}
		if other := cells.find(in.XYZ); other != nil {
			return nil, fmt.Errorf("NODE %d coincides with node %d; reference the existing node", in.ID, other.Num)
		}
		n := NewNode()
		n.Num = in.ID
		copy(n.Coord, in.XYZ)
		nodes[n.Num] = n
		cells.add(n)
		maxNode = max(maxNode, n.Num)
	}
	edgeKey := func(a, b int) [2]int { return [2]int{min(a, b), max(a, b)} }
	edges := map[[2]int]bool{}
	if len(q.Elements) > 0 {
		for _, e := range elems {
			if e.IsLineElem() {
				edges[edgeKey(e.Enod[0].Num, e.Enod[1].Num)] = true
			}
		}
	}
	for _, in := range q.Elements {
		et := slices.Index(ETYPES, in.Type)
		if in.ID <= 0 || elems[in.ID] != nil || len(in.Nodes) != 2 || et < COLUMN || et > TRUSS {
			return nil, fmt.Errorf("invalid or occupied line ELEM %d", in.ID)
		}
		a, b := nodes[in.Nodes[0]], nodes[in.Nodes[1]]
		sec := sects[in.Section]
		if a == nil || b == nil || a == b || sec == nil || !sec.HasArea(0) {
			return nil, fmt.Errorf("ELEM %d has invalid end nodes or line section", in.ID)
		}
		key := edgeKey(a.Num, b.Num)
		if edges[key] {
			return nil, fmt.Errorf("ELEM %d duplicates the member between %d and %d", in.ID, a.Num, b.Num)
		}
		edges[key] = true
		e := NewLineElem([]*Node{a, b}, sec, et)
		e.Num, e.Frame = in.ID, f
		elems[e.Num] = e
		maxElem = max(maxElem, e.Num)
		receipt.Sections[sec.Num] = nil
		receipt.MemberTypes[ETYPES[e.Etype]]++
		axis := "oblique"
		diff := []float64{b.Coord[0] - a.Coord[0], b.Coord[1] - a.Coord[1], b.Coord[2] - a.Coord[2]}
		for i, v := range diff {
			if v != 0 && diff[(i+1)%3] == 0 && diff[(i+2)%3] == 0 {
				axis = string("XYZ"[i])
			}
		}
		receipt.Directions[axis]++
		length := e.Length()
		if !(length > 0) || math.IsInf(length, 0) {
			return nil, fmt.Errorf("ELEM %d has invalid length", in.ID)
		}
		if receipt.MinLength == 0 || length < receipt.MinLength {
			receipt.MinLength = length
		}
		receipt.MaxLength = max(receipt.MaxLength, length)
	}
	for id := range receipt.Sections {
		for _, fig := range sects[id].Figs {
			p := fig.Prop
			material := p.Name
			if p.Material != nil {
				material = p.Material.Name()
			}
			r := ConstructedFig{Property: p.Num, Material: material}
			if fig.Shape != nil {
				r.Shape = fig.Shape.String()
			}
			switch sh := fig.Shape.(type) {
			case HKYOU:
				r.Dimensions = []float64{10 * sh.H, 10 * sh.B, 10 * sh.Tw, 10 * sh.Tf}
			case HWEAK:
				r.Dimensions = []float64{10 * sh.H, 10 * sh.B, 10 * sh.Tw, 10 * sh.Tf}
			case RPIPE:
				r.Dimensions = []float64{10 * sh.H, 10 * sh.B, 10 * sh.Tw, 10 * sh.Tf}
			case CPIPE:
				r.Dimensions = []float64{10 * sh.D, 10 * sh.T}
			case PLATE:
				r.Dimensions = []float64{10 * sh.H, 10 * sh.B}
			}
			receipt.Sections[id] = append(receipt.Sections[id], r)
			receipt.Properties[p.Num] = material
		}
	}
	if beforeCommit != nil {
		beforeCommit()
	}
	f.Props, f.Sects, f.Nodes, f.Elems = props, sects, nodes, elems
	f.Maxnnum, f.Maxenum = maxNode, maxElem
	for _, in := range q.Sections {
		f.Show.Sect[in.ID] = true
		f.Maxsnum = max(f.Maxsnum, in.ID)
	}
	receipt.Added = ConstructionCounts{len(q.Properties), len(q.Sections), len(q.Nodes), len(q.Elements)}
	receipt.Total = ConstructionCounts{len(f.Props), len(f.Sects), len(f.Nodes), len(f.Elems)}
	return receipt, nil
}
