package st

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sort"

	"strings"

	"github.com/signintech/gopdf"
)

const fig2PointsPerMM = 72.0 / 25.4

// WriteFig2PDF draws the document using the Go Fig2 and frame renderers.
func WriteFig2PDF(frame *Frame, source, output string) error {
	if frame == nil {
		return fmt.Errorf("Fig2 PDF requires an open model")
	}
	sheets, err := loadFig2Sheets(source, make(map[string]bool), frame)
	if err != nil {
		return err
	}
	if len(sheets) == 0 {
		return fmt.Errorf("Fig2 document has no pages")
	}
	if filepath.Clean(source) == filepath.Clean(output) || !strings.EqualFold(filepath.Ext(output), ".pdf") {
		return fmt.Errorf("Fig2 PDF output must have a .pdf extension")
	}
	size, err := fig2PaperSize(sheets[0].Paper)
	if err != nil {
		return err
	}
	p, err := newPDFCanvas(size, 0, 0, fig2PointsPerMM, false)
	if err != nil {
		return err
	}
	p.fig2 = true
	// The copy is reused so that sidecar reads and camera settings can
	// carry forward through the document without changing the live model.
	f := fig2RenderFrame(frame)
	f.View = frame.View.Copy()
	w := &fig2PDFWindow{frame: f, pdf: p, boxes: make(map[string]*TextBox)}
	baseMeasures := f.Measures
	for i, sheet := range sheets {
		size, err := fig2PaperSize(sheet.Paper)
		if err != nil {
			return fmt.Errorf("page %d: %w", i+1, err)
		}
		p.canvas.AddPageWithOption(gopdf.PageOption{PageSize: &size})
		p.canvas.SetLineWidth(0.25)
		width, height := size.W/fig2PointsPerMM, size.H/fig2PointsPerMM
		rows, columns := max(1, sheet.Rows), max(1, sheet.Columns)
		for j, view := range sheet.Views {
			if view.Kind == "BLANK" {
				continue
			}
			cellW, cellH := width/float64(columns), height/float64(rows)
			ox, oy := float64(view.Slot%columns)*cellW, float64(view.Slot/columns)*cellH
			n := 1
			if strings.HasPrefix(view.Kind, "FIGURE") {
				n = 3
			}
			camera := f.View.Copy()
			unit, unitName := slices.Clone(f.Show.Unit), slices.Clone(f.Show.UnitName)
			dfact, mfact := f.Show.Dfact, f.Show.Mfact
			for panel := 0; panel < n; panel++ {
				p.originX, p.originY, p.width, p.height = ox, oy, cellW, cellH
				if n == 3 {
					if view.Kind == "FIGURE_H" {
						p.width /= 3
						p.originX += float64(panel) * p.width
					} else {
						p.height /= 3
						p.originY += float64(panel) * p.height
					}
				}
				f.View = camera.Copy()
				f.Measures = baseMeasures
				w.boxes = make(map[string]*TextBox)
				resetFig2View(w)
				copy(f.Show.Unit, unit)
				copy(f.Show.UnitName, unitName)
				f.Show.Dfact, f.Show.Mfact = dfact, mfact
				// Preserve the previous camera values for underscore placeholders.
				f.View.Gfact, f.View.Perspective = camera.Gfact, camera.Perspective
				copy(f.View.Angle, camera.Angle)
				copy(f.View.Dists, camera.Dists)
				copy(f.View.Focus, camera.Focus)
				if err := applyFig2View(w, view.Source, fig2PanelCommands(view, panel)); err != nil {
					return fmt.Errorf("%s, page %d, view %d: %w", source, i+1, j+1, err)
				}
				if n == 3 {
					if panel > 0 {
						w.TextBox("PAGETITLE").Hide()
					}
					b := w.TextBox("TITLE")
					b.SetPosition(30, 15)
					units := f.Show.UnitName[0]
					if panel == 2 {
						units += f.Show.UnitName[1]
					}
					b.AddText([3]string{"軸力", "せん断力", "曲げモーメント"}[panel] + "[" + units + "]")
					for _, args := range view.Commands {
						if !strings.EqualFold(args[0], "elem") {
							continue
						}
						var names []string
						for _, t := range args[1:] {
							if name := map[string]string{"COLUMN": "柱", "GIRDER": "梁", "BRACE": "ブレース", "WALL": "壁", "SLAB": "床", "WBRACE": "壁ブレース", "SBRACE": "床ブレース"}[strings.ToUpper(t)]; name != "" {
								names = append(names, name)
							}
						}
						if len(names) > 0 {
							b.AddText("表示部材: " + strings.Join(names, "、"))
						}
					}
					b.Show()
				}
				w.mergeData()
				p.Draw(f, w.TextBoxes())
				if p.layoutErr != nil {
					return fmt.Errorf("%s, page %d, view %d, panel %d: %w", source, i+1, j+1, panel+1, p.layoutErr)
				}
			}
		}
		number := sheet.Number
		if number != "" && !sheet.NoNumber {
			p.originX, p.originY = 0, 0
			p.Foreground(BLACK)
			p.TextOrientation(0)
			p.SetFig2FontSize(10)
			p.TextAlignment(SOUTH_EAST)
			p.Text(width-10, height-10, number)
		}
	}
	return p.SaveAs(output)
}

