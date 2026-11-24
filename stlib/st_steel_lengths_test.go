package st

import (
	"math"
	"strings"
	"testing"
)

func steelLengthFixture() (*Frame, *Elem, *SGirder) {
	f := NewFrame()
	s := f.AddSect(511)
	sg := NewSGirder(511, nakanoHBeams[4].shape, SN400)
	s.Allow = sg
	a, b := f.AddNode(0, 0, 0), f.AddNode(4, 0, 0)
	el := f.AddLineElem(-1, []*Node{a, b}, s, GIRDER)
	return f, el, sg
}

func resolvedSteelLengths(t *testing.T, f *Frame, el *Elem, opt SteelLengthOptions) SteelMemberLengths {
	t.Helper()
	l, err := NewSteelLengthResolver(f).Resolve(el, opt)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func checkSteelLength(t *testing.T, v SteelAdoptedLength, cm float64, source string, factor float64) {
	t.Helper()
	if math.Abs(v.CM-cm) > 1e-8 || v.Source != source || v.Factor != factor {
		t.Fatalf("got %+v; want %g cm from %s (factor %g)", v, cm, source, factor)
	}
}

func TestSteelLengthPriority(t *testing.T) {
	f, el, sg := steelLengthFixture()
	sg.BBLength, sg.BBFactor = []float64{200, 0}, []float64{3, 1.5}
	sg.BTLength, sg.BTFactor = []float64{0, 225}, []float64{0.75, 2}
	// Mixed absolute and factored axes must survive saving the LST.
	if err := f.ReadLstString(sg.String(), "lengths.lst"); err != nil {
		t.Fatal(err)
	}
	opt := SteelLengthOptions{DesignCM: 350}
	l := resolvedSteelLengths(t, f, el, opt)
	checkSteelLength(t, l.Buckling[0], 200, "BBLEN", 0)
	checkSteelLength(t, l.Buckling[1], 525, "BBFAC", 1.5)
	checkSteelLength(t, l.Lateral[0], 262.5, "BTFAC", 0.75)
	checkSteelLength(t, l.Lateral[1], 225, "BTLEN", 0)
	opt.BucklingCM[0], opt.LateralCM[1] = 285, 400
	l = resolvedSteelLengths(t, f, el, opt)
	checkSteelLength(t, l.Buckling[0], 285, "explicit", 0)
	checkSteelLength(t, l.Lateral[1], 400, "explicit", 0)
	checkSteelLength(t, l.Buckling[1], 525, "BBFAC", 1.5)
	checkSteelLength(t, l.Lateral[0], 262.5, "BTFAC", 0.75)
	if l.DesignCM != 350 || l.DesignSource != "explicit" {
		t.Fatalf("incorrect design provenance: %+v", l)
	}

	// Read-only query for one member, with command-line metres and axis overrides.
	w := &dxfDimensionCommandWindow{frame: f}
	msg := exCommand(w, ":yieldlength -elem=1001 -design=3.5 -lk=2.85 -lky=3 -lby=4", nil, nil)
	m, ok := msg.(Message)
	if !ok {
		t.Fatalf("expected length report, got %v", msg)
	}
	for _, want := range []string{"ELEM 1001", "設計長 3.5000 m", "Lkx 2.8500 m (明示)", "Lky 3.0000 m (明示)", "Lbx 2.6250 m (BTFAC 0.75", "Lby 4.0000 m (明示)"} {
		if !strings.Contains(string(m), want) {
			t.Errorf("missing %q in %s", want, m)
		}
	}
}

func TestSteelLengthMeshAndBracing(t *testing.T) {
	f, el, _ := steelLengthFixture()
	ns, parts, err := el.DivideAtRate(0.3, 1e-8)
	if err != nil {
		t.Fatal(err)
	}
	parts[0].Chain.Break() // Older INPs can lack CHAIN tags.
	mid := ns[0]
	mid.Load[2] = 3
	for _, part := range parts {
		l := resolvedSteelLengths(t, f, part, SteelLengthOptions{})
		checkSteelLength(t, l.Buckling[0], 400, "design", 1)
		if l.EndNodes != [2]int{parts[0].Enod[0].Num, parts[1].Enod[1].Num} {
			t.Fatalf("wrong span %v", l.EndNodes)
		}
	}

	// An intersecting member or a partial restraint need not brace both axes.
	mid.Conf[2] = true
	l := resolvedSteelLengths(t, f, parts[0], SteelLengthOptions{BucklingCM: [2]float64{300, 0}})
	if l.Review == "" || l.Buckling[1].CM != 0 {
		t.Fatalf("expected unresolved span: %+v", l)
	}
	checkSteelLength(t, l.Buckling[0], 300, "explicit", 0)
	if _, err := l.Yield(nil, -1, 5.4, 1); err == nil {
		t.Fatal("yield used ambiguous span")
	}
	mid.Conf[2] = false
	f.AddLineElem(-1, []*Node{mid, f.AddNode(1.2, 1, 0)}, el.Sect, GIRDER)
	l = resolvedSteelLengths(t, f, parts[0], SteelLengthOptions{})
	if l.Review == "" {
		t.Fatal("multi-member joint silently used as a brace")
	}
	// Only the major-axis lateral length is needed by an HKYOU girder.
	l = resolvedSteelLengths(t, f, parts[0], SteelLengthOptions{BucklingCM: [2]float64{400, 400}, LateralCM: [2]float64{400, 0}})
	if _, err := l.Yield(nil, -1, 5.4, 1); err != nil {
		t.Fatal(err)
	}
	mid.Conf[0], mid.Conf[1], mid.Conf[2] = true, true, true
	l = resolvedSteelLengths(t, f, parts[0], SteelLengthOptions{})
	checkSteelLength(t, l.Buckling[0], 120, "design", 1)
}

func TestSteelLengthAxisMapping(t *testing.T) {
	f, el, beam := steelLengthFixture()
	h := nakanoHBeams[0].shape
	beam.Shape = HWEAK(h)
	opt := SteelLengthOptions{BucklingCM: [2]float64{10, 2000}, LateralCM: [2]float64{400, 1000}}
	l := resolvedSteelLengths(t, f, el, opt)
	got, err := l.Yield(nil, 1, 5.4, 1)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got[0]-160.363526927) > 1e-8 || math.Abs(got[10]-41.880830639) > 1e-8 {
		t.Fatalf("HWEAK mapping %v", got)
	}

	p := nakanoTubeColumns[0].product
	el.Etype = COLUMN
	el.Sect.Allow = NewSColumn(el.Sect.Num, p.Shape, SN400)
	opt = SteelLengthOptions{BucklingCM: [2]float64{285, 285}, LateralCM: [2]float64{285, 100000}}
	l = resolvedSteelLengths(t, f, el, opt)
	got, err = l.Yield(&p, -1, 5.4, 1)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got[8]-47.76) > 1e-8 || math.Abs(got[10]-15.20394531646539) > 1e-8 {
		t.Fatalf("tube equal axes with separate restraints: %v", got)
	}
}

