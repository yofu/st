package st

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yofu/dxf/drawing"
)

type DXFSheetTemplate struct {
	ViewRotations map[string]int       `json:"view_rotations,omitempty"`
	Layout        string               `json:"layout,omitempty"` // right or bottom
	Paper         string               `json:"paper,omitempty"`
	MarginMM      float64              `json:"margin_mm,omitempty"`
	TitleWidthMM  float64              `json:"title_width_mm,omitempty"`
	Office        string               `json:"office,omitempty"`
	Structure     string               `json:"structure,omitempty"`
	Project       string               `json:"project,omitempty"`
	Date          string               `json:"date,omitempty"`
	DrawnBy       string               `json:"drawn_by,omitempty"`
	CheckedBy     string               `json:"checked_by,omitempty"`
	NumberPrefix  string               `json:"number_prefix,omitempty"`
	StartNumber   int                  `json:"start_number,omitempty"`
	NumberDigits  int                  `json:"number_digits,omitempty"`
	Notes         []string             `json:"notes,omitempty"`
	Revisions     []DXFSheetRevision   `json:"revisions,omitempty"`
	Pages         map[int]DXFSheetPage `json:"pages,omitempty"`
}
type DXFSheetRevision struct {
	Mark        string `json:"mark"`
	Date        string `json:"date,omitempty"`
	Description string `json:"description"`
	By          string `json:"by,omitempty"`
}
type DXFSheetPage struct {
	Title  string `json:"title,omitempty"`
	Number string `json:"number,omitempty"`
}
type DXFSheetReport struct {
	Page       int        `json:"page"`
	Number     string     `json:"number"`
	Title      string     `json:"title"`
	Scale      string     `json:"scale"`
	ContentBox [4]float64 `json:"content_box_mm"`
	TitleBox   [4]float64 `json:"title_box_mm"`
}

func DefaultDXFSheetTemplate() *DXFSheetTemplate {
	return &DXFSheetTemplate{Layout: "right", Paper: "A3", MarginMM: 10, TitleWidthMM: 45, NumberPrefix: "S-", StartNumber: 1, NumberDigits: 2}
}
func ResolveDXFSheetTemplate(input *DXFSheetTemplate) (*DXFSheetTemplate, error) {
	s := DefaultDXFSheetTemplate()
	if input == nil {
		return s, nil
	}
	if input.Layout != "" {
		s.Layout = input.Layout
	}
	if s.Layout != "right" && s.Layout != "bottom" {
		return nil, fmt.Errorf("sheet layout must be right or bottom")
	}
	if input.Paper != "" {
		s.Paper = strings.ToUpper(input.Paper)
	}
	if _, _, err := paperSize(s.Paper); err != nil {
		return nil, err
	}
	if input.MarginMM != 0 {
		s.MarginMM = input.MarginMM
	}
	if !finiteDXF(s.MarginMM) || s.MarginMM < 2 || s.MarginMM > 30 {
		return nil, fmt.Errorf("sheet margin must be 2..30 mm")
	}
	if input.TitleWidthMM != 0 {
		s.TitleWidthMM = input.TitleWidthMM
	}
	if !finiteDXF(s.TitleWidthMM) || s.TitleWidthMM < 25 || s.TitleWidthMM > 80 {
		return nil, fmt.Errorf("title width must be 25..80 mm")
	}
	s.Office, s.Structure, s.Project, s.Date, s.DrawnBy, s.CheckedBy = input.Office, input.Structure, input.Project, input.Date, input.DrawnBy, input.CheckedBy
	if input.NumberPrefix != "" {
		s.NumberPrefix = input.NumberPrefix
	}
	if input.StartNumber != 0 {
		s.StartNumber = input.StartNumber
	}
	if s.StartNumber < 1 || s.StartNumber > 999999 {
		return nil, fmt.Errorf("invalid sheet start number")
	}
	if input.NumberDigits != 0 {
		s.NumberDigits = input.NumberDigits
	}
	if s.NumberDigits < 1 || s.NumberDigits > 6 {
		return nil, fmt.Errorf("invalid sheet number digits")
	}
	s.Notes = append([]string(nil), input.Notes...)
	s.Revisions = append([]DXFSheetRevision(nil), input.Revisions...)
	if len(s.Revisions) > 10 || len(s.Notes) > 50 {
		return nil, fmt.Errorf("too many sheet revisions/notes")
	}
	for _, r := range s.Revisions {
		if strings.TrimSpace(r.Mark) == "" || strings.TrimSpace(r.Description) == "" {
			return nil, fmt.Errorf("sheet revisions require mark and description")
		}
	}
	s.ViewRotations = map[string]int{}
	for key, angle := range input.ViewRotations {
		if (!strings.HasPrefix(key, "plan:") && !strings.HasPrefix(key, "elevation:")) || (angle != 0 && angle != 90 && angle != 180 && angle != 270) {
			return nil, fmt.Errorf("invalid view rotation %q", key)
		}
		normalized := strings.ToLower(key)
		if _, exists := s.ViewRotations[normalized]; exists {
			return nil, fmt.Errorf("duplicate view rotation %q", key)
		}
		s.ViewRotations[normalized] = angle
	}
	s.Pages = make(map[int]DXFSheetPage, len(input.Pages))
	for id, v := range input.Pages {
		if id < 1 || id > 1000 {
			return nil, fmt.Errorf("invalid sheet override page %d", id)
		}
		s.Pages[id] = v
	}
	return s, nil
}
func (s *DXFSheetTemplate) boxes(w, h float64) ([4]float64, [4]float64, error) {
	m := s.MarginMM
	content, title := [4]float64{m, m, w - m, h - m}, [4]float64{w - m - s.TitleWidthMM, m, w - m, h - m}
	if s.Layout == "right" {
		content[2] = title[0] - 5
	} else {
		title = [4]float64{m, m, w - m, m + 25}
		content[1] = m + 30
	}
	if content[2]-content[0] < 40 || content[3]-content[1] < 40 {
		return content, title, fmt.Errorf("sheet template leaves too little drawing space")
	}
	return content, title, nil
}