func fig2PaperSize(paper string) (gopdf.Rect, error) {
	switch paper {
	case "A4", "A4TATE":
		return gopdf.Rect{W: 210 * fig2PointsPerMM, H: 297 * fig2PointsPerMM}, nil
	case "A4YOKO":
		return gopdf.Rect{W: 297 * fig2PointsPerMM, H: 210 * fig2PointsPerMM}, nil
	case "A3", "A3TATE":
		return gopdf.Rect{W: 297 * fig2PointsPerMM, H: 420 * fig2PointsPerMM}, nil
	case "A3YOKO":
		return gopdf.Rect{W: 420 * fig2PointsPerMM, H: 297 * fig2PointsPerMM}, nil
	default:
		return gopdf.Rect{}, fmt.Errorf("unknown Fig2 paper %q", paper)
	}
}

func fig2PanelCommands(view Fig2View, panel int) [][]string {
	if !strings.HasPrefix(view.Kind, "FIGURE") {
		return view.Commands
	}
	out := make([][]string, 0, len(view.Commands))
	for _, args := range view.Commands {
		if !strings.EqualFold(args[0], "stress") || len(args) < 4 {
			out = append(out, args)
			continue
		}
		target := strings.ToUpper(args[1])
		if panel > 0 && (target == "BRACE" || target == "WBRACE" || target == "SBRACE") {
			continue
		}
		component := []string{"N", "QY", "MX"}[panel]
		if len(args) > 3+panel {
			component = args[3+panel]
		}
		cmd := []string{args[0], args[1], args[2], component}
		for i, arg := range args[4:] {
			if strings.EqualFold(arg, "side") {
				cmd = append(cmd, args[i+4:]...)
				break
			}
		}
		out = append(out, cmd)
	}
	return append(out, []string{"nodata"}, []string{"nolegend"})
}

type fig2PDFWindow struct {
	Fig2Moder
	frame *Frame
	pdf   *PDFCanvas
	boxes map[string]*TextBox
}

func (w *fig2PDFWindow) Frame() *Frame                        { return w.frame }
func (w *fig2PDFWindow) Cwd() string                          { return filepath.Dir(w.frame.Path) }
func (w *fig2PDFWindow) EPS() float64                         { return 1e-6 }
func (w *fig2PDFWindow) History(string)                       {}
func (w *fig2PDFWindow) EnableLabel(string)                   {}
func (w *fig2PDFWindow) DisableLabel(string)                  {}
func (w *fig2PDFWindow) SetLabel(string, string)              {}
func (w *fig2PDFWindow) Redraw()                              {}
func (w *fig2PDFWindow) Deselect()                            {}
func (w *fig2PDFWindow) SetColorMode(c uint)                  { w.frame.Show.ColorMode = c }
func (w *fig2PDFWindow) DefaultColorMode() uint               { return ECOLOR_BLACKSECT }
func (w *fig2PDFWindow) CanvasDirection() int                 { return 1 }
func (w *fig2PDFWindow) GetCanvasSize() (int, int)            { return w.pdf.GetCanvasSize() }
func (w *fig2PDFWindow) AddSectionAlias(id int, value string) { w.pdf.AddSectionAlias(id, value) }
func (w *fig2PDFWindow) DeleteSectionAlias(id int)            { w.pdf.DeleteSectionAlias(id) }
func (w *fig2PDFWindow) ClearSectionAlias()                   { w.pdf.ClearSectionAlias() }

