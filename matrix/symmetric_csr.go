package matrix

import (
	"fmt"
	"sort"
)

// SymmetricCSR is an immutable compressed sparse matrix for repeated products.
// It uses the same upper-triangle/constraint convention as SymmetricLDLT.
type SymmetricCSR struct {
	Size             int
	offsets, columns []int
	values           []float64
}

func (co *COOMatrix) ToSymmetricCSR(csize int, conf []bool) (*SymmetricCSR, error) {
	if co == nil || csize < 0 || csize > co.Size || len(conf) != co.Size {
		return nil, fmt.Errorf("invalid symmetric CSR dimensions")
	}
	index := make([]int, co.Size)
	n := 0
	for i, fixed := range conf {
		index[i] = -1
		if !fixed {
			index[i] = n
			n++
		}
	}
	if n != co.Size-csize {
		return nil, fmt.Errorf("symmetric CSR constraint count mismatch")
	}
	rows := make([][]symmetricEntry, n)
	for i, row := range co.data {
		if i < 0 || i >= co.Size {
			return nil, fmt.Errorf("symmetric CSR row out of range")
		}
		for j, v := range row {
			if j < 0 || j >= co.Size {
				return nil, fmt.Errorf("symmetric CSR column out of range")
			}
			if j < i || index[i] < 0 || index[j] < 0 {
				continue
			}
			if !finiteSym(v) {
				return nil, fmt.Errorf("non-finite symmetric CSR input")
			}
			if v == 0 {
				continue
			}
			r, c := index[i], index[j]
			rows[r] = append(rows[r], symmetricEntry{c, v})
			if r != c {
				rows[c] = append(rows[c], symmetricEntry{r, v})
			}
		}
	}
	f := &SymmetricCSR{Size: n, offsets: make([]int, n+1)}
	for i, row := range rows {
		sort.Slice(row, func(a, b int) bool { return row[a].column < row[b].column })
		for _, e := range row {
			f.columns = append(f.columns, e.column)
			f.values = append(f.values, e.value)
		}
		f.offsets[i+1] = len(f.values)
	}
	return f, nil
}

// MulV visits only stored nonzeros, O(nnz), instead of scanning all columns.
func (f *SymmetricCSR) MulV(x []float64) []float64 {
	if len(x) != f.Size {
		panic("symmetric CSR vector size mismatch")
	}
	y := make([]float64, f.Size)
	for i := range y {
		for k := f.offsets[i]; k < f.offsets[i+1]; k++ {
			y[i] += f.values[k] * x[f.columns[k]]
		}
	}
	return y
}
