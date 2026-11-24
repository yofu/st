package stgui

import (
	"slices"
	"strings"
	"testing"

	st "github.com/yofu/st/stlib"
)

func TestNotice1459Report(t *testing.T) {
	f := st.NewFrame()
	f.Show.Period = "L"
	s := f.AddSect(501)
	s.Name = "G1"
	prop := &st.Prop{Material: st.FC24, EFactor: 1}
	s.Figs = []*st.Fig{{Prop: prop}}
	nodes := []*st.Node{f.AddNode(0, 0, 0), f.AddNode(5, 0, 0), f.AddNode(10, 0, 0)}
	for i := 0; i < 2; i++ {
		f.AddLineElem(21+i, nodes[i:i+2], s, st.GIRDER)
	}
	nodes[1].Disp["L"] = []float64{0, 0, -.03, 0, 0, 0}
	check := func(n int, wantMembers, wantAlpha, wantRatio, wantSlope string) {
		t.Helper()
		var history []string
		text := notice1459Report(f, nodes[:n], func(s string) { history = append(history, s) })
		if !strings.Contains(text, "部材 "+wantMembers+" 、断面 501") || !strings.Contains(text, "変形増大係数α= "+wantAlpha+"\n") ||
			!strings.HasSuffix(text, " = "+wantRatio+"\n") || !slices.Contains(history, "Slope: "+wantSlope) {
			t.Errorf("%d nodes: report %q history %q", n, text, history)
		}
	}
	check(2, "21", "8", "1/20", "1/20.8")
	check(3, "21, 22", "8", "1/41", "1/41.7")
	prop.Material = st.SN400
	check(2, "21", "1", "1/166", "1/166.7")
	check(3, "21, 22", "1", "1/333", "1/333.3")
	nodes[1].Disp["L"][2] = .03
	upward := notice1459Report(f, nodes, func(string) {})
	if !strings.Contains(upward, "= -3.000 [cm]") || !strings.Contains(upward, "α×|δ|/L=1×3.000/ 1000.000 = 1/333") {
		t.Fatal(upward)
	}
	nodes[1].Disp["L"][2] = 0
	check(3, "21, 22", "1", "0", "0")
	nodes[1].Coord[0], nodes[1].Coord[2] = 0, 5
	check(2, "21", "1", "算定不可", "算定不可")
}
