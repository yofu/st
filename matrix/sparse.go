package matrix

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// type Query struct {
//     row    int
//     column int
// }

// type Set struct {
//     row    int
//     column int
//     value  float64
// }

type COOMatrix struct {
	Size int
	nz   int
	data map[int]map[int]float64
}

func NewCOOMatrix(size int) *COOMatrix {
	rtn := new(COOMatrix)
	rtn.Size = size
	rtn.data = make(map[int]map[int]float64)
	return rtn
}

func (co *COOMatrix) String() string {
	var rtn bytes.Buffer
	for row := 0; row < co.Size; row++ {
		if rdata, rok := co.data[row]; rok {
			for col := 0; col < co.Size; col++ {
				if tmp, cok := rdata[col]; cok {
					rtn.WriteString(fmt.Sprintf("%d %d %25.18f\n", row+1, col+1, tmp))
				}
			}
		}
	}
	return rtn.String()
}

func (co *COOMatrix) Query(row, col int) float64 {
	if row > co.Size || col > co.Size {
		return 0.0
	}
	if rdata, rok := co.data[row]; rok {
		if tmp, cok := rdata[col]; cok {
			return tmp
		}
	}
	return 0.0
}

func (co *COOMatrix) Set(row, col int, val float64) {
	if row > co.Size || col > co.Size {
		return
	}
	if rdata, rok := co.data[row]; rok {
		if _, cok := rdata[col]; cok {
			co.data[row][col] = val
		} else {
			co.data[row][col] = val
			co.nz++
		}
	} else {
		co.data[row] = make(map[int]float64)
		co.data[row][col] = val
		co.nz++
	}
}

func (co *COOMatrix) Add(row, col int, val float64) {
	if row > co.Size || col > co.Size {
		return
	}
	if rdata, rok := co.data[row]; rok {
		if _, cok := rdata[col]; cok {
			co.data[row][col] += val
		} else {
			co.data[row][col] = val
			co.nz++
		}
	} else {
		co.data[row] = make(map[int]float64)
		co.data[row][col] = val
		co.nz++
	}
}

func (co *COOMatrix) MulV(csize int, conf []bool, vec []float64) []float64 {
	size := co.Size - csize
	rtn := make([]float64, size)
	rind := 0
	for row := 0; row < co.Size; row++ {
		if conf[row] {
			rind++
			continue
		}
		cind := 0
		if rdata, rok := co.data[row]; rok {
			for col := 0; col < co.Size; col++ {
				if conf[col] {
					cind++
					continue
				}
				if val, cok := rdata[col]; cok {
					rtn[row-rind] += vec[col-cind] * val
				}
			}
		}
	}
	return rtn
}
// func (co *COOMatrix) ToCRS() *CRSMatrix {
// 	nz := 0
// 	rtn := NewCRSMatrix(co.Size, co.nz)
// 	for row := 0; row < co.Size; row++ {
// 		rtn.row[row] = nz
// 		if rdata, rok := co.data[row]; rok {
// 			for col := 0; col < co.Size; col++ {
// 				if val, cok := rdata[col]; cok {
// 					rtn.value[nz] = val
// 					rtn.column[nz] = col
// 					nz++
// 				}
// 			}
// 		}
// 	}
// 	rtn.row[co.Size] = nz
// 	return rtn
// }

func (co *COOMatrix) ToCRS(csize int, conf []bool) *CRSMatrix {
	nz := 0
	size := co.Size - csize
	rtn := NewCRSMatrix(size, co.nz)
	rind := 0
	for row := 0; row < co.Size; row++ {
		if conf[row] {
			rind++
			continue
		}
		cind := 0
		rtn.row[row-rind] = nz
		if rdata, rok := co.data[row]; rok {
			for col := 0; col < co.Size; col++ {
				if conf[col] {
					cind++
					continue
				}
				if val, cok := rdata[col]; cok {
					rtn.value[nz] = val
					rtn.column[nz] = col - cind
					nz++
				}
			}
		}
	}
	rtn.row[size] = nz
	return rtn
}

