package matrix

import (
	"container/heap"
	"fmt"
	"math"
	"sort"
	"time"
)

// SymLDLTPlan owns an immutable copy of the constrained upper triangles and a
// minimum-degree ordering of their structural union. A plan can be shared by
// shifts and goroutines: every Factor owns its numerical workspace and result.
// Rebuild the plan when input values or constraints change.
type SymLDLTPlan struct {
	order           []int
	rows            [][]symmetricInput
	orderingSeconds float64
}

type symmetricInput struct {
	column int
	a, b   float64
}
type symmetricRow struct {
	columns []int
	values  []float64
}

// NewSymLDLTPlan analyzes co + shift*other. Structural entries in either matrix
// participate in the ordering even if they cancel at a particular shift.
func NewSymLDLTPlan(co, other *COOMatrix, csize int, conf []bool) (*SymLDLTPlan, error) {
	if co == nil || csize < 0 || csize > co.Size || len(conf) != co.Size || other != nil && other.Size != co.Size {
		return nil, fmt.Errorf("invalid symmetric LDLT dimensions")
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
	rows := make([][]symmetricInput, n)
	add := func(input *COOMatrix, second bool) error {
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
				if !finiteSym(v) {
					return fmt.Errorf("non-finite symmetric LDLT input at %d,%d", i, j)
				}
				if v == 0 {
					continue
				}
				e := symmetricInput{column: index[j], a: v}
				if second {
					e.a, e.b = 0, v
				}
				rows[index[i]] = append(rows[index[i]], e)
				if i != j {
					e.column = index[i]
					rows[index[j]] = append(rows[index[j]], e)
				}
			}
		}
		return nil
	}
	if err := add(co, false); err != nil {
		return nil, err
	}
	if other != nil {
		if err := add(other, true); err != nil {
			return nil, err
		}
	}
	for i, row := range rows {
		sort.Slice(row, func(a, b int) bool { return row[a].column < row[b].column })
		out := row[:0]
		for _, e := range row {
			if len(out) > 0 && out[len(out)-1].column == e.column {
				out[len(out)-1].a += e.a
				out[len(out)-1].b += e.b
			} else {
				out = append(out, e)
			}
		}
		rows[i] = out
	}
	start := time.Now()
	order := symmetricMinimumDegree(rows)
	p := &SymLDLTPlan{order: order, rows: make([][]symmetricInput, n), orderingSeconds: time.Since(start).Seconds()}
	inverse := make([]int, n)
	for i, v := range order {
		inverse[v] = i
	}
	for i, v := range order {
		p.rows[i] = make([]symmetricInput, len(rows[v]))
		for j, e := range rows[v] {
			e.column = inverse[e.column]
			p.rows[i][j] = e
		}
		sort.Slice(p.rows[i], func(a, b int) bool { return p.rows[i][a].column < p.rows[i][b].column })
	}
	return p, nil
}

// degreeQueue contains one live heap entry per vertex. Deterministic tie breaks
// and heap.Fix avoid stale-entry growth on large sparse graphs.
type degreeQueue struct{ vertices, positions, degrees []int }

func (q degreeQueue) Len() int { return len(q.vertices) }
func (q degreeQueue) Less(i, j int) bool {
	a, b := q.vertices[i], q.vertices[j]
	return q.degrees[a] < q.degrees[b] || q.degrees[a] == q.degrees[b] && a < b
}
func (q degreeQueue) Swap(i, j int) {
	q.vertices[i], q.vertices[j] = q.vertices[j], q.vertices[i]
	q.positions[q.vertices[i]] = i
	q.positions[q.vertices[j]] = j
}
func (q *degreeQueue) Push(x interface{}) {
	v := x.(int)
	q.positions[v] = len(q.vertices)
	q.vertices = append(q.vertices, v)
}
func (q *degreeQueue) Pop() interface{} {
	last := len(q.vertices) - 1
	v := q.vertices[last]
	q.vertices = q.vertices[:last]
	q.positions[v] = -1
	return v
}