// Use a conservative full-glyph width, including the cap-height/em ratio
// of Japanese TrueType fonts. CAD text metrics vary; wrapping early is safer
// than writing beyond a title box or cropping a card in its viewport.
func sheetTextLines(value string, width, height, factor float64) []string {
	if value == "" {
		return nil
	}
	capacity := int((width - 4) / (1.4 * height * factor))
	if capacity < 1 {
		capacity = 1
	}
	var lines []string
	var line []rune
	for _, r := range value {
		if r == '\n' {
			lines = append(lines, string(line))
			line = nil
			continue
		}
		if len(line) >= capacity {
			split := -1
			for i, c := range line {
				if c == ' ' && i >= capacity/2 {
					split = i
				}
			}
			if split >= 0 {
				lines = append(lines, string(line[:split]))
				line = append([]rune(nil), line[split+1:]...)
			} else {
				lines = append(lines, string(line))
				line = nil
			}
		}
		line = append(line, r)
	}
	if len(line) > 0 {
		lines = append(lines, string(line))
	}
	return lines
}

// Every metadata field is wrapped to its box. Nothing is silently truncated.
func drawDXFSheet(d *drawing.Drawing, s *DXFSheetTemplate, style *DXFDraftingStyle, w, h float64, page int, project string, placements []DXFPlacement) (DXFSheetReport, error) {
	content, title, err := s.boxes(w, h)
	if err != nil {
		return DXFSheetReport{}, err
	}
	titles := []string{}
	scales := []float64{}
	seenTitles := map[string]bool{}
	seenScales := map[float64]bool{}
	sectionsOnly := true
	for _, p := range placements {
		if p.Page != page {
			continue
		}
		sectionsOnly = sectionsOnly && p.Kind == "section"
		if !seenTitles[p.Title] {
			titles = append(titles, p.Title)
			seenTitles[p.Title] = true
		}
		if !seenScales[p.Scale] {
			scales = append(scales, p.Scale)
			seenScales[p.Scale] = true
		}
	}
	caption := strings.Join(titles, " / ")
	if sectionsOnly {
		caption = "断面リスト"
	}
	if override := s.Pages[page]; override.Title != "" {
		caption = override.Title
	}
	number := fmt.Sprintf("%s%0*d", s.NumberPrefix, s.NumberDigits, s.StartNumber+page-1)
	if override := s.Pages[page]; override.Number != "" {
		number = override.Number
	}
	sort.Float64s(scales)
	scaleNames := []string{}
	for _, v := range scales {
		scaleNames = append(scaleNames, fmt.Sprintf("1:%g", v))
	}
	scaleLabel := strings.Join(scaleNames, " / ")
	if len(scales) > 1 {
		scaleLabel += " 各図参照"
	}
	result := DXFSheetReport{page, number, caption, scaleLabel, content, title}
	text := func(value string, box [4]float64, height float64) error {
		if value == "" {
			return nil
		}
		lines := sheetTextLines(value, box[2]-box[0], height, style.TextWidthFactor)
		step := height * 1.6
		if height+float64(len(lines)-1)*step > box[3]-box[1]-4 {
			return fmt.Errorf("sheet %d: metadata does not fit: %s", page, value)
		}
		d.Layer("SHEET_TEXT", true)
		for i, line := range lines {
			t, e := d.Text(line, box[0]+2, box[3]-2-height-float64(i)*step, 0, height)
			if e != nil {
				return e
			}
			t.WidthFactor = style.TextWidthFactor
		}
		return nil
	}
	line := func(x0, y0, x1, y1 float64) { d.Layer("SHEET_FRAME", true); d.Line(x0, y0, 0, x1, y1, 0) }
	m := s.MarginMM
	d.Layer("SHEET_FRAME", true)
	if _, err = d.LwPolyline(true, []float64{m, m}, []float64{w - m, m}, []float64{w - m, h - m}, []float64{m, h - m}); err != nil {
		return result, err
	}
	if project == "" {
		project = s.Project
	}
	if project == "" {
		project = "構造図"
	}
	if s.Layout == "bottom" {
		line(title[0], title[3], title[2], title[3])
		if err = text(project+" / "+caption, [4]float64{m, m + 10, w - m - 50, m + 25}, 3); err != nil {
			return result, err
		}
		if err = text(strings.TrimSpace(strings.Join([]string{s.Date, s.DrawnBy, s.CheckedBy}, "  ")), [4]float64{m, m, w - m - 50, m + 10}, 2); err != nil {
			return result, err
		}
		if err = text(number+"\n"+scaleLabel, [4]float64{w - m - 50, m, w - m, m + 25}, 2.5); err != nil {
			return result, err
		}
		// Do not silently omit revision notes for a compact template.
		if len(s.Revisions) > 0 || len(s.Notes) > 0 || s.Office != "" || s.Structure != "" {
			return result, fmt.Errorf("bottom template cannot accommodate office/structure/notes/revisions; use right layout")
		}
		return result, nil
	}
	line(title[0], m, title[0], h-m)
	width := title[2] - title[0]
	type row struct {
		label, value string
		height       float64
	}
	makeRow := func(label, value string, minHeight float64) row {
		n := len(sheetTextLines(value, width, 2.5, style.TextWidthFactor))
		height := 9 + float64(n)*4
		if height < minHeight {
			height = minHeight
		}
		return row{label, value, height}
	}
	revisions := []string{}
	for _, r := range s.Revisions {
		revisions = append(revisions, strings.TrimSpace(strings.Join([]string{r.Mark, r.Date, r.Description, r.By}, " ")))
	}
	rows := []row{
		makeRow("図番", number, 14),
		makeRow("改訂", strings.Join(revisions, "\n"), 14),
		makeRow("確認", s.CheckedBy, 10),
		makeRow("作図", s.DrawnBy, 10),
		makeRow("日付", s.Date, 10),
		makeRow("縮尺", scaleLabel, 14),
		makeRow("図名", caption, 18),
		makeRow("工事名", project, 18),
	}
	office := makeRow("設計", s.Office, 20)
	structure := makeRow("構造", s.Structure, 10)
	required := office.height + structure.height
	for _, r := range rows {
		required += r.height
	}
	if required > title[3]-title[1] {
		return result, fmt.Errorf("sheet %d: title metadata exceeds available height; use larger paper/title width", page)
	}
	y := title[1]
	renderRow := func(r row, bottom float64) error {
		box := [4]float64{title[0], bottom, title[2], bottom + r.height}
		line(box[0], box[3], box[2], box[3])
		if err := text(r.label, [4]float64{box[0], box[3] - 6, box[2], box[3]}, 1.6); err != nil {
			return err
		}
		return text(r.value, [4]float64{box[0], box[1], box[2], box[3] - 5}, 2.5)
	}
	for _, r := range rows {
		if err = renderRow(r, y); err != nil {
			return result, err
		}
		y += r.height
	}
	top := title[3]
	for _, r := range []row{office, structure} {
		top -= r.height
		if err = renderRow(r, top); err != nil {
			return result, err
		}
	}
	if err = text(strings.Join(s.Notes, "\n"), [4]float64{title[0], y, title[2], top}, 2.2); err != nil {
		return result, err
	}
	return result, nil
}
