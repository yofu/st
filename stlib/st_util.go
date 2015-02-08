package st

import (
	"bufio"
	"bytes"
	"code.google.com/p/go.text/encoding"
	"code.google.com/p/go.text/encoding/japanese"
	"code.google.com/p/go.text/transform"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	RainbowColor = []int{38655, 65430, 65280, 9895680, 16507473, 16750130, 16711830} //  "BLUE", "BLUEGREEN", "GREEN", "YELLOWGREEN", "YELLOW", "ORANGE", "RED"
)

func Input(question string) string {
	fmt.Print(question)
	var answer string
	fmt.Scanln(&answer)
	return answer
}

func Unique(lis []interface{}) []interface{} {
	var i int
	var add bool
	tmp := make([]interface{}, len(lis))
	for _, val := range lis {
		add = true
		for _, r := range tmp {
			if val == r {
				add = false
				break
			}
		}
		if add {
			tmp = append(tmp, add)
			i++
		}
	}
	rtn := make([]interface{}, i)
	for j := 0; j < i; j++ {
		rtn[j] = tmp[i]
	}
	return rtn
}

func Inpfiles(path string) (inps []string, err error) {
	var (
		stat os.FileInfo
		f    *os.File
		lis  []string
	)
	stat, err = os.Stat(path)
	if err != nil {
		return
	}
	if stat.IsDir() {
		fmt.Printf("SearchInp: %s\n", path)
		f, err = os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()
		lis, err = f.Readdirnames(-1)
		if err != nil {
			return
		}
		for _, j := range lis {
			if filepath.Ext(j) == ".inp" {
				inps = append(inps, j)
			}
		}
		return
	}
	return
}

func SelectInp(path string) (fn string, err error) {
	var (
		stat os.FileInfo
		f    *os.File
		lis  []string
		sel  string
		val  int64
		i    int
	)
	stat, err = os.Stat(path)
	if err != nil {
		return
	}
	if stat.IsDir() {
		fmt.Printf("SearchInp: %s\n", path)
		f, err = os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()
		lis, err = f.Readdirnames(-1)
		if err != nil {
			return
		}
		inps := make([]string, len(lis))
		for _, j := range lis {
			if filepath.Ext(j) == ".inp" {
				inps[i] = j
				fmt.Printf("[%2d]: %s\n", i, j)
				i += 1
			}
		}
		sel = Input("Select Inpfile: ")
		val, err = strconv.ParseInt(sel, 10, 64)
		if err != nil {
			return
		}
		if 0 <= val && int(val) < i {
			return filepath.Join(path, inps[val]), nil
		} else {
			return
		}
	} else {
		if filepath.Ext(path) == ".inp" {
			return path, nil
		} else {
			return
		}
	}
}

// TODO: Is it possible to make filepath.Walk working concurrerntly?
func SearchInp(dirname string) chan string {
	rtn := make(chan string)
	go func() {
		filepath.Walk(dirname,
			func(path string, info os.FileInfo, err error) error {
				stat, err := os.Stat(path)
				if err != nil {
					return err
				}
				if !stat.IsDir() {
					if filepath.Ext(path) == ".inp" {
						rtn <- path
					}
					return nil
				}
				return nil
			})
		rtn <- ""
		defer close(rtn)
	}()
	return rtn
}

func ColorInt(str string) (rtn int) {
	sep, _ := regexp.Compile(" +")
	lis := sep.Split(str, -1)
	val := 65536
	for j := 0; j < 3; j++ {
		tmpcol, err := strconv.ParseInt(lis[j], 10, 64)
		if err != nil {
			return 0
		}
		rtn += int(tmpcol) * val
		val >>= 8
	}
	return
}

func IntColorList(col int) []string {
	rtn := make([]string, 3)
	val := 65536
	for i := 0; i < 3; i++ {
		tmp := 0
		for {
			if col >= val {
				col -= val
				tmp += 1
			} else {
				rtn[i] = fmt.Sprintf("%d", tmp)
				break
			}
		}
		val >>= 8
	}
	return rtn
}

