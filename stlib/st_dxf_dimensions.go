package st

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/yofu/dxf"
	"github.com/yofu/dxf/codepage"
	"github.com/yofu/dxf/entity"
	"github.com/yofu/dxf/insunit"
)

type dxfDatum struct {
	value float64
	name  string
}

// dimensionFrame isolates sidecar loading and temporary section projection
// from the live model. Existing (possibly unsaved) grids take precedence.
func (f *Frame) dimensionFrame() (*Frame, error) {
	r := f.Snapshot()
	if len(r.Kijuns) == 0 && f.Path != "" {
		path := Ce(f.Path, ".kjn")
		if _, err := os.Stat(path); err == nil {
			if err := r.ReadKjn(path); err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return r, nil
}

func mergeDXFDatums(values []dxfDatum) []dxfDatum {
	sort.Slice(values, func(i, j int) bool {
		if values[i].value == values[j].value {
			return values[i].name < values[j].name
		}
		return values[i].value < values[j].value
	})
	out := make([]dxfDatum, 0, len(values))
	for _, v := range values {
		if len(out) > 0 && math.Abs(v.value-out[len(out)-1].value) < 1e-4 {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (f *Frame) dxfGrids(axis int) []dxfDatum {
	var values []dxfDatum
	for _, k := range f.Kijuns {
		if math.Abs(k.Start[2]) > 1e-4 || math.Abs(k.End[2]) > 1e-4 {
			continue
		}
		if math.Abs(k.Start[axis]-k.End[axis]) < 1e-4 && math.Abs(k.Start[1-axis]-k.End[1-axis]) > 1e-4 {
			values = append(values, dxfDatum{k.Start[axis], k.Name})
		}
	}
	if len(values) > 0 {
		return mergeDXFDatums(values)
	}
	// When no grids exist, use column axes rather than every analysis node.
	for _, e := range f.Elems {
		if e.IsLineElem() && e.Etype == COLUMN {
			for _, n := range e.Enod {
				values = append(values, dxfDatum{n.Coord[axis], ""})
			}
		}
	}
	values = mergeDXFDatums(values)
	for i := range values {
		values[i].name = fmt.Sprintf("%s%d", []string{"X", "Y"}[axis], i+1)
	}
	return values
}

func (f *Frame) dxfLevels() []dxfDatum {
	if f.drawingSettings != nil && (len(f.drawingSettings.Levels) > 0 || f.drawingSettings.GroundLevelMM != nil) {
		var configured []dxfDatum
		if z := f.drawingSettings.GroundLevelMM; z != nil {
			configured = append(configured, dxfDatum{*z / 1000, "GL"})
		}
		for _, l := range f.drawingSettings.Levels {
			configured = append(configured, dxfDatum{l.ElevationMM / 1000, l.Name})
		}
		return mergeDXFDatums(configured)
	}
	minZ, maxZ := math.Inf(1), math.Inf(-1)
	for _, n := range f.Nodes {
		minZ = math.Min(minZ, n.Coord[2])
		maxZ = math.Max(maxZ, n.Coord[2])
	}
	var levels []dxfDatum
	for _, k := range f.Kijuns {
		isLevel := math.Abs(k.Start[2]) > 1e-4 || strings.HasSuffix(strings.ToUpper(k.Name), "FL")
		if isLevel && math.Abs(k.Start[2]-k.End[2]) < 1e-4 && k.Start[2] > minZ+1e-4 && k.Start[2] <= maxZ+1e-4 {
			levels = append(levels, dxfDatum{k.Start[2], k.Name})
		}
	}
	if len(levels) > 0 {
		return mergeDXFDatums(append(levels, dxfDatum{minZ, "GL"}))
	}
	// Ai.Boundary splits stories; it is not itself a set of floor elevations.
	// Infer a representative beam level in each story when no KJN levels exist.
	groups := make(map[int][]float64)
	for _, e := range f.Elems {
		if !e.IsLineElem() || e.Etype != GIRDER || math.Abs(e.Enod[0].Coord[2]-e.Enod[1].Coord[2]) > 1e-4 {
			continue
		}
		z := e.Enod[0].Coord[2]
		band := sort.SearchFloat64s(f.Ai.Boundary, z)
		if len(f.Ai.Boundary) == 0 {
			band = int(math.Round(z * 10000))
		}
		groups[band] = append(groups[band], z)
	}
	for _, zs := range groups {
		sort.Float64s(zs)
		best, count, current := zs[0], 0, 0
		last := math.Inf(-1)
		for _, z := range zs {
			if math.Abs(z-last) > 1e-4 {
				current = 0
			}
			current++
			if current > count {
				best, count = z, current
			}
			last = z
		}
		levels = append(levels, dxfDatum{best, ""})
	}
	levels = mergeDXFDatums(append(levels, dxfDatum{minZ, ""}, dxfDatum{maxZ, ""}))
	for i := range levels {
		levels[i].name = fmt.Sprintf("%dFL", i+1)
	}
	return levels
}

// WriteDxfPlanDimensions exports an annotated floor slab of the analysis
// model. scale converts model metres into drawing units (normally 1000/mm).
func (f *Frame) WriteDxfPlanDimensions(path string, floor int, scale, textheight, axissize float64) error {
	if floor <= 0 || floor >= len(f.Ai.Boundary) {
		return fmt.Errorf("floor out of bounds")
	}
	return f.WriteDxfCrosssectionDimensions(path, 2, f.Ai.Boundary[floor-1], f.Ai.Boundary[floor], 1, scale, textheight, axissize)
}

func (f *Frame) WriteDxfCrosssectionDimensions(path string, axis int, min, max float64, side int, scale, textheight, axissize float64) error {
	if axis < 0 || axis > 2 || side < 0 || side > 1 || math.IsNaN(min) || math.IsNaN(max) || math.IsInf(min, 0) || math.IsInf(max, 0) || min > max {
		return fmt.Errorf("invalid section plane or range")
	}
	for _, v := range []float64{scale, textheight, axissize} {
		if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("drawing sizes must be finite and positive")
		}
	}
	r, err := f.dimensionFrame()
	if err != nil {
		return err
	}
	if r.drawingSettings == nil {
		cfg, err := r.loadDXFDrawingSettings("")
		if err != nil {
			return err
		}
		if cfg != nil {
			if _, err = r.applyDXFDrawingSettings(cfg); err != nil {
				return err
			}
		}
	}
	drafting := r.dxfDraftingStyle()
	plotScale := textheight / 2.5
	r.drawingPlotScale = plotScale
	if r.drawingSettings == nil {
		r.drawingSettings = &DXFDrawingSettings{Version: 1}
	}
	r.drawingSettings.Style = drafting
	d, err := r.buildDxfCrosssection(axis, min, max, side, scale, drafting.TextHeightMM*plotScale, axissize, false)
	if err != nil {
		return err
	}
	// Bounds only from actual model geometry, before annotation is added.
	lo, hi := []float64{math.Inf(1), math.Inf(1)}, []float64{math.Inf(-1), math.Inf(-1)}
	geometry := false
	for _, e := range d.Entities() {
		if _, text := e.(*entity.Text); text {
			continue
		}
		a, b := e.BBox()
		for i := 0; i < 2; i++ {
			lo[i] = math.Min(lo[i], a[i])
			hi[i] = math.Max(hi[i], b[i])
		}
		geometry = true
	}
	if !geometry {
		return fmt.Errorf("no members in the requested drawing range")
	}
	d.Header().CodePage = codepage.Japanese
	if scale == 1000 {
		d.Header().InsUnit = insunit.Millimeters
	}
	for _, name := range []string{"GRID", "GRID_TEXT", "FLOOR_LEVEL", "DIM_SPAN", "DIM_TOTAL", "DIM_HEIGHT"} {
		if _, err = d.AddLayer(name, dxf.DefaultColor, dxf.DefaultLineType, false); err != nil {
			return err
		}
	}
	style, err := d.DimStyle("Standard")
	if err != nil {
		return err
	}
	style.TextHeight, style.ArrowSize, style.TextGap = drafting.Dimension.TextHeightMM*plotScale, drafting.Dimension.ArrowSizeMM*plotScale, drafting.Dimension.TextGapMM*plotScale
	style.ExtOffset, style.ExtExtend, style.Decimals = drafting.Dimension.ExtOffsetMM*plotScale, drafting.Dimension.ExtExtendMM*plotScale, *drafting.Dimension.Decimals
	var horizontal, vertical []dxfDatum
	if axis == 2 {
		horizontal, vertical = r.dxfGrids(0), r.dxfGrids(1)
	} else {
		horizontal, vertical = r.dxfGrids(1-axis), r.dxfLevels()
	}
	// Never include a grid which is outside this view merely because another
	// part of the building happens to use it.
	filter := func(values []dxfDatum, component int) []dxfDatum {
		var out []dxfDatum
		for _, v := range values {
			v.value *= scale
			if v.value >= lo[component]-axissize && v.value <= hi[component]+axissize {
				out = append(out, v)
			}
		}
		return out
	}
	horizontal, vertical = filter(horizontal, 0), filter(vertical, 1)
	text := func(name string, x, y float64) error {
		if _, err := d.Layer("GRID_TEXT", true); err != nil {
			return err
		}
		t, err := d.Text(name, x, y, 0, drafting.GridTextHeightMM*plotScale)
		if err != nil {
			return err
		}
		t.Anchor(entity.CENTER_CENTER)
		return nil
	}
	for _, g := range horizontal {
		d.Layer("GRID", true)
		l, err := d.Line(g.value, lo[1]-2*textheight, 0, g.value, hi[1]+2*textheight, 0)
		if err != nil {
			return err
		}
		lt, _ := d.LineType("MCDASH6")
		l.SetLineType(lt)
		if _, err = d.Circle(g.value, lo[1]-3*textheight, 0, axissize); err != nil {
			return err
		}
		if err = text(g.name, g.value, lo[1]-3*textheight); err != nil {
			return err
		}
	}
	for _, g := range vertical {
		layer := "GRID"
		if axis != 2 {
			layer = "FLOOR_LEVEL"
		}
		d.Layer(layer, true)
		l, err := d.Line(lo[0]-2*textheight, g.value, 0, hi[0]+2*textheight, g.value, 0)
		if err != nil {
			return err
		}
		lt, _ := d.LineType("MCDASH6")
		l.SetLineType(lt)
		if axis == 2 {
			if _, err = d.Circle(lo[0]-3*textheight, g.value, 0, axissize); err != nil {
				return err
			}
		}
		if err = text(g.name, lo[0]-3*textheight, g.value); err != nil {
			return err
		}
	}
	dim := func(a, b, location []float64, angle float64, layer string) error {
		if _, err := d.Layer(layer, true); err != nil {
			return err
		}
		_, err := d.LinearDimension(a, b, location, angle)
		return err
	}
	for i := 1; i < len(horizontal); i++ {
		if err = dim([]float64{horizontal[i-1].value, lo[1], 0}, []float64{horizontal[i].value, lo[1], 0}, []float64{0, lo[1] - 6*textheight, 0}, 0, "DIM_SPAN"); err != nil {
			return err
		}
	}
	if len(horizontal) > 1 {
		if err = dim([]float64{horizontal[0].value, lo[1], 0}, []float64{horizontal[len(horizontal)-1].value, lo[1], 0}, []float64{0, lo[1] - 10*textheight, 0}, 0, "DIM_TOTAL"); err != nil {
			return err
		}
	}
	layer := "DIM_SPAN"
	if axis != 2 {
		layer = "DIM_HEIGHT"
	}
	for i := 1; i < len(vertical); i++ {
		if err = dim([]float64{hi[0], vertical[i-1].value, 0}, []float64{hi[0], vertical[i].value, 0}, []float64{hi[0] + 6*textheight, 0, 0}, 90, layer); err != nil {
			return err
		}
	}
	if len(vertical) > 1 {
		if err = dim([]float64{hi[0], vertical[0].value, 0}, []float64{hi[0], vertical[len(vertical)-1].value, 0}, []float64{hi[0] + 10*textheight, 0, 0}, 90, "DIM_TOTAL"); err != nil {
			return err
		}
	}
	x, y := 0, 1
	if axis != 2 {
		x, y = 1-axis, 2
	}
	if err = r.drawDXFAnnotationsInRange(d, r.dxfDrawingViewKey(axis, min, max), axis, min, max, x, y, plotScale); err != nil {
		return err
	}
	applyDXFDraftingEntities(d, drafting, plotScale)
	d.SetExt()
	return d.SaveAs(path)
}

// resolveDxfJiku selects an axis-aligned grid without depending on its endpoint
// order. Slanted frame projections require a separate local-coordinate view.
func (f *Frame) resolveDxfJiku(name string) (int, float64, error) {
	k := f.Kijuns[strings.ToLower(name)]
	if k == nil {
		return 0, 0, fmt.Errorf("kijun %s doesn't exist", name)
	}
	dx, dy := math.Abs(k.End[0]-k.Start[0]), math.Abs(k.End[1]-k.Start[1])
	if dx < 1e-4 && dy > 1e-4 {
		return 0, k.Start[0], nil
	}
	if dy < 1e-4 && dx > 1e-4 {
		return 1, k.Start[1], nil
	}
	return 0, 0, fmt.Errorf("kijun %s is not parallel to X or Y", name)
}