func (co *COOMatrix) ToLLS(csize int, conf []bool) *LLSMatrix {
	var wg sync.WaitGroup
	size := co.Size - csize
	rtn := NewLLSMatrix(size)
	rind := 0
	for row := 0; row < co.Size; row++ {
		if conf[row] {
			rind++
			continue
		}
		wg.Add(1)
		go func(r, ri int) {
			var n *LLSNode
			defer wg.Done()
			cind := ri
			if rdata, rok := co.data[r]; rok {
				rtn.diag[r-ri].value=rdata[r]
				n = rtn.diag[r-ri]
				for col := r+1; col < co.Size; col++ {
					if conf[col] {
						cind++
						continue
					}
					if val, cok := rdata[col]; cok {
						newnode := NewLLSNode(col-cind, r-ri, val)
						n.down = newnode
						newnode.up = n
						n = newnode
					}
				}
			}
		}(row, rind)
	}
	wg.Wait()
	return rtn
}

func (co *COOMatrix) FELower(vec []float64) []float64 {
	for i := 0; i < co.Size; i++ {
		for j := 0; j < i; j++ {
			vec[i] -= co.Query(i, j) * vec[j]
		}
	}
	return vec
}

func (co *COOMatrix) FEUpper(vec []float64) []float64 {
	for i := 0; i < co.Size; i++ {
		for j := 0; j < i; j++ {
			vec[i] -= co.Query(j, i) * vec[j]
		}
	}
	return vec
}

func (co *COOMatrix) BSUpper(vec []float64) []float64 {
	n := co.Size
	for i := n - 1; i >= 0; i-- {
		for j := i + 1; j < n; j++ {
			vec[i] -= co.Query(i, j) * vec[j]
		}
	}
	return vec
}

func (co *COOMatrix) BSLower(vec []float64) []float64 {
	n := co.Size
	for i := n - 1; i >= 0; i-- {
		for j := i + 1; j < n; j++ {
			vec[i] -= co.Query(j, i) * vec[j]
		}
	}
	return vec
}

type CRSMatrix struct {
	Size   int
	nz     int
	value  []float64
	row    []int
	column []int
}

func NewCRSMatrix(size int, nz int) *CRSMatrix {
	rtn := new(CRSMatrix)
	rtn.Size = size
	rtn.nz = nz
	rtn.value = make([]float64, nz)
	rtn.row = make([]int, size+1)
	rtn.column = make([]int, nz)
	return rtn
}

func (cr *CRSMatrix) Copy() *CRSMatrix {
	rtn := NewCRSMatrix(cr.Size, cr.nz)
	for i := 0; i < cr.Size+1; i++ {
		rtn.row[i] = cr.row[i]
	}
	for i := 0; i < cr.nz; i++ {
		rtn.column[i] = cr.column[i]
		rtn.value[i] = cr.value[i]
	}
	return rtn
}

func ReadMtx(fn string) (*CRSMatrix, error) {
	f, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	var lis []string
	if ok := strings.HasSuffix(string(f), "\r\n"); ok {
		lis = strings.Split(string(f), "\r\n")
	} else {
		lis = strings.Split(string(f), "\n")
	}
	ind := 0
	for _, j := range lis {
		if strings.HasPrefix(j, "%") {
			ind++
			continue
		}
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
			ind++
			continue
		} else {
			break
		}
	}
	var words []string
	for _, k := range strings.Split(lis[ind], " ") {
		if k != "" {
			words = append(words, k)
		}
	}
	nums := make([]int, 3) // [row, col, nz]
	for i := 0; i < 3; i++ {
		num, err := strconv.ParseInt(words[i], 10, 64)
		if err != nil {
			return nil, err
		}
		nums[i] = int(num)
	}
	co := NewCOOMatrix(nums[0])
	for _, j := range lis[ind+1:] {
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
			continue
		}
		row, err := strconv.ParseInt(words[0], 10, 64)
		if err != nil {
			continue
		}
		col, err := strconv.ParseInt(words[1], 10, 64)
		if err != nil {
			continue
		}
		val, err := strconv.ParseFloat(words[2], 64)
		if err != nil {
			continue
		}
		co.Add(int(row)-1, int(col)-1, val)
		if row != col {
			co.Add(int(col)-1, int(row)-1, val)
		}
	}
	conf := make([]bool, nums[0])
	return co.ToCRS(0, conf), nil
}

