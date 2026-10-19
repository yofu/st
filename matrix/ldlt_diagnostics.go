package matrix

// SymLDLTStatistics is collected only when explicitly requested. Schur entry
// counts include both triangles and nonzero diagonals; InputUpperNNZ counts
// only the upper triangle. FillCreated counts new upper-triangle positions
// introduced by numerical updates, excluding input assembly. It can count a
// position again if exact cancellation removed it before it was recreated.
type SymLDLTStatistics struct {
	Ordering         string  `json:"ordering,omitempty"`
	OrderingSeconds  float64 `json:"ordering_seconds"`
	BuildSeconds     float64 `json:"build_seconds"`
	NumericSeconds   float64 `json:"numeric_seconds"`
	Size             int     `json:"size"`
	InputUpperNNZ    int     `json:"input_upper_nnz"`
	PeakSchurEntries int     `json:"peak_schur_entries"`
	FillCreated      int     `json:"fill_created"`
	MaxNeighbors     int     `json:"max_neighbors"`
	StoredLNNZ       int     `json:"stored_l_nnz"`
	OneByOne         int     `json:"one_by_one_blocks"`
	TwoByTwo         int     `json:"two_by_two_blocks"`
	Swaps            int     `json:"symmetric_swaps"`
	CompletedPivots  int     `json:"completed_pivots"`
}

// SymmetricLDLTWithStats factors co + shift*other. Unlike the normal APIs this
// explicitly enables additional fill-in counters. A nil other factors co alone.
func (co *COOMatrix) SymmetricLDLTWithStats(other *COOMatrix, shift float64, csize int, conf []bool, progress chan int, inertiaOnly bool) (*SymLDLTFactor, *SymLDLTStatistics, error) {
	stats := new(SymLDLTStatistics)
	f, err := co.symmetricLDLT(other, shift, csize, conf, progress, inertiaOnly, stats)
	return f, stats, err
}

// LLSDiagnosticStatistics describes stored nodes, including numerical zeros.
// It is a post-factorization scan, not a measurement of peak storage or of
// the legacy solver's maximum active neighborhood.
type LLSDiagnosticStatistics struct {
	StoredEntries    int  `json:"stored_entries"`
	MaxColumnEntries int  `json:"max_column_entries"`
	Finite           bool `json:"finite"`
	Positive         int  `json:"positive"`
	Zero             int  `json:"zero"`
	Negative         int  `json:"negative"`
}

func (ll *LLSMatrix) DiagnosticStatistics() LLSDiagnosticStatistics {
	s := LLSDiagnosticStatistics{Finite: true}
	for _, diag := range ll.diag {
		length := 0
		for node := diag; node != nil; node = node.down {
			length++
			s.StoredEntries++
			if !finiteSym(node.value) {
				s.Finite = false
			}
		}
		if length > s.MaxColumnEntries {
			s.MaxColumnEntries = length
		}
		if diag.value > 0 {
			s.Positive++
		} else if diag.value < 0 {
			s.Negative++
		} else {
			s.Zero++
		}
	}
	return s
}
