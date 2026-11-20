package st

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/yofu/dxf"
	"github.com/yofu/dxf/color"
	"github.com/yofu/dxf/drawing"
	"github.com/yofu/dxf/entity"
	"github.com/yofu/dxf/table"
)

// Sizes are paper mm. Layer weights are normalized to DXF's supported weights.
type DXFDraftingStyle struct {
	Monochrome       *bool                       `json:"monochrome,omitempty"`
	Font             string                      `json:"font,omitempty"`
	TextHeightMM     float64                     `json:"text_height_mm,omitempty"`
	TextWidthFactor  float64                     `json:"text_width_factor,omitempty"`
	GridTextHeightMM float64                     `json:"grid_text_height_mm,omitempty"`
	NoteTextHeightMM float64                     `json:"note_text_height_mm,omitempty"`
	Dimension        DXFDraftingDimension        `json:"dimension,omitempty"`
	Layers           map[string]DXFDraftingLayer `json:"layers,omitempty"`
}
type DXFDraftingDimension struct {
	TextHeightMM float64 `json:"text_height_mm,omitempty"`
	ArrowSizeMM  float64 `json:"arrow_size_mm,omitempty"`
	TextGapMM    float64 `json:"text_gap_mm,omitempty"`
	ExtOffsetMM  float64 `json:"extension_offset_mm,omitempty"`
	ExtExtendMM  float64 `json:"extension_extend_mm,omitempty"`
	Decimals     *int    `json:"decimals,omitempty"`
}
type DXFDraftingLayer struct {
	LineWeightMM *float64 `json:"line_weight_mm,omitempty"`
	LineType     string   `json:"line_type,omitempty"` // continuous, dashed, center
	Color        *int     `json:"color,omitempty"`
}

