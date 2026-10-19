package matrix

import "testing"

func TestSymmetricLDLTDiagnosticStatistics(t *testing.T) {
	a := symmetricTestCOO([][]float64{{0, 1, 0}, {1, 0, .5}, {0, .5, 2}})
	f, stats, err := a.SymmetricLDLTWithStats(nil, 0, 0, make([]bool, 3), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Size != 3 || stats.InputUpperNNZ != 3 || stats.CompletedPivots != 3 || stats.TwoByTwo != 1 || stats.OneByOne != 1 || stats.StoredLNNZ == 0 || stats.MaxNeighbors < 1 || stats.PeakSchurEntries < 5 {
		t.Fatalf("stats=%+v", stats)
	}
	if stats.BuildSeconds < 0 || stats.NumericSeconds < 0 {
		t.Fatal("invalid timing")
	}
	checkSymmetricFactor(t, [][]float64{{0, 1, 0}, {1, 0, .5}, {0, .5, 2}}, f, false)
	i, s, err := a.SymmetricLDLTWithStats(nil, 0, 0, make([]bool, 3), nil, true)
	if err != nil || s.StoredLNNZ != 0 {
		t.Fatal(s, err)
	}
	p, z, n := f.Inertia()
	ip, iz, in := i.Inertia()
	if p != ip || z != iz || n != in {
		t.Fatal("instrumentation changed inertia")
	}
	ll := a.ToLLS(0, make([]bool, 3))
	before := ll.DiagnosticStatistics()
	// ToLLS retains the explicit zero upper entry (0,2) as a stored node.
	if before.StoredEntries != 6 || !before.Finite {
		t.Fatal(before)
	}
}