// Exact minimum degree using sorted adjacency arrays and clique merges. Storage
// follows symbolic fill, not n*n. Very small systems retain natural ordering to
// avoid analysis overhead. Numeric pivoting remains independent of this order.
func symmetricMinimumDegree(input [][]symmetricInput) []int {
	n := len(input)
	order := make([]int, 0, n)
	if n <= 16 {
		for i := 0; i < n; i++ {
			order = append(order, i)
		}
		return order
	}
	graph := make([][]int, n)
	q := &degreeQueue{vertices: make([]int, n), positions: make([]int, n), degrees: make([]int, n)}
	for i, row := range input {
		for _, e := range row {
			if e.column != i {
				graph[i] = append(graph[i], e.column)
			}
		}
		q.vertices[i] = i
		q.positions[i] = i
		q.degrees[i] = len(graph[i])
	}
	heap.Init(q)
	var scratch []int
	for q.Len() > 0 {
		v := heap.Pop(q).(int)
		order = append(order, v)
		neighbors := graph[v]
		for _, i := range neighbors {
			old := graph[i]
			scratch = scratch[:0]
			a, b := 0, 0
			for a < len(old) || b < len(neighbors) {
				var j int
				if b == len(neighbors) || a < len(old) && old[a] < neighbors[b] {
					j = old[a]
					a++
				} else if a == len(old) || neighbors[b] < old[a] {
					j = neighbors[b]
					b++
				} else {
					j = old[a]
					a++
					b++
				}
				if j != i && j != v {
					scratch = append(scratch, j)
				}
			}
			graph[i], scratch = scratch, old[:0]
			q.degrees[i] = len(graph[i])
			heap.Fix(q, q.positions[i])
		}
		graph[v] = nil
	}
	return order
}

func (p *SymLDLTPlan) Factor(shift float64, progress chan int, inertiaOnly bool) (*SymLDLTFactor, error) {
	return p.factor(shift, progress, inertiaOnly, nil)
}
func (p *SymLDLTPlan) FactorWithStats(shift float64, progress chan int, inertiaOnly bool) (*SymLDLTFactor, *SymLDLTStatistics, error) {
	s := new(SymLDLTStatistics)
	f, e := p.factor(shift, progress, inertiaOnly, s)
	return f, s, e
}