func (w *fig2PDFWindow) ShowCenter() {
	f, v := w.frame, w.frame.View
	f.SetFocus(nil)
	v.Set(w.CanvasDirection())
	for _, n := range f.Nodes {
		v.ProjectNode(n)
	}
	x0, x1, y0, y1 := f.Bbox2D(true)
	availableW, availableH := max(1, w.pdf.width-20), max(1, w.pdf.height-20)
	dx, dy := x1-x0, y1-y0
	if dx <= 0 && dy <= 0 {
		return
	}
	factor := min(availableW/max(1e-9, dx), availableH/max(1e-9, dy))
	v.Center[0] += factor * (v.Center[0] - (x0+x1)/2)
	v.Center[1] += factor * (v.Center[1] - (y0+y1)/2)
	v.Gfact *= factor
}

func (w *fig2PDFWindow) TextBox(name string) *TextBox {
	if b := w.boxes[name]; b != nil {
		return b
	}
	b := NewTextBox(&fig2PDFFont{size: 10, color: BLACK})
	switch name {
	case "PAGETITLE":
		b.Font.SetSize(16)
		b.SetPosition(30, w.pdf.height-30)
	case "TITLE":
		b.SetPosition(35, w.pdf.height-55)
	case "DATA":
		b.SetPosition(35, 65)
	case "INPUTFILE":
		b.SetPosition(12, 20)
	case "OUTPUTFILE":
		b.SetPosition(12, 15)
	case "VIEWPOINT":
		b.SetPosition(w.pdf.width-48, 26)
	case "TEXT":
		b.SetPosition(w.pdf.width/2+40, 65)
	default:
		b.SetPosition(w.pdf.width/2+60, 65)
	}
	w.boxes[name] = b
	return b
}

func (w *fig2PDFWindow) TextBoxes() []*TextBox {
	names := make([]string, 0, len(w.boxes))
	for name := range w.boxes {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*TextBox, 0, len(names))
	for _, name := range names {
		out = append(out, w.boxes[name])
	}
	return out
}

type fig2PDFFont struct {
	face        string
	size, color int
}

func (f *fig2PDFFont) Face() string        { return f.face }
func (f *fig2PDFFont) Size() int           { return f.size }
func (f *fig2PDFFont) Color() int          { return f.color }
func (f *fig2PDFFont) SetFace(face string) { f.face = face }
func (f *fig2PDFFont) SetSize(size int)    { f.size = size }
func (f *fig2PDFFont) SetColor(color int)  { f.color = color }

func fig2RenderFrame(source *Frame) *Frame {
	f := source.Snapshot()
	f.Fes = source.Fes
	f.fig2JointRates = source.fig2JointRates
	clear(f.Kijuns)
	for name, k := range source.Kijuns {
		f.Kijuns[strings.ToLower(name)] = k.Snapshot()
	}
	for id, src := range source.Elems {
		e := f.Elems[id]
		if e == src {
			copy := *src
			copy.Frame = f
			copy.Sect = f.Sects[src.Sect.Num]
			copy.Enod = make([]*Node, len(src.Enod))
			for i, n := range src.Enod {
				copy.Enod[i] = f.Nodes[n.Num]
			}
			e = &copy
			f.Elems[id] = e
		}
		e.Stress = make(map[string]map[int][]float64, len(src.Stress))
		for period, ends := range src.Stress {
			copied := make(map[int][]float64, len(ends))
			for id, v := range ends {
				copied[id] = slices.Clone(v)
			}
			e.Stress[period] = copied
		}
		e.Rate = maps.Clone(src.Rate)
		e.MaxRate = slices.Clone(src.MaxRate)
		e.Values = maps.Clone(src.Values)
		e.Phinge = make(map[string]map[int]bool, len(src.Phinge))
		for period, ends := range src.Phinge {
			e.Phinge[period] = maps.Clone(ends)
		}
	}
	for id, src := range source.Elems {
		e := f.Elems[id]
		if src.Parent != nil {
			e.Parent = f.Elems[src.Parent.Num]
		}
		e.Children = make([]*Elem, len(src.Children))
		for i, ch := range src.Children {
			if ch != nil {
				e.Children[i] = f.Elems[ch.Num]
			}
		}
	}
	return f
}

func (w *fig2PDFWindow) mergeData() {
	data := w.TextBox("DATA")
	lines := append([]string(nil), data.Text()...)
	lines = append(lines, w.frame.Show.fig2ValueLabels...)
	for _, name := range []string{"INPUTFILE", "OUTPUTFILE", "VIEWPOINT"} {
		if b := w.boxes[name]; b != nil && !b.IsHidden(w.frame.Show) {
			lines = append(lines, b.Text()...)
			b.Hide()
		}
	}
	if len(lines) > 0 {
		data.SetText(lines)
		data.Show()
	}
}
