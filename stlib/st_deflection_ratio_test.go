package st

import (
	"math"
	"path/filepath"
	"testing"
)

func TestNotice1459DeflectionRatio(t *testing.T) {
	for _, tc := range []struct {
		span, delta, alpha float64
		report, history    string
	}{
		{1000, 3, 1, "1/333", "1/333.3"},
		{1000, -3, 8, "1/41", "1/41.7"},
		{1000, 0, 8, "0", "0"},
		{0, 3, 1, "算定不可", "算定不可"},
		{0, 0, 1, "算定不可", "算定不可"},
		{1, 2, 1, "2", "2"},
		{100, math.NaN(), 8, "算定不可", "算定不可"},
	} {
		report, history := DeflectionRatio(tc.span, tc.alpha*tc.delta)
		if report != tc.report || history != tc.history {
			t.Errorf("span=%g delta=%g alpha=%g: report=%q history=%q", tc.span, tc.delta, tc.alpha, report, history)
		}
	}
}

func TestWriteReactionUnknownNodeSet(t *testing.T) {
	dir := t.TempDir()
	f := NewFrame()
	f.Path = filepath.Join(dir, "model.inp")
	w := &dxfDimensionCommandWindow{frame: f, dir: dir}
	err := exCommand(w, ":writereaction -nodeset=ABC reaction.txt 2", nil, nil)
	if err == nil || err.Error() != "nodeset ABC is not defined" {
		t.Fatalf("error = %v", err)
	}
}