func (cr *CRSMatrix) Print() string {
	var rtn bytes.Buffer
	for row := 0; row < cr.Size; row++ {
		last := 0
		for col := cr.row[row]; col < cr.row[row+1]; col++ {
			for i := 0; i < cr.column[col]-last; i++ {
				rtn.WriteString(fmt.Sprintf("%8.3f ", 0.0))
			}
			rtn.WriteString(fmt.Sprintf("%8.3f ", cr.value[col]))
			last = cr.column[col] + 1
		}
		for i := 0; i < cr.Size-last; i++ {
			rtn.WriteString(fmt.Sprintf("%8.3f ", 0.0))
		}
		rtn.WriteString("\n")
	}
	return rtn.String()
}

func (cr *CRSMatrix) String() string {
	var rtn bytes.Buffer
	for row := 0; row < cr.Size; row++ {
		for col := cr.row[row]; col < cr.row[row+1]; col++ {
			rtn.WriteString(fmt.Sprintf("%d %d %25.18f\n", row, cr.column[col], cr.value[col]))
		}
	}
	return rtn.String()
}

func (cr *CRSMatrix) Query(row, col int) float64 {
	if row > cr.Size || col > cr.Size {
		return 0.0
	}
	if cr.column[cr.row[row]] > col || cr.column[cr.row[row+1]-1] < col {
		return 0.0
	}
	for c := cr.row[row]; c < cr.row[row+1]; c++ {
		tmpcol := cr.column[c]
		if tmpcol > col {
			return 0.0
		}
		if tmpcol == col {
			return cr.value[c]
		}
	}
	return 0.0
}

func (cr *CRSMatrix) ToCOO() *COOMatrix {
	rtn := NewCOOMatrix(cr.Size)
	for row, val := range cr.row {
		rtn.Set(row, cr.column[val], cr.value[val])
	}
	return rtn
}

func (cr *CRSMatrix) SetFuncNZ(row, col int, val float64, f func(float64) float64) float64 {
	if row > cr.Size || col > cr.Size {
		return 0.0
	}
	if cr.column[cr.row[row]] > col || cr.column[cr.row[row+1]-1] < col {
		return 0.0
	}
	for c := cr.row[row]; c < cr.row[row+1]; c++ {
		tmpcol := cr.column[c]
		if tmpcol > col {
			return 0.0
		}
		if tmpcol == col {
			tmp := f(cr.value[c])
			cr.value[c] = tmp
			return tmp
		}
	}
	return 0.0
}

func (cr *CRSMatrix) SetFunc(row, col int, val float64, f func(float64) float64) float64 {
	if row > cr.Size || col > cr.Size {
		return 0.0
	}
	start := cr.row[row]
	end := cr.row[row+1]
	update := func(index int) {
		tmpcolumn := make([]int, cr.nz+1)
		tmpvalue := make([]float64, cr.nz+1)
		for i := 0; i < index; i++ {
			tmpcolumn[i] = cr.column[i]
			tmpvalue[i] = cr.value[i]
		}
		tmpcolumn[index] = col
		tmpvalue[index] = val
		for i := index; i < cr.nz; i++ {
			tmpcolumn[i+1] = cr.column[i]
			tmpvalue[i+1] = cr.value[i]
		}
		cr.column = tmpcolumn
		cr.value = tmpvalue
		for i := row + 1; i < cr.Size+1; i++ {
			cr.row[i]++
		}
		cr.nz++
	}
	if start == end || col < cr.column[start] {
		update(start)
		return val
	} else if cr.column[end-1] < col {
		update(end)
		return val
	} else {
		for c := start; c < end; c++ {
			tmpcol := cr.column[c]
			if tmpcol > col {
				update(c)
				return val
			}
			if cr.column[c] == col {
				tmp := f(cr.value[c])
				cr.value[c] = tmp
				return tmp
			}
		}
		return 0.0
	}
}