func IntColor(col int) string {
	rtn := make([]string, 3)
	val := 65536
	for i := 0; i < 3; i++ {
		tmp := 0
		for {
			if col >= val {
				col -= val
				tmp += 1
			} else {
				rtn[i] = fmt.Sprintf("%3d", tmp)
				break
			}
		}
		val >>= 8
	}
	return strings.Join(rtn, " ")
}

func Rainbow(val float64, boundary []float64) int {
	var ind int
	for _, bound := range boundary {
		if val <= bound {
			return RainbowColor[ind]
		}
		ind++
	}
	return RainbowColor[ind]
}

func ListMax(list []float64) float64 {
	l := len(list)
	if l == 0 {
		return 0.0
	} else if l == 1 {
		return list[0]
	} else {
		rtn := list[0]
		for _, val := range list[1:] {
			if val >= rtn {
				rtn = val
			}
		}
		return rtn
	}
}

func Normalize(vec []float64) []float64 {
	sum := 0.0
	for _, val := range vec {
		sum += val * val
	}
	sum = math.Sqrt(sum)
	if sum == 0.0 {
		return vec
	}
	for i := range vec {
		vec[i] /= sum
	}
	return vec
}

func Dot(x, y []float64, size int) float64 {
	rtn := 0.0
	for i := 0; i < size; i++ {
		rtn += x[i] * y[i]
	}
	return rtn
}

func Cross(x, y []float64) []float64 {
	rtn := make([]float64, 3)
	rtn[0] = x[1]*y[2] - x[2]*y[1]
	rtn[1] = x[2]*y[0] - x[0]*y[2]
	rtn[2] = x[0]*y[1] - x[1]*y[0]
	return rtn
}

func RotateVector(coord, center, vector []float64, angle float64) []float64 {
	r := make([]float64, 3)
	rtn := make([]float64, 3)
	for i := 0; i < 3; i++ {
		r[i] = coord[i] - center[i]
	}
	n := Normalize(vector)
	d1 := Dot(n, r, 3)
	c1 := Cross(r, n)
	for i := 0; i < 3; i++ {
		rtn[i] = d1*n[i] + (r[i]-d1*n[i])*math.Cos(angle) - c1[i]*math.Sin(angle) + center[i]
	}
	return rtn
}

func Ce(fn, ext string) string {
	if len(ext) == 0 {
		return strings.Replace(fn, filepath.Ext(fn), "", 1)
	}
	if ext[0] != '.' {
		ext = fmt.Sprintf(".%s", ext)
	}
	defext := filepath.Ext(fn)
	if defext == "" {
		return fmt.Sprintf("%s%s", fn, ext)
	} else {
		return strings.Replace(fn, defext, ext, 1)
	}
}

func Increment(fn, div string, pos, times int) (string, error) {
	ext := filepath.Ext(fn)
	ls := strings.Split(strings.Replace(fn, ext, "", 1), div)
	if len(ls) == pos {
		return fmt.Sprintf("%s%s%02d%s", strings.Replace(fn, ext, "", 1), div, times, ext), nil
	} else if len(ls) < pos+1 {
		return fn, errors.New("Increment: IndexError")
	}
	if pos == 0 {
		pat := regexp.MustCompile("^([^0-9]+)([0-9]+)$")
		tmp := pat.FindStringSubmatch(ls[0])
		if len(tmp) < 3 {
			return fn, errors.New("Increment: ValueError")
		}
		val, _ := strconv.ParseInt(tmp[2], 10, 64)
		ls[0] = fmt.Sprintf("%s%02d", tmp[1], int(val)+times)
	} else {
		num := ls[pos]
		val, err := strconv.ParseInt(num, 10, 64)
		if err != nil {
			return fn, err
		}
		ls[pos] = fmt.Sprintf("%02d", int(val)+times)
	}
	return strings.Join(ls, "_") + ext, nil
}

func PruneExt(fn string) string {
	ext := filepath.Ext(fn)
	return strings.Replace(fn, ext, "", 1)
}

