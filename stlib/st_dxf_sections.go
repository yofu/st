package st

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type DXFRebar struct {
	X        float64 `json:"x_mm"`
	Y        float64 `json:"y_mm"`
	Radius   float64 `json:"radius_mm"`
	Caption  string  `json:"caption"`
	Material string  `json:"material"`
}

// DXFSection describes the actual current Allow definition. Missing details
// remain missing; equivalent area alone never implies a bar layout or shape.
type DXFSection struct {
	ID       int        `json:"id"`
	Symbol   string     `json:"symbol"`
	Kind     string     `json:"kind"`
	Material string     `json:"material"`
	Shape    string     `json:"shape"`
	MainBars string     `json:"main_bars,omitempty"`
	Hoops    string     `json:"hoops,omitempty"`
	Floors   []string   `json:"floors"`
	Members  int        `json:"members"`
	Bars     []DXFRebar `json:"bars,omitempty"`
	Warnings []string   `json:"warnings,omitempty"`
	vertices [][]float64
	circles  [][3]float64
}

func dxfShapeMM(shape Shape) string {
	parts := strings.Fields(shape.String())
	for i := 1; i < len(parts); i++ {
		if n, e := strconv.ParseFloat(parts[i], 64); e == nil {
			parts[i] = strconv.FormatFloat(n*10, 'f', -1, 64)
		}
	}
	if len(parts) > 1 {
		return parts[0] + " " + strings.Join(parts[1:], "×")
	}
	return shape.String()
}