func (cr *CRSMatrix) Set(row, col int, val float64) float64 {
	if val == 0.0 {
		return 0.0
	}
	return cr.SetFunc(row, col, val, func(org float64) float64 { return val })
}

func (cr *CRSMatrix) Add(row, col int, val float64) float64 {
	if val == 0.0 {
		return 0.0
	}
	return cr.SetFunc(row, col, val, func(org float64) float64 { return org + val })
}

func (cr *CRSMatrix) Mul(row, col int, val float64) float64 {
	return cr.SetFuncNZ(row, col, val, func(org float64) float64 { return org * val })
}

func (cr *CRSMatrix) Sqrt(row, col int) float64 {
	return cr.SetFuncNZ(row, col, 0.0, func(org float64) float64 { return math.Sqrt(org) })
}

func (cr *CRSMatrix) MulV(vec []float64) []float64 {
	rtn := make([]float64, cr.Size)
	for row := 0; row < cr.Size; row++ {
		for col := cr.row[row]; col < cr.row[row+1]; col++ {
			tmpcol := cr.column[col]
			rtn[row] += vec[tmpcol] * cr.value[col]
		}
	}
	return rtn
}

// func (cr *CRSMatrix) FELower (vec []float64, divide bool) []float64 {
//     for i:=0; i<cr.Size; i++ {
//         for j:=0; j<i; j++ {
//             vec[i] -= cr.Query(i, j) * vec[j]
//         }
//         if divide {
//             vec[i] /= cr.Query(i, i)
//         }
//     }
//     return vec
// }

func (cr *CRSMatrix) FELower(vec []float64) []float64 {
	for row := 0; row < cr.Size; row++ {
		if cr.column[cr.row[row]] > row {
			continue
		}
		for c := cr.row[row]; c < cr.row[row+1]; c++ {
			tmpcol := cr.column[c]
			if tmpcol >= row {
				break
			}
			vec[row] -= cr.value[c] * vec[tmpcol]
		}
	}
	return vec
}

// func (cr *CRSMatrix) FEUpper (vec []float64, divide bool) []float64 {
//     for i:=0; i<cr.Size; i++ {
//         for j:=0; j<i; j++ {
//             vec[i] -= cr.Query(j, i) * vec[j]
//         }
//         if divide {
//             vec[i] /= cr.Query(i, i)
//         }
//     }
//     return vec
// }

// func (cr *CRSMatrix) BSUpper (vec []float64, divide bool) []float64 {
//     n := cr.Size
//     for i:=n-1; i>=0; i-- {
//         for j:=i+1; j<n; j++ {
//             vec[i] -= cr.Query(i, j) * vec[j]
//         }
//         if divide {
//             vec[i] /= cr.Query(i, i)
//         }
//     }
//     return vec
// }

func (cr *CRSMatrix) BSUpper(vec []float64) []float64 {
	n := cr.Size
	for row := n - 1; row >= 0; row-- {
		if cr.column[cr.row[row]] > row {
			continue
		}
		for c := cr.row[row]; c < cr.row[row+1]; c++ {
			tmpcol := cr.column[c]
			if tmpcol <= row {
				continue
			}
			vec[row] -= cr.value[c] * vec[tmpcol]
		}
	}
	return vec
}