func (p *SymLDLTPlan) factor(shift float64, progress chan int, inertiaOnly bool, stats *SymLDLTStatistics) (*SymLDLTFactor, error) {
	if p == nil || !finiteSym(shift) {
		return nil, fmt.Errorf("invalid symmetric LDLT shift or plan")
	}
	start := time.Now()
	n := len(p.rows)
	a := make([]symmetricRow, n)
	live := 0
	for i, row := range p.rows {
		a[i].columns = make([]int, 0, len(row))
		a[i].values = make([]float64, 0, len(row))
		for _, e := range row {
			v := e.a + shift*e.b
			if !finiteSym(v) {
				return nil, fmt.Errorf("non-finite symmetric LDLT input at %d,%d", i, e.column)
			}
			if v != 0 {
				a[i].columns = append(a[i].columns, e.column)
				a[i].values = append(a[i].values, v)
				live++
				if stats != nil && e.column >= i {
					stats.InputUpperNNZ++
				}
			}
		}
	}
	if stats != nil {
		stats.Size = n
		stats.PeakSchurEntries = live
		stats.OrderingSeconds = p.orderingSeconds
		stats.Ordering = "minimum_degree"
		if n <= 16 {
			stats.Ordering = "natural_small"
		}
		stats.BuildSeconds = time.Since(start).Seconds()
	}
	numericStart := time.Now()
	f := &SymLDLTFactor{Size: n, permutation: append([]int(nil), p.order...), inertiaOnly: inertiaOnly}
	if !inertiaOnly {
		f.lower = make([][]symmetricEntry, n)
	}
	// Rows keep stable vertex IDs. Only these two O(n) vectors move on a
	// symmetric pivot; no sparse columns have to be searched or relabelled.
	vertex, position := make([]int, n), make([]int, n)
	for i := range vertex {
		vertex[i] = i
		position[i] = i
	}
	swap := func(i, j int) {
		if i == j {
			return
		}
		vertex[i], vertex[j] = vertex[j], vertex[i]
		position[vertex[i]], position[vertex[j]] = i, j
		f.permutation[i], f.permutation[j] = f.permutation[j], f.permutation[i]
		if !inertiaOnly {
			f.lower[i], f.lower[j] = f.lower[j], f.lower[i]
		}
		if stats != nil {
			stats.Swaps++
		}
	}
	get := func(i, j int) float64 {
		r := a[i]
		s := sort.SearchInts(r.columns, j)
		if s < len(r.columns) && r.columns[s] == j {
			return r.values[s]
		}
		return 0
	}
	alpha := (1 + math.Sqrt(17)) / 8
	rows := make([]int, 0)
	marks := make([]int, n)
	w0, w1, col0, col1 := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	var scratch symmetricRow
	for k := 0; k < n; {
		step, kp := 1, k
		v := vertex[k]
		absakk, colmax, imax := math.Abs(get(v, v)), 0.0, k
		for j, i := range a[v].columns {
			value := math.Abs(a[v].values[j])
			if position[i] > k && (value > colmax || value == colmax && position[i] < imax) {
				colmax, imax = value, position[i]
			}
		}
		if colmax != 0 && absakk < alpha*colmax {
			u := vertex[imax]
			rowmax := 0.0
			for j, i := range a[u].columns {
				if position[i] >= k && i != u {
					rowmax = math.Max(rowmax, math.Abs(a[u].values[j]))
				}
			}
			if absakk >= alpha*colmax*(colmax/rowmax) {
				kp = k
			} else if math.Abs(get(u, u)) >= alpha*rowmax {
				kp = imax
			} else {
				step, kp = 2, imax
			}
		}
		swap(k+step-1, kp)
		v = vertex[k]
		u := v
		if step == 2 {
			u = vertex[k+1]
		}
		block := symmetricBlock{start: k, size: step, a: get(v, v)}
		if step == 2 {
			block.b, block.c = get(v, u), get(u, u)
		}
		rows = rows[:0]
		for j := 0; j < step; j++ {
			for _, i := range a[vertex[k+j]].columns {
				if position[i] >= k+step && marks[i] != k+1 {
					marks[i] = k + 1
					rows = append(rows, i)
				}
			}
		}
		sort.Ints(rows)
		if stats != nil && len(rows) > stats.MaxNeighbors {
			stats.MaxNeighbors = len(rows)
		}
		for _, i := range rows {
			col0[i] = get(i, v)
			col1[i] = 0
			if step == 2 {
				col1[i] = get(i, u)
			}
			var x, y float64
			if step == 1 {
				x = col0[i] / block.a
			} else {
				var err error
				x, y, err = solveSymmetricBlock(block, col0[i], col1[i])
				if err != nil {
					return nil, fmt.Errorf("symmetric LDLT block %d: %w", k, err)
				}
			}
			if !finiteSym(x) || !finiteSym(y) {
				return nil, fmt.Errorf("non-finite symmetric LDLT multiplier at %d", i)
			}
			w0[i], w1[i] = x, y
			if !inertiaOnly {
				r := position[i]
				if x != 0 {
					f.lower[r] = append(f.lower[r], symmetricEntry{k, x})
					if stats != nil {
						stats.StoredLNNZ++
					}
				}
				if step == 2 && y != 0 {
					f.lower[r] = append(f.lower[r], symmetricEntry{k + 1, y})
					if stats != nil {
						stats.StoredLNNZ++
					}
				}
			}
		}
		// Merge a whole sparse row with the sorted pivot neighborhood. No
		// hashing or per-element binary search occurs in this hot loop.
		for _, i := range rows {
			old := a[i]
			scratch.columns = scratch.columns[:0]
			scratch.values = scratch.values[:0]
			ii, jj := 0, 0
			for ii < len(old.columns) || jj < len(rows) {
				var j int
				value := 0.0
				update, existed := false, false
				if jj == len(rows) || ii < len(old.columns) && old.columns[ii] < rows[jj] {
					j = old.columns[ii]
					value = old.values[ii]
					ii++
					existed = true
				} else if ii == len(old.columns) || rows[jj] < old.columns[ii] {
					j = rows[jj]
					jj++
					update = true
				} else {
					j = old.columns[ii]
					value = old.values[ii]
					ii++
					jj++
					update, existed = true, true
				}
				if position[j] < k+step {
					continue
				}
				if update {
					// Use the identical expression on both triangles, including
					// after dynamic pivot swaps, to preserve exact symmetry.
					lo, hi := i, j
					if lo > hi {
						lo, hi = hi, lo
					}
					value -= col0[lo] * w0[hi]
					if step == 2 {
						value -= col1[lo] * w1[hi]
					}
					if !finiteSym(value) {
						return nil, fmt.Errorf("non-finite symmetric LDLT update at %d,%d", i, j)
					}
				}
				if value != 0 {
					scratch.columns = append(scratch.columns, j)
					scratch.values = append(scratch.values, value)
					if stats != nil && !existed && j >= i {
						stats.FillCreated++
					}
				}
			}
			live += len(scratch.columns) - len(old.columns)
			a[i], scratch = scratch, old
			if stats != nil && live > stats.PeakSchurEntries {
				stats.PeakSchurEntries = live
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
			id := vertex[k+j]
			live -= len(a[id].columns)
			a[id] = symmetricRow{}
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