func FileExists(fn string) bool {
	if _, err := os.Stat(fn); err == nil {
		return true
	} else {
		return false
	}
}

func CopyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer df.Close()
	io.Copy(df, sf)
	return nil
}

func ProjectName(fn string) string {
	pat := regexp.MustCompile("^[^0-9._]+")
	return pat.FindString(filepath.Base(fn))
}

func IsParallel(v1, v2 []float64, eps float64) bool {
	var dot, l1, l2 float64
	for i := 0; i < 3; i++ {
		dot += v1[i] * v2[i]
		l1 += v1[i] * v1[i]
		l2 += v2[i] * v2[i]
	}
	if l1 == 0 || l2 == 0 {
		return false
	}
	sub := (dot * dot / (l1 * l2)) - 1.0
	if math.Abs(sub) < eps {
		return true
	} else {
		return false
	}
}

func IsOrthogonal(v1, v2 []float64, eps float64) bool {
	var dot, l1, l2 float64
	for i := 0; i < 3; i++ {
		dot += v1[i] * v2[i]
		l1 += v1[i] * v1[i]
		l2 += v2[i] * v2[i]
	}
	if l1 == 0 || l2 == 0 {
		return false
	}
	sub := (dot * dot / (l1 * l2))
	if math.Abs(sub) < eps {
		return true
	} else {
		return false
	}
}

func ClockWise(p1, p2, p3 []float64) (float64, bool) {
	v1 := []float64{p2[0] - p1[0], p2[1] - p1[1]}
	v2 := []float64{p3[0] - p1[0], p3[1] - p1[1]}
	var sum1, sum2 float64
	for i := 0; i < 2; i++ {
		sum1 += v1[i] * v1[i]
		sum2 += v2[i] * v2[i]
	}
	if sum1 == 0 || sum2 == 0 {
		return 0.0, false
	}
	sum1 = math.Sqrt(sum1)
	sum2 = math.Sqrt(sum2)
	for i := 0; i < 2; i++ {
		v1[i] /= sum1
		v2[i] /= sum2
	}
	if val := v2[0]*v1[1] - v2[1]*v1[0]; val > 0 {
		return math.Atan2(val, v1[0]*v2[0]+v1[1]*v2[1]), true
	} else {
		return math.Atan2(val, v1[0]*v2[0]+v1[1]*v2[1]), false
	}
}

func ClockWise2(p1, p2, p3 []float64) (float64, bool) {
	v1 := []float64{p2[0] - p1[0], p2[1] - p1[1]}
	v2 := []float64{p3[0] - p2[0], p3[1] - p2[1]}
	var sum1, sum2 float64
	for i := 0; i < 2; i++ {
		sum1 += v1[i] * v1[i]
		sum2 += v2[i] * v2[i]
	}
	if sum1 == 0 || sum2 == 0 {
		return 0.0, false
	}
	sum1 = math.Sqrt(sum1)
	sum2 = math.Sqrt(sum2)
	for i := 0; i < 2; i++ {
		v1[i] /= sum1
		v2[i] /= sum2
	}
	if val := v2[0]*v1[1] - v2[1]*v1[0]; val > 0 {
		return math.Acos(v1[0]*v2[0] + v1[1]*v2[1]), true
	} else {
		return math.Acos(v1[0]*v2[0] + v1[1]*v2[1]), false
	}
}

