package matrix

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"strconv"
	"strings"
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

func (co *COOMatrix) ToCRS() *CRSMatrix {
	nz := 0
	rtn := NewCRSMatrix(co.Size, co.nz)
	for row := 0; row < co.Size; row++ {
		rtn.row[row] = nz
		if rdata, rok := co.data[row]; rok {
			for col := 0; col < co.Size; col++ {
				if val, cok := rdata[col]; cok {
					rtn.value[nz] = val
					rtn.column[nz] = col
					nz++
				}
			}
		}
	}
	rtn.row[co.Size] = nz
	return rtn
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
	return co.ToCRS(), nil
}

func (cr *CRSMatrix) String() string {
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

func (cr *CRSMatrix) FELower(pass []bool, vec []float64) []float64 {
	for row := 0; row < cr.Size; row++ {
		if pass[row] {
			continue
		}
		if cr.column[cr.row[row]] > row {
			continue
		}
		for c := cr.row[row]; c < cr.row[row+1]; c++ {
			tmpcol := cr.column[c]
			if tmpcol >= row {
				break
			}
			if pass[tmpcol] {
				continue
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

func (cr *CRSMatrix) BSUpper(pass []bool, vec []float64) []float64 {
	n := cr.Size
	for row := n - 1; row >= 0; row-- {
		if pass[row] {
			continue
		}
		if cr.column[cr.row[row]] > row {
			continue
		}
		for c := cr.row[row]; c < cr.row[row+1]; c++ {
			tmpcol := cr.column[c]
			if tmpcol <= row {
				continue
			}
			if pass[tmpcol] {
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

func (cr *CRSMatrix) LDLT(pass []bool) *CRSMatrix {
	var d, w, v float64
	rtn := cr.Copy()
	n := cr.Size
	for row := 0; row < n; row++ {
		if pass[row] {
			continue
		}
		for c := rtn.row[row]; c < rtn.row[row+1]; c++ {
			tmpcol := rtn.column[c]
			if pass[tmpcol] {
				continue
			}
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
			if pass[tmpcol] {
				continue
			}
			v = d * rtn.value[c]
			for cc := c; cc < rtn.row[row+1]; cc++ {
				if pass[cc] {
					continue
				}
				rtn.Add(rtn.column[c], rtn.column[cc], -v*rtn.value[cc])
			}
		}
	}
	for row := 0; row < n; row++ {
		if pass[row] {
			continue
		}
		for col := 0; col < row; col++ {
			if pass[col] {
				continue
			}
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

func (cr *CRSMatrix) Solve(pass []bool, vecs ...[]float64) [][]float64 {
	size := cr.Size
	C := cr.LDLT(pass)
	rtn := make([][]float64, len(vecs))
	for v, vec := range vecs {
		tmp := make([]float64, size)
		for i := 0; i < size; i++ {
			tmp[i] = vec[i]
		}
		tmp = C.FELower(pass, tmp)
		for i := 0; i < size; i++ {
			if pass[i] {
				continue
			}
			tmp[i] /= C.Query(i, i)
		}
		rtn[v] = C.BSUpper(pass, tmp)
	}
	return rtn
}

func (cr *CRSMatrix) CG(vec []float64) []float64 {
	size := cr.Size
	var alpha, beta float64
	x := make([]float64, size)
	r := make([]float64, size)
	p := make([]float64, size)
	rand.Seed(int64(time.Now().Nanosecond()))
	for i := 0; i < size; i++ {
		x[i] = rand.Float64()
		r[i] = vec[i]
		p[i] = vec[i]
	}
	bnorm := Dot(vec, vec, size)
	lnorm := 0.0
	rnorm := Dot(r, r, size)
	for k := 0; k < size; k++ {
		q := cr.MulV(p)
		alpha = rnorm / Dot(p, q, size)
		lnorm = rnorm
		for i := 0; i < size; i++ {
			x[i] += alpha * p[i]
			r[i] -= alpha * q[i]
		}
		rnorm = Dot(r, r, size)
		// fmt.Println(rnorm/bnorm)
		if rnorm/bnorm < 1e-12 {
			fmt.Println(k)
			return x
		}
		beta = rnorm / lnorm
		for i := 0; i < size; i++ {
			p[i] = r[i] + beta*p[i]
		}
	}
	return x
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
	rtn := new(LLSNode)
	rtn.row = row
	rtn.column = col
	rtn.value = val
	return rtn
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

func (ll *LLSMatrix) String() string {
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

func (ll *LLSMatrix) LDLT() *LLSMatrix { // TODO: Test
	var n *LLSNode
	size := ll.Size
	for row := 0; row < size; row++ {
		n = ll.diag[row]
		w := 1.0 / n.value
		for {
			n = n.down
			if n == nil {
				break
			}
			n.value *= w
		}
		n = ll.diag[row]
		for {
			n = n.down
			if n == nil {
				break
			}
			v := ll.diag[row].value * n.value
			ni := n
			nj := ll.diag[n.column]
			for { // ni: (row, i) down to (row, size)
				ci := ni.column
				cj := nj.column
				ni = ni.down
				if ni == nil {
					break
				}
				val := -v * ni.value
				if ci == cj {
					nj.value += val
					nj = nj.down
				} else if ci > cj {
					for {
						if nj.down == nil {
							newnode := NewLLSNode(row, ci, val)
							newnode.After(nj)
						}
						cj = nj.down.column
						if ci > cj {
							nj = nj.down
							continue
						} else if ci == cj {
							nj.down.value += val
							break
						} else {
							newnode := NewLLSNode(row, ci, val)
							newnode.After(nj)
							break
						}
					}
				} else {
					newnode := NewLLSNode(row, ci, val)
					newnode.Before(nj)
				}
			}
		}
	}
	return ll
}

func Dot(x, y []float64, size int) float64 {
	rtn := 0.0
	for i := 0; i < size; i++ {
		rtn += x[i] * y[i]
	}
	return rtn
}

// func main() {
//     // test, err := ReadMtx("LFAT5.mtx")
//     // if err != nil {
//     //     return
//     // }

//     // vec := make([]float64, test.size)
//     // rand.Seed(int64(time.Now().Nanosecond()))
//     // for i:=0; i<test.size; i++ {
//     //     vec[i] = rand.Float64()
//     // }
//     // fmt.Println(test)
//     // fmt.Println(test.Query(5,1))
//     // ans := test.CG(vec)
//     // fmt.Println(ans)
//     // fmt.Println(vec)
//     // fmt.Println(test.Mul(ans))
//     test := NewLLSMatrix(4)
//     test.Add(0, 0, 1.0)
//     test.Add(1, 1, 7.0)
//     test.Add(2, 2, 8.0)
//     test.Add(3, 3, 9.0)
//     test.Add(0, 1, 2.0)
//     // test.Add(1, 0, 2.0)
//     test.Add(0, 2, 3.0)
//     // test.Add(2, 0, 3.0)
//     test.Add(1, 3, 4.0)
//     // test.Add(3, 1, 4.0)
//     test.Add(2, 3, 6.0)
//     // test.Add(3, 2, 6.0)
//     fmt.Println(test.Query(3 ,1))
//     fmt.Println(test)
//     // crs := test.ToCRS()
//     // crs.Add(3, 0, 2.0)

//     // vec1 := make([]float64, test.size)
//     // vec2 := make([]float64, test.size)
//     // vec3 := make([]float64, test.size)
//     // vecs := [][]float64{vec1, vec2, vec3}
//     // for i:=0; i<test.size; i++ {
//     //     vecs[i%3][i] = float64(i)
//     // }
//     // ans := test.Solve(vec1, vec2, vec3)
//     // for i:=0; i<3; i++ {
//     //     fmt.Println(vecs[i])
//     //     fmt.Println(test.MulV(ans[i]))
//     // }

//     // fmt.Println(test.Chol())
//     // fmt.Println(crs.row)
//     // fmt.Println(crs.column)
//     // fmt.Println(crs.value)
//     // vec := []float64{ 0.0, 1.0, 0.0, 1.0 }
//     // vec2 := []float64{ 0.0, 0.5, 0.0, 0.1 }
//     // ans := crs.CG(vec2)
//     // fmt.Println(ans)
//     // fmt.Println(crs.Mul(ans))
//     // fmt.Println(crs.Mul(vec))
//     // fmt.Println(crs.Mul(vec2))
// }