func dxfPtr[T any](v T) *T { return &v }
func DefaultDXFDraftingStyle() *DXFDraftingStyle {
	return &DXFDraftingStyle{Monochrome: dxfPtr(true), Font: "msgothic.ttc", TextHeightMM: 2.5, TextWidthFactor: 0.9, GridTextHeightMM: 2.5, NoteTextHeightMM: 2.5,
		Dimension: DXFDraftingDimension{TextHeightMM: 2.5, ArrowSizeMM: 1.5, TextGapMM: .75, ExtOffsetMM: 1, ExtExtendMM: 1.5, Decimals: dxfPtr(0)},
		Layers: map[string]DXFDraftingLayer{
			"column":    {LineWeightMM: dxfPtr(.35), LineType: "continuous"},
			"beam":      {LineWeightMM: dxfPtr(.25), LineType: "continuous"},
			"brace":     {LineWeightMM: dxfPtr(.13), LineType: "dashed"},
			"grid":      {LineWeightMM: dxfPtr(.13), LineType: "center"},
			"floor":     {LineWeightMM: dxfPtr(.13), LineType: "center"},
			"dimension": {LineWeightMM: dxfPtr(.13), LineType: "continuous"},
			"text":      {LineWeightMM: dxfPtr(.13), LineType: "continuous"},
			"section":   {LineWeightMM: dxfPtr(.25), LineType: "continuous"},
			"rebar":     {LineWeightMM: dxfPtr(.13), LineType: "continuous"},
			"frame":     {LineWeightMM: dxfPtr(.25), LineType: "continuous"},
			"leader":    {LineWeightMM: dxfPtr(.13), LineType: "continuous"},
		}}
}
func ResolveDXFDraftingStyle(input *DXFDraftingStyle) (*DXFDraftingStyle, error) {
	out := DefaultDXFDraftingStyle()
	positive := func(name string, v float64, dst *float64, max float64) error {
		if !finiteDXF(v) || v < 0 || v > max {
			return fmt.Errorf("invalid drafting %s", name)
		}
		if v > 0 {
			*dst = v
		}
		return nil
	}
	if input == nil {
		return out, nil
	}
	if input.Monochrome != nil {
		out.Monochrome = dxfPtr(*input.Monochrome)
	}
	if input.Font != "" {
		if strings.ContainsAny(input.Font, "\r\n\x00") {
			return nil, fmt.Errorf("invalid drafting font")
		}
		out.Font = input.Font
	}
	for _, p := range []struct {
		name string
		v    float64
		dst  *float64
		max  float64
	}{
		{"text_height_mm", input.TextHeightMM, &out.TextHeightMM, 20},
		{"text_width_factor", input.TextWidthFactor, &out.TextWidthFactor, 2},
		{"grid_text_height_mm", input.GridTextHeightMM, &out.GridTextHeightMM, 20},
		{"note_text_height_mm", input.NoteTextHeightMM, &out.NoteTextHeightMM, 20},
		{"dimension.text_height_mm", input.Dimension.TextHeightMM, &out.Dimension.TextHeightMM, 20},
		{"dimension.arrow_size_mm", input.Dimension.ArrowSizeMM, &out.Dimension.ArrowSizeMM, 20},
		{"dimension.text_gap_mm", input.Dimension.TextGapMM, &out.Dimension.TextGapMM, 20},
		{"dimension.extension_offset_mm", input.Dimension.ExtOffsetMM, &out.Dimension.ExtOffsetMM, 20},
		{"dimension.extension_extend_mm", input.Dimension.ExtExtendMM, &out.Dimension.ExtExtendMM, 20},
	} {
		if err := positive(p.name, p.v, p.dst, p.max); err != nil {
			return nil, err
		}
	}
	if out.TextWidthFactor < .3 {
		return nil, fmt.Errorf("text_width_factor must be at least 0.3")
	}
	if input.Dimension.Decimals != nil {
		if *input.Dimension.Decimals < 0 || *input.Dimension.Decimals > 6 {
			return nil, fmt.Errorf("dimension decimals must be 0..6")
		}
		out.Dimension.Decimals = dxfPtr(*input.Dimension.Decimals)
	}
	keys := make([]string, 0, len(input.Layers))
	for k := range input.Layers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		base, ok := out.Layers[key]
		if !ok {
			return nil, fmt.Errorf("unknown drafting layer role %q", key)
		}
		v := input.Layers[key]
		if v.LineWeightMM != nil {
			if !finiteDXF(*v.LineWeightMM) || *v.LineWeightMM < 0 || *v.LineWeightMM > 2.11 {
				return nil, fmt.Errorf("invalid line weight for %s", key)
			}
			base.LineWeightMM = dxfPtr(float64(table.NearestLineWeight(int(math.Round(*v.LineWeightMM*100)))) / 100)
		}
		if v.LineType != "" {
			if v.LineType != "continuous" && v.LineType != "dashed" && v.LineType != "center" {
				return nil, fmt.Errorf("invalid line type for %s", key)
			}
			base.LineType = v.LineType
		}
		if v.Color != nil {
			if *v.Color < 1 || *v.Color > 255 {
				return nil, fmt.Errorf("invalid layer color for %s", key)
			}
			base.Color = dxfPtr(*v.Color)
		}
		out.Layers[key] = base
	}
	return out, nil
}
func (f *Frame) dxfDraftingStyle() *DXFDraftingStyle {
	var input *DXFDraftingStyle
	if f.drawingSettings != nil {
		input = f.drawingSettings.Style
	}
	out, _ := ResolveDXFDraftingStyle(input)
	return out // validated before rendering
}
func (f *Frame) dxfPlotScale(textheight float64) float64 {
	if f.drawingPlotScale > 0 {
		return f.drawingPlotScale
	}
	return textheight / 2.5
}