// func (cr *CRSMatrix) BSLower (vec []float64, divide bool) []float64 {
//     n := cr.Size
//     for i:=n-1; i>=0; i-- {
//         for j:=i+1; j<n; j++ {
//             vec[i] -= cr.Query(j, i) * vec[j]
//         }
//         if divide {
//             vec[i] /= cr.Query(i, i)
//         }
//     }
//     return vec
// }

// func (cr *CRSMatrix) LDLT () *CRSMatrix {
//     rtn := cr.Copy()
//     n := cr.Size
//     for k:=0; k<n; k++ {
//         w := 1.0 / rtn.Query(k, k)
//         for i:=k+1; i<n; i++ {
//             rtn.Mul(k, i, w)
//         }
//         for j:=k+1; j<n; j++ {
//             v := rtn.Query(k, k) * rtn.Query(k, j)
//             for i:=j; i<n; i++ {
//                 rtn.Add(j, i, -v * rtn.Query(k, i))
//             }
//         }
//     }
//     for row:=0; row<n; row++ {
//         for col:= 0; col<row; col++ {
//             rtn.Set(row, col, rtn.Query(col, row))
//         }
//     }
//     return rtn
// }

func (cr *CRSMatrix) LDLT() *CRSMatrix {
	var d, w, v float64
	rtn := cr.Copy()
	n := cr.Size
	for row := 0; row < n; row++ {
		for c := rtn.row[row]; c < rtn.row[row+1]; c++ {
			tmpcol := rtn.column[c]
			if tmpcol < row {
				continue
			} else if tmpcol == row {
				d = rtn.value[c]
				w = 1.0 / d
			} else {
				rtn.value[c] *= w
			}
		}
		for c := rtn.row[row]; c < rtn.row[row+1]; c++ {
			tmpcol := rtn.column[c]
			if tmpcol <= row {
				continue
			}
			v = d * rtn.value[c]
			for cc := c; cc < rtn.row[row+1]; cc++ {
				rtn.Add(rtn.column[c], rtn.column[cc], -v*rtn.value[cc])
			}
		}
	}
	for row := 0; row < n; row++ {
		for col := 0; col < row; col++ {
			rtn.Set(row, col, rtn.Query(col, row))
		}
	}
	return rtn
}

func (cr *CRSMatrix) Chol() *CRSMatrix {
	rtn := cr.Copy()
	n := cr.Size
	for k := 0; k < n; k++ {
		val := rtn.Sqrt(k, k)
		w := 1.0 / val
		for i := k + 1; i < n; i++ {
			rtn.Mul(i, k, w)
		}
		for j := k + 1; j < n; j++ {
			for i := j; i < n; i++ {
				rtn.Add(i, j, -1.0*rtn.Query(i, k)*rtn.Query(j, k))
			}
		}
	}
	return rtn
}

func (cr *CRSMatrix) Solve(vecs ...[]float64) [][]float64 {
	start := time.Now()
	size := cr.Size
	C := cr.LDLT()
	end := time.Now()
	fmt.Printf("LDLT: %fsec\n", (end.Sub(start)).Seconds())
	rtn := make([][]float64, len(vecs))
	for v, vec := range vecs {
		tmp := make([]float64, size)
		for i := 0; i < size; i++ {
			tmp[i] = vec[i]
		}
		tmp = C.FELower(tmp)
		end = time.Now()
		fmt.Printf("FE: %fsec\n", (end.Sub(start)).Seconds())
		for i := 0; i < size; i++ {
			tmp[i] /= C.Query(i, i)
		}
		rtn[v] = C.BSUpper(tmp)
		end = time.Now()
		fmt.Printf("BS: %fsec\n", (end.Sub(start)).Seconds())
	}
	return rtn
}

