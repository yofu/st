package matrix

import (
	"fmt"
	"math"
	"sort"
	"time"
)

func (co *COOMatrix) symmetricLDLTMapReference(other *COOMatrix, shift float64, csize int, conf []bool, progress chan int, inertiaOnly bool, stats *SymLDLTStatistics) (*SymLDLTFactor, error) {
	var buildStart, numericStart time.Time
	if stats != nil {
		buildStart = time.Now()
	}
	if co == nil || csize < 0 || csize > co.Size || len(conf) != co.Size {
		return nil, fmt.Errorf("invalid symmetric LDLT dimensions")
	}
	if !finiteSym(shift) || other != nil && other.Size != co.Size {
		return nil, fmt.Errorf("invalid symmetric LDLT shift")
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
		return nil, fmt.Errorf("symmetric LDLT constraint count mismatch")
	}
	a := make([]map[int]float64, n)
	for i := range a {
		a[i] = make(map[int]float64)
	}
	liveEntries := 0
	building := true
	if stats != nil {
		stats.Size = n
	}
	set := func(i, j int, v float64) {
		if stats != nil {
			was, now := a[i][j] != 0, v != 0
			if was != now {
				change := 2
				if i == j {
					change = 1
				}
				if !now {
					change = -change
				}
				liveEntries += change
				if liveEntries > stats.PeakSchurEntries {
					stats.PeakSchurEntries = liveEntries
				}
				if !building && now {
					stats.FillCreated++
				}
			}
		}
		if v == 0 {
			delete(a[i], j)
			delete(a[j], i)
		} else {
			a[i][j], a[j][i] = v, v
		}
	}
	addInput := func(input *COOMatrix, factor float64) error {
		for i, row := range input.data {
			if i < 0 || i >= co.Size {
				return fmt.Errorf("symmetric LDLT row out of range: %d", i)
			}
			for j, v := range row {
				if j < 0 || j >= co.Size {
					return fmt.Errorf("symmetric LDLT column out of range: %d", j)
				}
				if j < i || index[i] < 0 || index[j] < 0 {
					continue
				}
				value := a[index[i]][index[j]] + factor*v
				if !finiteSym(v) || !finiteSym(value) {
					return fmt.Errorf("non-finite symmetric LDLT input at %d,%d", i, j)
				}
				set(index[i], index[j], value)
			}
		}
		return nil
	}
	if err := addInput(co, 1); err != nil {
		return nil, err
	}
	if other != nil {
		if err := addInput(other, shift); err != nil {
			return nil, err
		}
	}
	if stats != nil {
		for i, row := range a {
			for j := range row {
				if j >= i {
					stats.InputUpperNNZ++
				}
			}
		}
	}
	building = false
	if stats != nil {
		stats.BuildSeconds = time.Since(buildStart).Seconds()
		numericStart = time.Now()
	}
	f := &SymLDLTFactor{Size: n, permutation: make([]int, n), inertiaOnly: inertiaOnly}
	if !inertiaOnly {
		f.lower = make([][]symmetricEntry, n)
	}
	for i := range f.permutation {
		f.permutation[i] = i
	}
	swapMarks, swapRows, swapStamp := make([]int, n), make([]int, 0), 0
	swap := func(i, j int) {
		if i == j {
			return
		}
		if stats != nil {
			stats.Swaps++
		}
		swapStamp++
		swapRows = swapRows[:0]
		add := func(r int) {
			if swapMarks[r] != swapStamp {
				swapMarks[r] = swapStamp
				swapRows = append(swapRows, r)
			}
		}
		add(i)
		add(j)
		for r := range a[i] {
			add(r)
		}
		for r := range a[j] {
			add(r)
		}
		a[i], a[j] = a[j], a[i]
		for _, r := range swapRows {
			row := a[r]
			x, y := row[i], row[j]
			delete(row, i)
			delete(row, j)
			if y != 0 {
				row[i] = y
			}
			if x != 0 {
				row[j] = x
			}
		}
		// Previously computed columns of L must follow the permutation too.
		if !inertiaOnly {
			f.lower[i], f.lower[j] = f.lower[j], f.lower[i]
		}
		f.permutation[i], f.permutation[j] = f.permutation[j], f.permutation[i]
	}
	alpha := (1 + math.Sqrt(17)) / 8
	marks := make([]int, n)
	rows := make([]int, 0)
	w0, w1, col0, col1 := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	for k := 0; k < n; {
		step, kp := 1, k
		absakk, colmax, imax := math.Abs(a[k][k]), 0.0, k
		for i, v := range a[k] {
			if i > k && (math.Abs(v) > colmax || math.Abs(v) == colmax && i < imax) {
				colmax, imax = math.Abs(v), i
			}
		}
		if colmax != 0 && absakk < alpha*colmax {
			rowmax := 0.0
			for j, v := range a[imax] {
				if j >= k && j != imax {
					rowmax = math.Max(rowmax, math.Abs(v))
				}
			}
			if absakk >= alpha*colmax*(colmax/rowmax) {
				kp = k
			} else if math.Abs(a[imax][imax]) >= alpha*rowmax {
				kp = imax
			} else {
				step, kp = 2, imax
			}
		}
		swap(k+step-1, kp)
		block := symmetricBlock{start: k, size: step, a: a[k][k]}
		if step == 2 {
			block.b, block.c = a[k][k+1], a[k+1][k+1]
		}
		rows = rows[:0]
		for j := 0; j < step; j++ {
			for i := range a[k+j] {
				if i >= k+step && marks[i] != k+1 {
					marks[i] = k + 1
					rows = append(rows, i)
				}
			}
		}
		sort.Ints(rows)
		if stats != nil && len(rows) > stats.MaxNeighbors {
			stats.MaxNeighbors = len(rows)
		}
		for ii, i := range rows {
			col0[ii] = a[i][k]
			if step == 2 {
				col1[ii] = a[i][k+1]
			}
			var x, y float64
			if step == 1 {
				// An isolated zero column is a valid zero block for inertia.
				x = col0[ii] / block.a
			} else {
				var err error
				x, y, err = solveSymmetricBlock(block, col0[ii], col1[ii])
				if err != nil {
					return nil, fmt.Errorf("symmetric LDLT block %d: %w", k, err)
				}
			}
			if !finiteSym(x) || !finiteSym(y) {
				return nil, fmt.Errorf("non-finite symmetric LDLT multiplier at %d", i)
			}
			w0[ii], w1[ii] = x, y
			if !inertiaOnly {
				if x != 0 {
					f.lower[i] = append(f.lower[i], symmetricEntry{k, x})
					if stats != nil {
						stats.StoredLNNZ++
					}
				}
				if step == 2 && y != 0 {
					f.lower[i] = append(f.lower[i], symmetricEntry{k + 1, y})
					if stats != nil {
						stats.StoredLNNZ++
					}
				}
			}
		}
		for ii, i := range rows {
			for jj := ii; jj < len(rows); jj++ {
				j := rows[jj]
				v := a[i][j] - col0[ii]*w0[jj]
				if step == 2 {
					v -= col1[ii] * w1[jj]
				}
				if !finiteSym(v) {
					return nil, fmt.Errorf("non-finite symmetric LDLT update at %d,%d", i, j)
				}
				set(i, j, v)
			}
		}
		f.blocks = append(f.blocks, block)
		if stats != nil {
			if step == 1 {
				stats.OneByOne++
			} else {
				stats.TwoByTwo++
			}
			stats.CompletedPivots = k + step
		}
		for j := 0; j < step; j++ {
			for _, i := range rows {
				if stats != nil && a[i][k+j] != 0 {
					liveEntries--
				}
				delete(a[i], k+j)
			}
			if stats != nil {
				liveEntries -= len(a[k+j])
			}
			a[k+j] = make(map[int]float64)
			if progress != nil {
				progress <- k + j
			}
		}
		k += step
	}
	if stats != nil {
		stats.NumericSeconds = time.Since(numericStart).Seconds()
	}
	return f, nil
}