// Offset directions are paper XY, including when the view is rotated.
func (f *Frame) dxfPaperOffset(v [2]float64) [2]float64 {
	angle := -float64(f.drawingRotation) * math.Pi / 180
	c, s := math.Cos(angle), math.Sin(angle)
	return [2]float64{v[0]*c - v[1]*s, v[0]*s + v[1]*c}
}
func dxfDraftingRole(layer string) string {
	switch layer {
	case "HASHIRA":
		return "column"
	case "HARI":
		return "beam"
	case "BRACE":
		return "brace"
	case "GRID", "KIJUN", "AXIS":
		return "grid"
	case "FLOOR_LEVEL":
		return "floor"
	case "DIM_SPAN", "DIM_TOTAL", "DIM_HEIGHT":
		return "dimension"
	case "SECTION_SHAPE":
		return "section"
	case "SECTION_REBAR":
		return "rebar"
	case "SECTION_CARD", "SHEET_FRAME":
		return "frame"
	case "LABEL_LEADER":
		return "leader"
	default:
		return "text"
	}
}
func applyDXFDraftingLayers(d *drawing.Drawing, style *DXFDraftingStyle) {
	d.Header().LtScale = 1
	// Entity scales explicitly encode the plot scale for each view. Disable
	// viewport-dependent linetype rescaling to avoid applying that scale twice.
	for _, name := range []string{"$PSLTSCALE", "$MSLTSCALE", "$LWDISPLAY"} {
		code, value := "70", "0"
		if name == "$LWDISPLAY" {
			code, value = "290", "1"
		}
		// This is called once for each drawing, before/after its view import.
		found := false
		for _, p := range d.Header().Extra {
			if p[0] == "9" && p[1] == name {
				found = true
			}
		}
		if !found {
			d.Header().Extra = append(d.Header().Extra, [2]string{"9", name}, [2]string{code, value})
		}
	}
	dashed, _ := d.LineType("ST_DASHED")
	if dashed == nil {
		dashed, _ = d.AddLineType("ST_DASHED", "3 mm dash / 1.5 mm gap", 3, -1.5)
	}
	center, _ := d.LineType("ST_CENTER")
	if center == nil {
		center, _ = d.AddLineType("ST_CENTER", "8 mm dash / dot", 8, -1.5, .5, -1.5)
	}
	for _, layer := range d.Layers {
		role := style.Layers[dxfDraftingRole(layer.Name())]
		layer.SetLineWeight(int(math.Round(*role.LineWeightMM * 100)))
		layer.LineType = dxf.DefaultLineType
		if role.LineType == "dashed" {
			layer.LineType = dashed
		}
		if role.LineType == "center" {
			layer.LineType = center
		}
		if *style.Monochrome {
			layer.Color = color.White
			layer.ClearTrueColor()
		} else if role.Color != nil {
			layer.Color = color.ColorNumber(*role.Color)
			layer.ClearTrueColor()
		}
	}
}
func applyDXFDraftingEntities(d *drawing.Drawing, style *DXFDraftingStyle, plotScale float64) {
	applyDXFDraftingLayers(d, style)
	font, _ := d.Style("ST_DRAWING", true)
	if font == nil {
		font, _ = d.AddStyle("ST_DRAWING", style.Font, "", true)
	}
	font.FontName, font.BigFontName, font.WidthFactor = style.Font, "", style.TextWidthFactor
	// Includes dimension graphics; these must print with the same line weights
	// and fonts as their editable DIMENSION owner.
	seen := map[int]bool{}
	var apply func(entity.Entity)
	apply = func(e entity.Entity) {
		if e.Handle() != 0 && seen[e.Handle()] {
			return
		}
		seen[e.Handle()] = true
		if c, ok := e.(entity.Colored); ok {
			c.SetColorByLayer()
			if *style.Monochrome {
				if clear, ok := e.(interface{ ClearTrueColor() }); ok {
					clear.ClearTrueColor()
				}
			}
		}
		if p, ok := e.(entity.LineProperties); ok {
			p.SetLineWeight(table.LineWeightByLayer)
			p.SetLineType(nil)
			if dxfDraftingRole(e.Layer().Name()) == "grid" || dxfDraftingRole(e.Layer().Name()) == "floor" {
				if _, line := e.(*entity.Line); !line {
					p.SetLineType(dxf.DefaultLineType)
				}
			}
		}
		e.SetLtscale(plotScale)
		if t, ok := e.(*entity.Text); ok {
			t.Style = font
			t.WidthFactor = style.TextWidthFactor
		}
	}
	for _, e := range d.Entities() {
		apply(e)
	}
	dimLayers := map[string]*table.Layer{}
	for _, e := range d.Entities() {
		if dim, ok := e.(*entity.Dimension); ok {
			dimLayers[dim.BlockName] = dim.Layer()
		}
	}
	for _, b := range d.Blocks() {
		for _, e := range b.Entities {
			if layer := dimLayers[b.Name]; layer != nil {
				e.SetLayer(layer)
			}
			apply(e)
		}
	}
}