func (cr *CRSMatrix) CG(vec []float64, eps float64) []float64 {
	size := cr.Size
	var alpha, beta float64
	x := make([]float64, size)
	r := make([]float64, size)
	p := make([]float64, size)
	for i := 0; i < size; i++ {
		x[i] = 0.0
		r[i] = vec[i]
		p[i] = vec[i]
	}
	bnorm := Dot(vec, vec, size)
	lnorm := 0.0
	rnorm := Dot(r, r, size)
	for {
		q := cr.MulV(p)
		alpha = rnorm / Dot(p, q, size)
		lnorm = rnorm
		for i := 0; i < size; i++ {
			x[i] += alpha * p[i]
			r[i] -= alpha * q[i]
		}
		rnorm = Dot(r, r, size)
		// fmt.Printf("%25.18f\n", rnorm/bnorm)
		if rnorm/bnorm < eps {
			return x
		}
		beta = rnorm / lnorm
		for i := 0; i < size; i++ {
			p[i] = r[i] + beta*p[i]
		}
	}
}

type LLSMatrix struct {
	Size int
	diag []*LLSNode
}

type LLSNode struct {
	row    int
	column int
	value  float64
	up     *LLSNode
	down   *LLSNode
}

func NewLLSMatrix(size int) *LLSMatrix {
	rtn := new(LLSMatrix)
	rtn.Size = size
	rtn.diag = make([]*LLSNode, size)
	for i := 0; i < size; i++ {
		rtn.diag[i] = NewLLSNode(i, i, 0.0)
	}
	return rtn
}

func NewLLSNode(row, col int, val float64) *LLSNode {
	if row < col {
		row, col = col, row
	}
	rtn := new(LLSNode)
	rtn.row = row
	rtn.column = col
	rtn.value = val
	return rtn
}

func (ln *LLSNode) String() string {
	return fmt.Sprintf("ROW: %d COL: %d VAL: %25.18f", ln.row, ln.column, ln.value)
}

func (ln *LLSNode) Before(n *LLSNode) { // TODO: Check
	ln.down = n
	if n.up != nil {
		ln.up = n.up
		n.up.down = ln
	}
	n.up = ln
}

func (ln *LLSNode) After(n *LLSNode) {
	ln.up = n
	if n.down != nil {
		ln.down = n.down
		n.down.up = ln
	}
	n.down = ln
}

func (ll *LLSMatrix) Print() string {
	var rtn bytes.Buffer
	var n *LLSNode
	size := ll.Size
	for row := 0; row < size; row++ {
		n = ll.diag[row]
		for i := 0; i < row; i++ {
			rtn.WriteString("         ")
		}
		rtn.WriteString(fmt.Sprintf("%8.3f ", n.value))
		last := row
		for {
			n = n.down
			if n == nil {
				break
			}
			for i := last; i < n.row-1; i++ {
				rtn.WriteString("         ")
			}
			rtn.WriteString(fmt.Sprintf("%8.3f ", n.value))
			last = n.row
		}
		rtn.WriteString("\n")
	}
	return rtn.String()
}

func (ll *LLSMatrix) String() string {
	var rtn bytes.Buffer
	var n *LLSNode
	size := ll.Size
	for row := 0; row < size; row++ {
		n = ll.diag[row]
		rtn.WriteString(fmt.Sprintf("%d %d %25.18f\n", n.row, n.column, n.value))
		for {
			n = n.down
			if n == nil {
				break
			}
			rtn.WriteString(fmt.Sprintf("%d %d %25.18f\n", n.row, n.column, n.value))
		}
	}
	return rtn.String()
}

func (ll *LLSMatrix) Query(row, col int) float64 {
	if row == col {
		return ll.diag[col].value
	}
	if row < col {
		row, col = col, row
	}
	var n *LLSNode
	n = ll.diag[col]
	for {
		n = n.down
		if n == nil {
			return 0.0
		}
		r := n.row
		if r < row {
			continue
		} else if r == row {
			return n.value
		} else { // r > row
			return 0.0
		}
	}
}