// DXFSections returns used definitions by default; explicit IDs may also name
// unused sections. Coordinates/dimensions are converted from section cm to mm.
func (f *Frame) DXFSections(ids []int) ([]DXFSection, error) {
	use := map[int][]*Elem{}
	for _, e := range f.Elems {
		if e.Sect != nil {
			use[e.Sect.Num] = append(use[e.Sect.Num], e)
		}
	}
	if len(ids) == 0 {
		for id := range use {
			ids = append(ids, id)
		}
	} else {
		ids = append([]int(nil), ids...)
	}
	sort.Ints(ids)
	levels := f.dxfLevels()
	var records []DXFSection
	seen := map[int]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		s := f.Sects[id]
		if s == nil {
			return nil, fmt.Errorf("section %d not found", id)
		}
		r := DXFSection{ID: id, Symbol: strings.Trim(s.Name, "\" "), Members: len(use[id])}
		if s.Allow != nil && strings.Trim(s.Allow.Name(), "\" ") != "" {
			r.Symbol = strings.Trim(s.Allow.Name(), "\" ")
		}
		if r.Symbol == "" {
			r.Symbol = fmt.Sprintf("S%d", id)
		}
		floorSet := map[string]bool{}
		floorOrder := map[string]float64{}
		for _, e := range use[id] {
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, n := range e.Enod {
				lo = math.Min(lo, n.Coord[2])
				hi = math.Max(hi, n.Coord[2])
			}
			if hi-lo > .05 && (e.Etype == COLUMN || e.Etype == BRACE) {
				for i := 1; i < len(levels); i++ {
					if lo < levels[i].value-1e-4 && hi > levels[i-1].value+1e-4 {
						name := levels[i-1].name + "～" + levels[i].name
						floorSet[name] = true
						floorOrder[name] = levels[i-1].value
					}
				}
			} else if len(levels) > 0 {
				closest := 0
				for i := range levels {
					if math.Abs(levels[i].value-(lo+hi)/2) < math.Abs(levels[closest].value-(lo+hi)/2) {
						closest = i
					}
				}
				name := levels[closest].name
				if math.Abs(levels[closest].value-(lo+hi)/2) > .05 {
					name += "付近"
				}
				floorSet[name] = true
				floorOrder[name] = levels[closest].value
			}
		}
		for name := range floorSet {
			r.Floors = append(r.Floors, name)
		}
		sort.Slice(r.Floors, func(i, j int) bool {
			if floorOrder[r.Floors[i]] == floorOrder[r.Floors[j]] {
				return r.Floors[i] < r.Floors[j]
			}
			return floorOrder[r.Floors[i]] < floorOrder[r.Floors[j]]
		})
		var rc *RCColumn
		var steel *SColumn
		switch a := s.Allow.(type) {
		case *RCColumn:
			rc = a
			r.Kind = "RC柱"
		case *RCGirder:
			rc = &a.RCColumn
			r.Kind = "RC梁"
		case *SColumn:
			steel = a
			r.Kind = "S柱"
		case *SGirder:
			steel = &a.SColumn
			r.Kind = "S梁"
		case *SBrace:
			steel = &a.SColumn
			r.Kind = "Sブレース"
		case *RCWall:
			r.Kind = "RC壁"
			r.Material = a.Concrete.Name()
			r.Shape = fmt.Sprintf("厚さ %g mm", a.Thick*10)
		case *RCSlab:
			r.Kind = "RC床"
			r.Material = a.Concrete.Name()
			r.Shape = fmt.Sprintf("厚さ %g mm", a.Thick*10)
		default:
			r.Kind = "未対応"
			r.Warnings = append(r.Warnings, "RC/S柱・梁等のLST定義がありません")
		}
		if rc != nil {
			r.Material = rc.Concrete.Name()
			if rc.CShape == nil {
				r.Warnings = append(r.Warnings, "コンクリート形状がありません")
			} else {
				r.Shape = fmt.Sprintf("%g×%g mm", rc.CShape.Breadth(false)*10, rc.CShape.Breadth(true)*10)
				r.vertices = rc.CShape.Vertices()
				if len(s.Figs) > 0 && s.Figs[0].Shape != nil {
					sh := s.Figs[0].Shape
					if math.Abs(sh.Breadth(true)-rc.CShape.Breadth(true)) > .1 || math.Abs(sh.Breadth(false)-rc.CShape.Breadth(false)) > .1 {
						r.Warnings = append(r.Warnings, "INPとLSTのコンクリート寸法が異なります（現在のLST定義を使用）")
					}
				}
			}
			counts := map[string]int{}
			for _, b := range rc.Reins {
				if len(b.Position) != 2 || b.Area <= 0 || !finiteDXF(b.Area) || !finiteDXF(b.Position[0]) || !finiteDXF(b.Position[1]) {
					return nil, fmt.Errorf("section %d: invalid reinforcing bar", id)
				}
				caption := strings.Trim(b.Caption, "\" ")
				if caption == "" {
					caption = fmt.Sprintf("φ%.1f相当", 2*b.Radius()*10)
				}
				group := caption
				if b.Material.Name != "" {
					group += " (" + b.Material.Name + ")"
				}
				counts[group]++
				r.Bars = append(r.Bars, DXFRebar{b.Position[0] * 10, b.Position[1] * 10, b.Radius() * 10, caption, b.Material.Name})
			}
			var captions []string
			for c := range counts {
				captions = append(captions, c)
			}
			sort.Strings(captions)
			for _, c := range captions {
				if r.MainBars != "" {
					r.MainBars += " / "
				}
				r.MainBars += fmt.Sprintf("%d-%s", counts[c], c)
			}
			r.Hoops = strings.Trim(rc.Hoops.Name, "\" ")
			if r.Hoops != "" && rc.Hoops.Material.Name != "" {
				r.Hoops += " (" + rc.Hoops.Material.Name + ")"
			}
			if len(r.Bars) == 0 {
				r.Warnings = append(r.Warnings, "主筋位置がありません")
			}
			if r.Hoops == "" {
				r.Warnings = append(r.Warnings, "帯筋・あばら筋の記号がありません（配筋を推定しません）")
			}
		}
		if steel != nil {
			r.Material = steel.Steel.Name()
			if steel.Shape == nil {
				r.Warnings = append(r.Warnings, "鋼材形状がありません")
			} else {
				r.Shape = dxfShapeMM(steel.Shape)
				switch shape := steel.Shape.(type) {
				case SAREA:
					r.Shape = fmt.Sprintf("断面積 %g mm²（外形未定義）", shape.Area*100)
				case THICK:
					r.Shape = fmt.Sprintf("厚さ %g mm（幅未定義）", shape.Thickness*10)
				case CPIPE:
					pipe := shape
					r.circles = [][3]float64{{0, 0, pipe.D * 5}, {0, 0, (pipe.D/2 - pipe.T) * 10}}
				default:
					r.vertices = steel.Shape.Vertices()
				}
				if len(r.vertices) == 0 && len(r.circles) == 0 {
					r.Warnings = append(r.Warnings, "この鋼材形状の外形を描画できません")
				}
				if len(s.Figs) > 0 && s.Figs[0].Shape != nil && s.Figs[0].Shape.String() != steel.Shape.String() {
					r.Warnings = append(r.Warnings, "INPとLSTの鋼材形状が異なります（現在のLST定義を使用）")
				}
			}
		}
		for i, v := range r.vertices {
			if v != nil {
				r.vertices[i] = []float64{v[0] * 10, v[1] * 10}
			}
		}
		if r.Material == "" {
			r.Warnings = append(r.Warnings, "材料名がありません")
		}
		if rc == nil && steel == nil {
			r.Warnings = append(r.Warnings, "この断面種別の断面図は未対応です")
		}
		records = append(records, r)
	}
	return records, nil
}