func TestSteelLengthObliqueMesh(t *testing.T) {
	f, el, _ := steelLengthFixture()
	el.Enod[1].Coord = []float64{2.123, 1.456, 0.789}
	if err := el.SetPrincipalAxis(); err != nil {
		t.Fatal(err)
	}
	before := resolvedSteelLengths(t, f, el, SteelLengthOptions{})
	ns, parts, err := el.DivideAtRate(0.37, 1e-8)
	if err != nil {
		t.Fatal(err)
	}
	for axis, x := range ns[0].Coord {
		ns[0].Coord[axis] = math.Round(x*1000) / 1000
	}
	for _, part := range parts {
		if err := part.SetPrincipalAxis(); err != nil {
			t.Fatal(err)
		}
	}
	for _, part := range parts {
		l := resolvedSteelLengths(t, f, part, SteelLengthOptions{})
		if l.Buckling != before.Buckling || l.Lateral != before.Lateral {
			t.Fatalf("rounded oblique split: %s", l)
		}
	}
	ns[0].Coord[2] += 0.02
	if l := resolvedSteelLengths(t, f, parts[0], SteelLengthOptions{}); l.Review == "" {
		t.Fatal("bent chain adopted as a straight member")
	}
}

func TestSteelLengthNakanoSubdivision(t *testing.T) {
	f := readNakanoSteelYieldModel(t)
	// Column 1002: an absolute input bypasses BBFAC=1.2 on both local axes.
	p := nakanoTubeColumns[0].product
	col := f.Elems[1002]
	l := resolvedSteelLengths(t, f, col, SteelLengthOptions{BucklingCM: [2]float64{285, 285}, LateralCM: [2]float64{285, 285}})
	y, err := l.Yield(&p, -1, 5.4, 1)
	if err != nil {
		t.Fatal(err)
	}
	checkCold01Limits(t, 201, y, nakanoTubeColumns[0].want)
	l = resolvedSteelLengths(t, f, col, SteelLengthOptions{DesignCM: 381.5})
	checkSteelLength(t, l.Buckling[0], 457.8, "BBFAC", 1.2)
	checkSteelLength(t, l.Lateral[0], 381.5, "design", 1)
	columnLimits, err := l.Yield(&p, -1, 5.4, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, columnParts, err := col.DivideAtMid(1e-8)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range columnParts {
		lengths := resolvedSteelLengths(t, f, part, SteelLengthOptions{DesignCM: 381.5})
		got, err := lengths.Yield(&p, -1, 5.4, 1)
		if err != nil || got != columnLimits {
			t.Fatalf("column %d after split: %v, %v; want %v", part.Num, got, err, columnLimits)
		}
	}

	// Beam 2399 runs from NODE 380 to 385, 0.935 m. Both LST factors are 2.
	beam := f.Elems[2399]
	before := resolvedSteelLengths(t, f, beam, SteelLengthOptions{})
	checkSteelLength(t, before.Buckling[1], 187, "BBFAC", 2)
	checkSteelLength(t, before.Lateral[0], 187, "BTFAC", 2)
	want, err := before.Yield(nil, -1, 5.4, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Independent direct call with the final 187 cm catches any reapplication of factors.
	c := cold01HBeamCondition()
	c.StrongBucklingLength, c.WeakBucklingLength, c.LateralLength = 187, 187, 187
	direct, err := beam.Sect.Allow.(*SGirder).HBeamYield(c)
	if err != nil {
		t.Fatal(err)
	}
	_, parts, err := beam.DivideInN(3, 1e-8)
	if err != nil || len(parts) != 3 {
		t.Fatalf("three-way split: %d elements, %v", len(parts), err)
	}
	for _, part := range parts {
		l := resolvedSteelLengths(t, f, part, SteelLengthOptions{})
		if l.Buckling != before.Buckling || l.Lateral != before.Lateral || l.EndNodes != before.EndNodes {
			t.Fatalf("ELEM %d: %s", part.Num, l)
		}
		got, err := l.Yield(nil, -1, 5.4, 1)
		if err != nil {
			t.Fatal(err)
		}
		for i := range got {
			if math.Abs(want[i]-direct[i]) > 1e-9 || got[i] != want[i] {
				t.Errorf("ELEM %d limit %d: unsplit %g split %g direct %g", part.Num, i, want[i], got[i], direct[i])
			}
		}
	}
	t.Logf("ELEM 2399: design %.3f m, effective %.3f m, NZMAX %.9f tf, MXMAX %.9f tf; %d parts", before.DesignCM/100, before.Lateral[0].CM/100, want[0], want[8], len(parts))
	// Reload the split geometry and the original section list. The endpoints
	// survive INP coordinate rounding even when the two new interior nodes move.
	reloaded := NewFrame()
	if err := reloaded.ReadInpSnapshot(f.InpBytes(), f.Path); err != nil {
		t.Fatal(err)
	}
	if err := reloaded.ReadLst(f.LstFileName); err != nil {
		t.Fatal(err)
	}
	for _, part := range parts {
		l := resolvedSteelLengths(t, reloaded, reloaded.Elems[part.Num], SteelLengthOptions{})
		got, err := l.Yield(nil, -1, 5.4, 1)
		if err != nil || got != want {
			t.Fatalf("reloaded ELEM %d: %s, %v; got %v, want %v", part.Num, l, err, got, want)
		}
	}
}