func (ll *LLSMatrix) Add(row, col int, val float64) float64 {
	if row == col {
		ll.diag[col].value += val
		return ll.diag[col].value
	}
	if row < col {
		row, col = col, row
	}
	var n *LLSNode
	n = ll.diag[col]
	for {
		if n.down == nil {
			newnode := NewLLSNode(row, col, val)
			newnode.After(n)
			return val
		}
		r := n.down.row
		if r < row {
			n = n.down
			continue
		} else if r == row {
			n.down.value += val
			return n.down.value
		} else { // n.down.row > row
			newnode := NewLLSNode(row, col, val)
			newnode.After(n)
			return val
		}
	}
}

func (ll *LLSMatrix) LDLT() *LLSMatrix {
	var n *LLSNode
	size := ll.Size
	for col := 0; col < size; col++ {
		n = ll.diag[col]
		w := 1.0 / n.value
		for {
			n = n.down
			if n == nil {
				break
			}
			n.value *= w
		}
		n = ll.diag[col]
		for {
			n = n.down
			if n == nil {
				break
			}
			v := ll.diag[col].value * n.value
			ni := n
			nj := ll.diag[n.row]
			for {
				if ni == nil {
					break
				}
				ci := ni.row
				val := -v * ni.value
				if nj.row > ci {
					newnode := NewLLSNode(n.row, ci, val)
					newnode.Before(nj)
				} else if nj.row == ci {
					nj.value += val
				} else {
					for {
						if nj.down == nil {
							newnode := NewLLSNode(n.row, ci, val)
							newnode.After(nj)
							nj = newnode
							break
						}
						if nj.down.row > ci {
							newnode := NewLLSNode(n.row, ci, val)
							newnode.After(nj)
							nj = nj.down
							break
						} else if nj.down.row == ci {
							nj.down.value += val
							nj = nj.down
							break
						}
						nj = nj.down
					}
				}
				ni = ni.down
			}
		}
	}
	return ll
}

func (ll *LLSMatrix) ILDLT() *LLSMatrix {
	var n *LLSNode
	size := ll.Size
	for col := 0; col < size; col++ {
		n = ll.diag[col]
		w := 1.0 / n.value
		for {
			n = n.down
			if n == nil {
				break
			}
			n.value *= w
		}
		n = ll.diag[col]
		for {
			n = n.down
			if n == nil {
				break
			}
			v := ll.diag[col].value * n.value
			ni := n
			nj := ll.diag[n.row]
			for {
				if ni == nil {
					break
				}
				ci := ni.row
				val := -v * ni.value
				if nj.row == ci {
					nj.value += val
				} else if nj.row < ci {
					for {
						if nj.down == nil {
							break
						}
						if nj.down.row == ci {
							nj.down.value += val
							nj = nj.down
							break
						} else {
							nj = nj.down
							break
						}
					}
				}
				ni = ni.down
			}
		}
	}
	return ll
}

func (ll *LLSMatrix) DiagUp() {
	var n *LLSNode
	for col := 0; col < ll.Size; col++ {
		n = ll.diag[col]
		for {
			n = n.down
			if n == nil {
				break
			}
			n.up = ll.diag[n.row].up
			ll.diag[n.row].up = n
		}
	}
}

func (ll *LLSMatrix) FELower(vec []float64) []float64 {
	var n *LLSNode
	for i := 0; i < ll.Size; i++ {
		n = ll.diag[i]
		for {
			n = n.up
			if n == nil {
				break
			}
			vec[i] -= n.value * vec[n.column]
		}
	}
	return vec
}

func (ll *LLSMatrix) BSUpper(vec []float64) []float64 {
	m := ll.Size
	var n *LLSNode
	for i := m - 1; i >= 0; i-- {
		n = ll.diag[i]
		for {
			n = n.down
			if n == nil {
				break
			}
			vec[i] -= n.value * vec[n.row]
		}
	}
	return vec
}

