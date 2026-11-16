package st

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/text/encoding/japanese"
)

func TestFig2DocumentExpansionAndLayout(t *testing.T) {
	dir := t.TempDir()
	included := filepath.Join(dir, "included.fig2")
	if err := os.WriteFile(included, []byte("PAGE A3YOKO\nTITLE included\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "drawing.fig2")
	data, err := japanese.ShiftJIS.NewEncoder().String(`COMMON
CONST axis Z
IOTA seq 7
GFACT 1
LOAD included
FOR a,b in [1,2]*[3,4]
PAGE A4TATE
TITLE 図$a-$b-$seq
$axis 0 $b
ENDFOR
MULTIPLE 2 1
PAGE A4YOKO ESCAPE
TITLE 上
PAGE A4YOKO ESCAPE
TITLE 下
NOFEED
PAGE A4YOKO ESCAPE
TITLE 重ね描き
PAGE A4TATE
BLANK
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	sheets, err := LoadFig2Sheets(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(sheets) != 5 || sheets[0].Paper != "A3YOKO" || sheets[4].Views[0].Kind != "BLANK" {
		t.Fatalf("unexpected sheets %#v", sheets)
	}
	v := sheets[1].Views[0]
	if got := strings.Join(v.Commands[1], " "); got != "TITLE 図1-3-7" {
		t.Fatalf("title = %q", got)
	}
	if got := strings.Join(v.Commands[2], " "); got != "Z 0 3" {
		t.Fatalf("filter = %q", got)
	}
	sheet := sheets[3]
	if len(sheet.Views) != 3 || sheet.Rows != 2 || sheet.Views[0].Slot != 0 || sheet.Views[1].Slot != 1 || sheet.Views[2].Slot != 1 || !sheet.NoNumber {
		t.Fatalf("multiple/overlay layout = %#v", sheet)
	}
	if err := os.WriteFile(included, []byte("LOAD drawing\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFig2Sheets(path); err == nil || !strings.Contains(err.Error(), "recursive") {
		t.Fatalf("cycle error = %v", err)
	}
}

func TestFig2ReactionCombination(t *testing.T) {
	n := NewNode()
	n.Conf[2] = true
	n.Weight[1] = 2
	n.Reaction["L"] = []float64{0, 0, 10, 0, 0, 0}
	n.Reaction["X"] = []float64{0, 0, 3, 0, 0, 0}
	spec, err := fig2NodalSpec([]string{"Z", "L", "W", "-X", "0.005"}, false)
	if err != nil {
		t.Fatal(err)
	}
	value, ok := fig2NodeValue(n, spec)
	if !ok || value != 9 || spec.scale != 0.005 {
		t.Fatalf("value=%v drawable=%v spec=%#v", value, ok, spec)
	}
}

func TestFig2PDFWritesPagesWithoutMutatingFrame(t *testing.T) {
	if _, err := os.Stat(defaultfont); err != nil {
		t.Skip("Japanese PDF font unavailable")
	}
	dir := t.TempDir()
	f := NewFrame()
	f.Path = filepath.Join(dir, "model.inp")
	for i, coord := range [][]float64{{0, 0, 0}, {2, 0, 3}, {4, 0, 6}} {
		n := NewNode()
		n.Num = i + 1
		n.Coord = coord
		n.Frame = f
		f.Nodes[n.Num] = n
	}
	f.Ai.Nfloor = 2
	f.Ai.Boundary = []float64{-1, 4, 7}
	f.View.Angle[1] = 166
	f.Nodes[3].Hide()
	f.Measures = []*Measure{NewMeasure(f.Nodes[1].Coord, f.Nodes[2].Coord, XAXIS)}
	fig2 := filepath.Join(dir, "drawing.fig2")
	if err := os.WriteFile(fig2, []byte("PAGE A4TATE\nHEIGHT 0 1\nTEXTBOX NOTE 10 30\nNOTE 寸法\nXMEASURE 1 2 Z 0.2 _mm 0.1\nPAGE A4YOKO\nTITLE 次の図\n"), 0600); err != nil {
		t.Fatal(err)
	}
	pdf := filepath.Join(dir, "drawing.pdf")
	if err := WriteFig2PDF(f, fig2, pdf); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(pdf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "%PDF-") || len(regexp.MustCompile(`/Type /Page\b`).FindAll(data, -1)) != 2 {
		t.Fatal("expected a readable two-page PDF")
	}
	if f.View.Angle[1] != 166 || !f.Nodes[3].hide || len(f.Measures) != 1 || f.Show.Fig2Document {
		t.Fatal("PDF generation changed the live frame")
	}
}