func DistLineLine(coord1, vec1, coord2, vec2 []float64) (float64, float64, float64, error) {
	var l1, l2 float64
	for i := 0; i < 3; i++ {
		l1 += vec1[i] * vec1[i]
		l2 += vec2[i] * vec2[i]
	}
	if l1 == 0 || l2 == 0 {
		return 0.0, 0.0, 0.0, errors.New("DistLineLine: Zero Vector")
	}
	l1 = math.Sqrt(l1)
	l2 = math.Sqrt(l2)
	a := make([]float64, 3)
	var uv, tmp, ua float64
	for i := 0; i < 3; i++ {
		vec1[i] /= l1
		vec2[i] /= l2
		tmp = vec1[i] * vec2[i]
		uv += tmp
		a[i] = coord2[i] - coord1[i]
		ua += vec1[i] * a[i]
	}
	det := 1.0 - uv*uv
	if det <= 1e-4 { // vec1 is parallel to vec2
		la := 0.0
		for i := 0; i < 3; i++ {
			la += a[i] * a[i]
		}
		return 0.0, 0.0, math.Sqrt(la - ua*ua), errors.New("DistLineLine: Parallel")
	} else {
		va := 0.0
		for i := 0; i < 3; i++ {
			va += vec2[i] * a[i]
		}
		s := (ua - uv*va) / det
		t := (uv*ua - va) / det
		d := 0.0
		for i := 0; i < 3; i++ {
			tmp := a[i] - s*vec1[i] + t*vec2[i]
			d += tmp * tmp
		}
		return s / l1, t / l2, math.Sqrt(d), nil
	}
}

func ParseFile(filename string, do func([]string) error) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		var words []string
		for _, k := range strings.Split(s.Text(), " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
			continue
		}
		err := do(words)
		if err != nil {
			return err
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func AddCR(before bytes.Buffer) bytes.Buffer {
	var after bytes.Buffer
	after.WriteString(strings.Replace(before.String(), "\n", "\r\n", -1))
	return after
}

func OnTheSameLine(n1, n2, n3 []float64, eps float64) bool {
	v1 := make([]float64, 3)
	v2 := make([]float64, 3)
	sum1 := 0.0
	sum2 := 0.0
	for i := 0; i < 3; i++ {
		v1[i] = n2[i] - n1[i]
		v2[i] = n3[i] - n1[i]
		sum1 += v1[i] * v1[i]
		sum2 += v2[i] * v2[i]
	}
	sum1 = math.Sqrt(sum1)
	sum2 = math.Sqrt(sum2)
	if sum1 < eps || sum2 < eps {
		return true
	}
	for i := 0; i < 3; i++ {
		v1[i] /= sum1
		v2[i] /= sum2
	}
	return math.Abs(v1[1]*v2[2]-v2[1]*v1[2]) < eps && math.Abs(v1[2]*v2[0]-v2[2]*v1[0]) < eps && math.Abs(v1[0]*v2[1]-v2[0]*v1[1]) < eps
}

func OnTheSamePlane(n1, n2, n3, n4 []float64, eps float64) bool {
	v1 := make([]float64, 3)
	v2 := make([]float64, 3)
	v3 := make([]float64, 3)
	sum1 := 0.0
	sum2 := 0.0
	sum3 := 0.0
	for i := 0; i < 3; i++ {
		v1[i] = n2[i] - n1[i]
		v2[i] = n3[i] - n1[i]
		v3[i] = n4[i] - n1[i]
		sum1 += v1[i] * v1[i]
		sum2 += v2[i] * v2[i]
		sum3 += v3[i] * v3[i]
	}
	sum1 = math.Sqrt(sum1)
	sum2 = math.Sqrt(sum2)
	sum3 = math.Sqrt(sum3)
	if sum1 < eps || sum2 < eps || sum3 < eps {
		return true
	}
	for i := 0; i < 3; i++ {
		v1[i] /= sum1
		v2[i] /= sum2
		v3[i] /= sum3
	}
	return math.Abs(v1[0]*(v2[1]*v3[2]-v3[1]*v2[2])+v1[1]*(v2[2]*v3[0]-v3[2]*v2[0])+v1[2]*(v2[0]*v3[1]-v3[0]*v2[1])) < eps
}

func Convert(str string) string {
	_, _, err := transform.String(encoding.UTF8Validator, str)
	if err != nil {
		for _, enc := range []encoding.Encoding{japanese.ShiftJIS, japanese.EUCJP, japanese.ISO2022JP} {
			tmp, _, err := transform.String(enc.NewDecoder(), str)
			if err != nil {
				continue
			}
			return tmp
		}
	}
	return str
}
