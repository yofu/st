package st

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/yofu/dxf"
	"github.com/yofu/dxf/codepage"
	"github.com/yofu/dxf/drawing"
	"github.com/yofu/dxf/entity"
	"github.com/yofu/dxf/insunit"
	"github.com/yofu/dxf/object"
)

type DXFViewSpec struct {
	Kind     string `json:"kind"` // plan or elevation
	Floor    int    `json:"floor,omitempty"`
	Grid     string `json:"grid,omitempty"`
	Title    string `json:"title,omitempty"`
	Rotation *int   `json:"rotation,omitempty"`
}
type DXFDocumentOptions struct {
	Title          string        `json:"title,omitempty"`
	Font           string        `json:"font,omitempty"`
	Paper          string        `json:"paper,omitempty"` // A3/A4 landscape, A3P/A4P portrait
	Scale          float64       `json:"scale,omitempty"` // zero: select a fitting standard scale
	Columns        int           `json:"columns,omitempty"`
	Rows           int           `json:"rows,omitempty"`
	Views          []DXFViewSpec `json:"views,omitempty"`
	Sections       bool          `json:"sections,omitempty"`
	SectionIDs     []int         `json:"section_ids,omitempty"`
	SectionColumns int           `json:"section_columns,omitempty"`
	SectionRows    int           `json:"section_rows,omitempty"`
	// DisableSidecar is used by scoped file hosts which load KJN through os.Root.
	DisableSidecar bool                `json:"-"`
	Settings       *DXFDrawingSettings `json:"settings,omitempty"`
	SettingsPath   string              `json:"-"`
}
type DXFPlacement struct {
	Page     int        `json:"page"`
	Layout   string     `json:"layout"`
	Title    string     `json:"title"`
	Kind     string     `json:"kind"`
	Section  int        `json:"section,omitempty"`
	Scale    float64    `json:"scale"`
	PaperBox [4]float64 `json:"paper_box_mm"`
	ModelBox [4]float64 `json:"model_box_mm"`
	Rotation int        `json:"rotation,omitempty"`
}
type DXFDocumentReport struct {
	Pages         int               `json:"pages"`
	Paper         string            `json:"paper"`
	Placements    []DXFPlacement    `json:"placements"`
	Sections      []DXFSection      `json:"sections,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
	DrawingIssues []DXFDrawingIssue `json:"drawing_issues,omitempty"`
	Drafting      *DXFDraftingStyle `json:"drafting"`
	SheetTemplate *DXFSheetTemplate `json:"sheet_template"`
	Sheets        []DXFSheetReport  `json:"sheets"`
}

func paperSize(name string) (float64, float64, error) {
	switch strings.ToUpper(name) {
	case "A3":
		return 420, 297, nil
	case "A4":
		return 297, 210, nil
	case "A3P":
		return 297, 420, nil
	case "A4P":
		return 210, 297, nil
	}
	return 0, 0, fmt.Errorf("paper must be A3, A4, A3P or A4P")
}
func finiteDXF(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }
func dxfBounds(es entity.Entities) ([4]float64, error) {
	b := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	if len(es) == 0 {
		return b, fmt.Errorf("empty drawing")
	}
	for _, e := range es {
		lo, hi := e.BBox()
		for i := 0; i < 2; i++ {
			if !finiteDXF(lo[i]) || !finiteDXF(hi[i]) {
				return b, fmt.Errorf("non-finite drawing geometry")
			}
			b[i] = math.Min(b[i], lo[i])
			b[2+i] = math.Max(b[2+i], hi[i])
		}
	}
	return b, nil
}

func dxfWrapped(s string, maxUnits int) []string {
	var out []string
	line := ""
	units := 0
	for _, r := range s {
		u := 1
		if r > 127 {
			u = 2
		}
		if r == '\n' || units+u > maxUnits {
			out = append(out, line)
			line = ""
			units = 0
			if r == '\n' {
				continue
			}
		}
		line += string(r)
		units += u
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}

func buildSectionCard(r DXFSection, scale, width, height float64, font string, styles ...*DXFDraftingStyle) (*drawing.Drawing, error) {
	d := dxf.NewDrawing()
	drafting := DefaultDXFDraftingStyle()
	if len(styles) > 0 {
		drafting = styles[0]
	}
	if font == "" {
		font = "msgothic.ttc"
	}
	d.AddStyle("ST_DRAWING", font, "", true)
	d.Header().CodePage = codepage.Japanese
	d.Header().InsUnit = insunit.Millimeters
	d.AddLayer("SECTION_CARD", dxf.DefaultColor, dxf.DefaultLineType, true)
	d.LwPolyline(true, []float64{0, 0}, []float64{width * scale, 0}, []float64{width * scale, height * scale}, []float64{0, height * scale})
	texts := []string{fmt.Sprintf("%s [%d] %s", r.Symbol, r.ID, r.Kind), "使用階: " + strings.Join(r.Floors, ", "), "材料: " + r.Material, "断面: " + r.Shape}
	if r.MainBars != "" {
		texts = append(texts, "主筋: "+r.MainBars)
	}
	if r.Hoops != "" {
		texts = append(texts, "帯筋/あばら筋: "+r.Hoops)
	}
	if len(r.Warnings) > 0 {
		texts = append(texts, fmt.Sprintf("要確認: %d項目（出力レポート参照）", len(r.Warnings)))
	}
	y := height - 5
	for _, s := range texts {
		for _, line := range sheetTextLines(s, width-4, drafting.TextHeightMM, drafting.TextWidthFactor) {
			d.Text(line, 4*scale, y*scale, 0, drafting.TextHeightMM*scale)
			y -= drafting.TextHeightMM * 1.6
		}
	}
	availableH := y - 7
	availableW := width - 12
	if availableH < 15 {
		return nil, fmt.Errorf("section %d: metadata does not fit; reduce section rows/columns or use larger paper", r.ID)
	}
	var geom entity.Entities
	d.AddLayer("SECTION_SHAPE", dxf.DefaultColor, dxf.DefaultLineType, true)
	var path [][]float64
	flush := func() {
		if len(path) > 1 {
			p, _ := d.LwPolyline(true, path...)
			geom = append(geom, p)
		}
		path = nil
	}
	for _, v := range r.vertices {
		if v == nil {
			flush()
		} else {
			path = append(path, v)
		}
	}
	flush()
	for _, c := range r.circles {
		if c[2] <= 0 || !finiteDXF(c[2]) {
			return nil, fmt.Errorf("section %d: invalid circle", r.ID)
		}
		e, _ := d.Circle(c[0], c[1], 0, c[2])
		geom = append(geom, e)
	}
	d.AddLayer("SECTION_REBAR", dxf.DefaultColor, dxf.DefaultLineType, true)
	for _, bar := range r.Bars {
		if !finiteDXF(bar.X) || !finiteDXF(bar.Y) || !finiteDXF(bar.Radius) || bar.Radius <= 0 {
			return nil, fmt.Errorf("section %d: invalid bar geometry", r.ID)
		}
		e, _ := d.Circle(bar.X, bar.Y, 0, bar.Radius)
		geom = append(geom, e)
	}
	if len(geom) > 0 {
		box, err := dxfBounds(geom)
		if err != nil {
			return nil, err
		}
		if box[2]-box[0] > availableW*scale || box[3]-box[1] > availableH*scale {
			return nil, fmt.Errorf("section %d: section figure does not fit at 1:%g", r.ID, scale)
		}
		if err = d.Move(width*scale/2-(box[0]+box[2])/2, availableH*scale/2+3*scale-(box[1]+box[3])/2, 0, geom...); err != nil {
			return nil, err
		}
	} else {
		d.Layer("SECTION_CARD", true)
		d.Text("断面図なし（定義不足/未対応）", 4*scale, 10*scale, 0, 2.5*scale)
	}
	applyDXFDraftingEntities(d, drafting, scale)
	if bounds, err := dxfBounds(d.Entities()); err != nil {
		return nil, err
	} else if bounds[0] < -1e-6 || bounds[1] < -1e-6 || bounds[2] > width*scale+1e-6 || bounds[3] > height*scale+1e-6 {
		return nil, fmt.Errorf("section %d: metadata exceeds its card", r.ID)
	}
	return d, nil
}

// BuildDXFDocument composes model-space view blocks and paper-space sheets.
// Geometry stays in mm; only viewports set plot scale. No source model edits.
func (f *Frame) BuildDXFDocument(o DXFDocumentOptions) (*drawing.Drawing, *DXFDocumentReport, error) {
	if o.SettingsPath != "" && (o.Settings != nil || o.DisableSidecar) {
		return nil, nil, fmt.Errorf("choose explicit settings or an allowed settings path")
	}

	settings := o.Settings
	var err error
	if settings == nil && !o.DisableSidecar {
		settings, err = f.loadDXFDrawingSettings(o.SettingsPath)
		if err != nil {
			return nil, nil, err
		}
	}
	var draftingInput *DXFDraftingStyle
	var sheetInput *DXFSheetTemplate
	if settings != nil {
		draftingInput = settings.Style
		sheetInput = settings.Sheet
	}
	drafting, err := ResolveDXFDraftingStyle(draftingInput)
	if err != nil {
		return nil, nil, err
	}
	sheet, err := ResolveDXFSheetTemplate(sheetInput)
	if err != nil {
		return nil, nil, err
	}
	if o.Font == "" {
		o.Font = drafting.Font
	} else {
		drafting.Font = o.Font
	}
	if o.Paper == "" {
		o.Paper = sheet.Paper
	}
	o.Paper = strings.ToUpper(o.Paper)
	sheet.Paper = o.Paper
	if o.Title != "" {
		sheet.Project = o.Title
	}
	w, h, err := paperSize(o.Paper)
	if err != nil {
		return nil, nil, err
	}
	if o.Columns == 0 {
		o.Columns = 1
	}
	if o.Rows == 0 {
		o.Rows = 1
	}
	if o.SectionColumns == 0 {
		o.SectionColumns = 3
	}
	if o.SectionRows == 0 {
		o.SectionRows = 2
	}
	for _, n := range []int{o.Columns, o.Rows, o.SectionColumns, o.SectionRows} {
		if n < 1 || n > 4 {
			return nil, nil, fmt.Errorf("layout rows and columns must be between 1 and 4")
		}
	}
	if !finiteDXF(o.Scale) || o.Scale < 0 || o.Scale > 10000 {
		return nil, nil, fmt.Errorf("invalid scale")
	}
	if len(o.Views) > 100 || len(o.SectionIDs) > 500 {
		return nil, nil, fmt.Errorf("too many drawing requests")
	}
	if len(o.Views) == 0 && !o.Sections {
		return nil, nil, fmt.Errorf("request at least one view or section list")
	}
	if len(o.SectionIDs) > 0 && !o.Sections {
		return nil, nil, fmt.Errorf("enable sections to use section_ids")
	}
	r := f.Snapshot()
	if !o.DisableSidecar {
		r, err = f.dimensionFrame()
		if err != nil {
			return nil, nil, err
		}
	}
	var settingsReport *DXFDrawingCheckReport
	if settings != nil {
		settingsReport, err = r.applyDXFDrawingSettings(settings)
		if err != nil {
			return nil, nil, err
		}
	}
	if r.drawingSettings == nil {
		r.drawingSettings = &DXFDrawingSettings{Version: 1}
	}
	r.drawingSettings.Style = drafting
	contentBox, _, err := sheet.boxes(w, h)
	if err != nil {
		return nil, nil, err
	}
	contentW, contentH := contentBox[2]-contentBox[0], contentBox[3]-contentBox[1]
	// Downstream rendering must use only these already-resolved in-memory grids.
	r.Path = ""
	// Floor/grid specifications, not a temporary GUI display filter, define
	// a document. Reset visibility only on this private snapshot.
	r.Show = NewShow(r)
	for _, n := range r.Nodes {
		n.Show()
	}
	for _, e := range r.Elems {
		e.Show()
	}
	r.Show.Xrange = []float64{math.Inf(-1), math.Inf(1)}
	r.Show.Yrange = []float64{math.Inf(-1), math.Inf(1)}
	r.Show.Zrange = []float64{math.Inf(-1), math.Inf(1)}
	report := &DXFDocumentReport{Paper: o.Paper, Drafting: drafting, SheetTemplate: sheet}
	if settingsReport != nil {
		report.DrawingIssues = settingsReport.Issues
		for _, i := range settingsReport.Issues {
			if !o.Sections || i.Code != "section_data" && i.Code != "inp_lst_mismatch" {
				report.Warnings = append(report.Warnings, i.Message)
			}
		}
	}
	if o.Sections {
		report.Sections, err = r.DXFSections(o.SectionIDs)
		if err != nil {
			return nil, nil, err
		}
		if len(report.Sections) > 500 {
			return nil, nil, fmt.Errorf("too many sections; select section_ids")
		}
		for _, s := range report.Sections {
			for _, warning := range s.Warnings {
				report.Warnings = append(report.Warnings, fmt.Sprintf("断面%d: %s", s.ID, warning))
			}
		}
		if len(report.Sections) == 0 {
			return nil, nil, fmt.Errorf("no sections selected")
		}
	}
	d := dxf.NewDrawing()
	d.AddStyle("ST_DRAWING", o.Font, "", true)
	d.Header().CodePage = codepage.Japanese
	d.Header().InsUnit = insunit.Millimeters
	d.AddLayer("SHEET_FRAME", dxf.DefaultColor, dxf.DefaultLineType, false)
	d.AddLayer("SHEET_TEXT", dxf.DefaultColor, dxf.DefaultLineType, false)
	d.AddLayer("SHEET_VIEWPORT", dxf.DefaultColor, dxf.DefaultLineType, false)
	// A dedicated non-plot layer keeps viewport boundary rectangles off prints.
	d.Layers["SHEET_VIEWPORT"].Extra = append(d.Layers["SHEET_VIEWPORT"].Extra, [2]string{"290", "0"})
	tmp, err := os.MkdirTemp("", "st-dxf-document-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(tmp)
	page := 0
	modelX := 0.
	newPage := func() (string, error) {
		page++
		name := fmt.Sprintf("S%03d", page)
		if page == 1 {
			if e := d.RenameLayout("Layout1", name); e != nil {
				return "", e
			}
		} else {
			if _, e := d.AddLayout(name); e != nil {
				return "", e
			}
		}
		l, _ := d.Layout(name)
		l.LimMin = []float64{0, 0}
		l.LimMax = []float64{w, h}
		l.PageSetup(w, h, object.PLOT_MILLIMETERS, [4]float64{0, 0, 0, 0}, 0, 1, "")
		l.PlotSettings.Flags |= 128      // print entity/layer lineweights
		l.PlotSettings.Flags &^= 32 | 64 // no CTB override or scaling of lineweights
		l.PlotSettings.StyleSheet = ""
		if e := d.UseLayout(name); e != nil {
			return "", e
		}

		return name, nil
	}
	place := func(src *drawing.Drawing, name, title, kind string, section, slot, cols, rows int, scale float64, box [4]float64, rotation int) error {
		cellW := (contentW - 5*float64(cols-1)) / float64(cols)
		cellH := (contentH - 5*float64(rows-1)) / float64(rows)
		x := contentBox[0] + float64(slot%cols)*(cellW+5)
		y := contentBox[3] - float64(slot/cols+1)*cellH - float64(slot/cols)*5
		viewW, viewH := cellW-4, cellH-12
		index := len(report.Placements) + 1
		// Anonymous dimension block names are local to each source drawing.
		// Give them unique names before import so no view reuses another's text.
		for _, b := range append(src.Blocks()[:0:0], src.Blocks()...) {
			if strings.HasPrefix(b.Name, "*D") {
				old := b.Name
				newName := fmt.Sprintf("ST_DIM_%d_%s", index, strings.TrimPrefix(old, "*"))
				if e := src.RenameBlock(old, newName); e != nil {
					return e
				}
				b.Flag &^= 1
				for _, e := range src.Entities() {
					if dim, ok := e.(*entity.Dimension); ok && dim.BlockName == old {
						dim.BlockName = newName
					}
				}
			}
		}
		if style, e := src.DimStyle("Standard"); e == nil {
			newName := fmt.Sprintf("ST_DIMSTYLE_%d", index)
			style.SetName(newName)
			// Imported symbol records must receive destination handles, rather
			// than retain the same source handle from every temporary drawing.
			style.SetSourceHandle("")
			for _, e := range src.Entities() {
				if dim, ok := e.(*entity.Dimension); ok {
					dim.Style = newName
				}
			}
		}
		if style, e := src.Style("ST_DRAWING", false); e == nil {
			style.FontName = o.Font
			style.BigFontName = ""
		}
		// Paper-space viewports perform linetype scaling (PSLTSCALE=1).
		// Source single-view DXFs have already encoded their print scale in
		// group 48; reset it here to prevent scaling the pattern twice.
		for _, e := range src.Entities() {
			e.SetLtscale(1)
		}
		for _, b := range src.Blocks() {
			for _, e := range b.Entities {
				e.SetLtscale(1)
			}
		}
		d.ModelSpace()
		blockName := fmt.Sprintf("ST_VIEW_%03d", index)
		if _, e := d.BeginBlock(blockName, 0, 0, 0); e != nil {
			return e
		}
		if _, e := dxf.ImportEntities(d, src, src.ModelSpaceEntities()...); e != nil {
			return e
		}
		if e := d.EndBlock(); e != nil {
			return e
		}
		// A model-space INSERT must be on a plotting layer. Inheriting the
		// preceding paper-space viewport layer can hide the entire block at print.
		d.Layer("0", true)
		if _, e := d.Insert(blockName, modelX, 0, 0); e != nil {
			return e
		}
		if e := d.UseLayout(name); e != nil {
			return e
		}
		d.Layer("SHEET_TEXT", true)
		label := fmt.Sprintf("%s  1:%g", title, scale)
		if len(dxfWrapped(label, int((cellW-4)/1.8))) > 1 || utf8.RuneCountInString(label) == 0 {
			return fmt.Errorf("view title is too long for its cell")
		}
		d.Text(label, x+2, y+cellH-5, 0, 3)
		d.Layer("SHEET_VIEWPORT", true)
		v, e := d.Viewport([]float64{x + cellW/2, y + 2 + viewH/2, 0}, viewW, viewH, []float64{modelX + (box[0]+box[2])/2, (box[1] + box[3]) / 2}, viewH*scale)
		if e != nil {
			return e
		}
		v.Flags |= 16384 // lock scale/position
		if rotation != 0 {
			v.ViewTarget = []float64{modelX + (box[0]+box[2])/2, (box[1] + box[3]) / 2, 0}
			v.ViewCenter = []float64{0, 0}
			v.TwistAngle = -float64(rotation)
		}
		report.Placements = append(report.Placements, DXFPlacement{page, name, title, kind, section, scale, [4]float64{x, y, x + cellW, y + cellH}, box, rotation})
		// Keep adjacent view geometry outside all of this viewport's field of view.
		extent := math.Max(viewW, viewH) * scale
		modelX += math.Max(box[2]-box[0], extent) + math.Max(extent, 100000)
		return nil
	}
	var name string
	for i, spec := range o.Views {
		rotation := 0
		key := spec.Kind + ":" + spec.Grid
		if spec.Kind == "plan" {
			key = fmt.Sprintf("plan:%d", spec.Floor)
		}
		if v, ok := sheet.ViewRotations[strings.ToLower(key)]; ok {
			rotation = v
		}
		if spec.Rotation != nil {
			rotation = *spec.Rotation
		}
		if rotation != 0 && rotation != 90 && rotation != 180 && rotation != 270 {
			return nil, nil, fmt.Errorf("view rotation must be 0,90,180 or 270")
		}
		if i%(o.Columns*o.Rows) == 0 {
			name, err = newPage()
			if err != nil {
				return nil, nil, err
			}
		}
		cellW := (contentW - 5*float64(o.Columns-1)) / float64(o.Columns)
		cellH := (contentH - 5*float64(o.Rows-1)) / float64(o.Rows)
		scales := []float64{10, 20, 25, 50, 75, 100, 150, 200, 250, 500, 1000}
		if o.Scale > 0 {
			scales = []float64{o.Scale}
		}
		var src *drawing.Drawing
		var box [4]float64
		chosen := 0.
		unresolved := 0
		title := spec.Title
		r.drawingRotation = rotation
		r.drawingView = spec.Kind + ":" + spec.Grid
		if spec.Kind == "plan" {
			r.drawingView = fmt.Sprintf("plan:%d", spec.Floor)
		}
		for _, s := range scales {
			p := filepath.Join(tmp, "view.dxf")
			switch spec.Kind {
			case "plan":
				err = r.WriteDxfPlanDimensions(p, spec.Floor, 1000, 2.5*s, 3*s)
				if title == "" {
					title = fmt.Sprintf("%d階伏図", spec.Floor)
				}
			case "elevation":
				axis, c, e := r.resolveDxfJiku(spec.Grid)
				if e != nil {
					return nil, nil, e
				}
				err = r.WriteDxfCrosssectionDimensions(p, axis, c-.1, c+.1, 1, 1000, 2.5*s, 3*s)
				if title == "" {
					title = spec.Grid + "通り軸組図"
				}
			default:
				return nil, nil, fmt.Errorf("unknown view kind %q", spec.Kind)
			}
			if err != nil {
				return nil, nil, err
			}
			src, err = dxf.FromFile(p)
			if err != nil {
				return nil, nil, err
			}
			unresolved = arrangeDXFMemberLabels(src)
			box, err = dxfBounds(src.ModelSpaceEntities())
			if err != nil {
				return nil, nil, err
			}
			viewWidth, viewHeight := box[2]-box[0], box[3]-box[1]
			if rotation%180 != 0 {
				viewWidth, viewHeight = viewHeight, viewWidth
			}
			if viewWidth/s <= cellW-6 && viewHeight/s <= cellH-14 {
				chosen = s
				break
			}
		}
		if chosen == 0 {
			return nil, nil, fmt.Errorf("view %s does not fit the sheet at requested scales", title)
		}
		if unresolved > 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s: %d件の部材符号の重なりを解消できませんでした", title, unresolved))
		}
		if err = place(src, name, title, spec.Kind, 0, i%(o.Columns*o.Rows), o.Columns, o.Rows, chosen, box, rotation); err != nil {
			return nil, nil, err
		}
	}
	for i, section := range report.Sections {
		if i%(o.SectionColumns*o.SectionRows) == 0 {
			name, err = newPage()
			if err != nil {
				return nil, nil, err
			}
		}
		cw := (contentW - 5*float64(o.SectionColumns-1)) / float64(o.SectionColumns)
		ch := (contentH - 5*float64(o.SectionRows-1)) / float64(o.SectionRows)
		var src *drawing.Drawing
		chosen := 0.
		for _, s := range []float64{10, 20, 25, 50, 75, 100} {
			src, err = buildSectionCard(section, s, cw-6, ch-14, o.Font, drafting)
			if err == nil {
				chosen = s
				break
			}
		}
		if chosen == 0 {
			return nil, nil, err
		}
		box, e := dxfBounds(src.Entities())
		if e != nil {
			return nil, nil, e
		}
		if err = place(src, name, section.Symbol, "section", section.ID, i%(o.SectionColumns*o.SectionRows), o.SectionColumns, o.SectionRows, chosen, box, 0); err != nil {
			return nil, nil, err
		}
	}
	seenNumbers := map[string]bool{}
	for p := 1; p <= page; p++ {
		if err = d.UseLayout(fmt.Sprintf("S%03d", p)); err != nil {
			return nil, nil, err
		}
		metadata, e := drawDXFSheet(d, sheet, drafting, w, h, p, o.Title, report.Placements)
		if e != nil {
			return nil, nil, e
		}
		if seenNumbers[metadata.Number] {
			return nil, nil, fmt.Errorf("duplicate sheet number %s", metadata.Number)
		}
		seenNumbers[metadata.Number] = true
		report.Sheets = append(report.Sheets, metadata)
	}
	for p := range sheet.Pages {
		if p > page {
			report.Warnings = append(report.Warnings, fmt.Sprintf("図枠のページ指定 %d は出力ページ数 %d を超えています", p, page))
		}
	}
	applyDXFDraftingLayers(d, drafting)
	for i, p := range d.Header().Extra {
		if p[0] == "9" && p[1] == "$PSLTSCALE" {
			d.Header().Extra[i+1] = [2]string{"70", "1"}
		}
	}
	report.Pages = page
	d.ModelSpace()
	d.SetExt()
	return d, report, nil
}

func (f *Frame) WriteDXFDocument(path string, o DXFDocumentOptions) (*DXFDocumentReport, error) {
	d, report, err := f.BuildDXFDocument(o)
	if err != nil {
		return nil, err
	}
	if err = d.SaveAs(path); err != nil {
		return nil, err
	}
	return report, nil
}