func (ll *LLSMatrix) Solve(vecs ...[]float64) [][]float64 {
	size := ll.Size
	C := ll.LDLT()
	rtn := make([][]float64, len(vecs))
	C.DiagUp()
	for v, vec := range vecs {
		tmp := make([]float64, size)
		for i := 0; i < size; i++ {
			tmp[i] = vec[i]
		}
		tmp = C.FELower(tmp)
		for i := 0; i < size; i++ {
			tmp[i] /= C.Query(i, i)
		}
		rtn[v] = C.BSUpper(tmp)
	}
	return rtn
}

func (ll *LLSMatrix) MulV(vec []float64) []float64 {
	var n *LLSNode
	rtn := make([]float64, ll.Size)
	for i := 0; i < ll.Size; i++ {
		n = ll.diag[i]
		for {
			n = n.up
			if n == nil {
				break
			}
			rtn[i] += vec[n.column] * n.value
		}
		n = ll.diag[i]
		rtn[i] += vec[i] * n.value
		for {
			n = n.down
			if n == nil {
				break
			}
			rtn[i] += vec[n.row] * n.value
		}
	}
	return rtn
}

func (ll *LLSMatrix) CG(vec []float64, eps float64) []float64 {
	size := ll.Size
	var alpha, beta float64
	x := make([]float64, size)
	r := make([]float64, size)
	p := make([]float64, size)
	for i := 0; i < size; i++ {
		x[i] = 0.0
		r[i] = vec[i]
		p[i] = vec[i]
	}
	bnorm := Dot(vec, vec, size)
	lnorm := 0.0
	rnorm := Dot(r, r, size)
	for {
		q := ll.MulV(p)
		alpha = rnorm / Dot(p, q, size)
		lnorm = rnorm
		for i := 0; i < size; i++ {
			x[i] += alpha * p[i]
			r[i] -= alpha * q[i]
		}
		rnorm = Dot(r, r, size)
		// fmt.Printf("%25.18f\n", rnorm/bnorm)
		if rnorm/bnorm < eps {
			return x
		}
		beta = rnorm / lnorm
		for i := 0; i < size; i++ {
			p[i] = r[i] + beta*p[i]
		}
	}
}

// TODO: test
func (ll *LLSMatrix) PCG(C *LLSMatrix, vec []float64) []float64 {
	size := ll.Size
	C = C.ILDLT()
	C.DiagUp()
	var mu, nu, alpha, beta float64
	x := make([]float64, size)
	r := make([]float64, size)
	p := make([]float64, size)
	for i := 0; i < size; i++ {
		x[i] = 0.0
		r[i] = vec[i]
		p[i] = vec[i]
	}
	cct := func (mat *LLSMatrix, v []float64) []float64 {
		v = mat.FELower(v)
		for i := 0; i < size; i++ {
			v[i] /= mat.Query(i, i)
		}
		v = mat.BSUpper(v)
		return v
	}
	p = cct(C, p)
	lo := Dot(r, p, size)
	bnorm := Dot(vec, vec, size)
	rnorm := Dot(r, r, size)
	var q []float64
	for {
		q = ll.MulV(p)
		nu = Dot(p, q, size)
		alpha = rnorm / nu
		for i := 0; i < size; i++ {
			x[i] += alpha * p[i]
			r[i] -= alpha * q[i]
			q[i] = r[i]
		}
		rnorm = Dot(r, r, size)
		fmt.Printf("%25.18f\n", rnorm/bnorm)
		if rnorm/bnorm < 1e-16 {
			return x
		}
		q = cct(C, q)
		mu = Dot(q, r, size)
		beta = mu / lo
		lo = mu
		for i := 0; i < size; i++ {
			p[i] = q[i] + beta*p[i]
		}
	}
}

func Dot(x, y []float64, size int) float64 {
	rtn := 0.0
	for i := 0; i < size; i++ {
		rtn += x[i] * y[i]
	}
	return rtn
}
