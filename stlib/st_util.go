package st

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	RainbowColor = []int{38655, 65430, 65280, 9895680, 16507473, 16750130, 16711830} //  "BLUE", "BLUEGREEN", "GREEN", "YELLOWGREEN", "YELLOW", "ORANGE", "RED"
)
const (
    hextable = "0123456789abcdef"
)

var (
	home            = os.Getenv("HOME")
	tooldir  = filepath.Join(home, ".st/tool")
)

type Hider interface {
	Hide()
	Show()
	IsHidden(*Show) bool
}

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

func IntColorList(col int) []int {
	rtn := make([]int, 3)
	val := 65536
	for i := 0; i < 3; i++ {
		tmp := 0
		for {
			if col >= val {
				col -= val
				tmp += 1
			} else {
				rtn[i] = tmp
				break
			}
		}
		val >>= 8
	}
	return rtn
}

func IntColorFloat64(col int) []float64 {
	rtn := make([]float64, 3)
	val := 65536
	for i := 0; i < 3; i++ {
		tmp := 0
		for {
			if col >= val {
				col -= val
				tmp += 1
			} else {
				rtn[i] = float64(tmp) / 255.0
				break
			}
		}
		val >>= 8
	}
	return rtn
}

func IntColorStringList(col int) []string {
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

func IntHexColor(col int) string {
    var rtn bytes.Buffer
    val := 1048576
    rtn.WriteString("#")
    for i:=0; i<6; i++ {
        tmp := 0
        for {
            if col>=val {
                col -= val
                tmp += 1
            } else {
                rtn.WriteByte(hextable[tmp])
                break
            }
        }
        val >>= 4
    }
    return rtn.String()
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
	// return PseudoColor(val, boundary[len(boundary) - 1], boundary[0])
}

func PseudoColor(val0, max, min float64) int {
	var val float64
	if max == min {
		val = val0
	} else {
		if min > max {
			max, min = min, max
		}
		val = (val0 - min) / (max - min)
	}
	if val > 1.0 {
		return 0xff0000
	} else if val > 0.75 {
		return 0xff0000 + int(255*4*(1.0-val))<<8
	} else if val > 0.5 {
		return 0x00ff00 + int(255*4*(val-0.5))<<16
	} else if val > 0.25 {
		return 0x00ff00 + int(255*4*(0.5-val))
	} else if val > 0.0 {
		return 0x0000ff + int(255*4*val)<<8
	} else {
		return 0x0000ff
	}
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
	return strings.Join(ls, div) + ext, nil
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
	j := 0
	val := 0.0
	for i := 0; i < 3; i++ {
		j = i + 1
		if j >= 3 {
			j -= 3
		}
		val = v1[i]*v2[j] - v1[j]*v2[i]
		if (val > 2*eps) || (val < -2*eps) {
			return false
		}
	}
	return true
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

func ClockWiseInt(p1, p2, p3 []int) (float64, bool) {
	v1 := []float64{float64(p2[0] - p1[0]), float64(p2[1] - p1[1])}
	v2 := []float64{float64(p3[0] - p1[0]), float64(p3[1] - p1[1])}
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

func ToUtf8string(str string) string {
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

func Vim(fn string) {
	cmd := exec.Command("gvim", fn)
	cmd.Start()
}

func Explorer(dir string) {
	stat, err := os.Stat(dir)
	if err != nil || !stat.IsDir() {
		dir = "."
	}
	cmd := exec.Command("cmd", "/C", "explorer", dir)
	cmd.Start()
}

func Edit(fn string) {
	cmd := exec.Command("cmd", "/C", "start", fn)
	cmd.Start()
}

func EditReadme(dir string) {
	fn := filepath.Join(dir, "readme.txt")
	Vim(fn)
}

func StartTool(fn string) {
	cmd := exec.Command("cmd", "/C", "start", filepath.Join(tooldir, fn))
	cmd.Start()
}

func SplitNums(nums string) []int {
	sectrange := regexp.MustCompile("(?i)^ *range *[(] *([0-9]+) *, *([0-9]+) *[)] *$")
	if sectrange.MatchString(nums) {
		fs := sectrange.FindStringSubmatch(nums)
		start, err := strconv.ParseInt(fs[1], 10, 64)
		end, err := strconv.ParseInt(fs[2], 10, 64)
		if err != nil {
			return nil
		}
		if start > end {
			return nil
		}
		sects := make([]int, int(end-start))
		for i := 0; i < int(end-start); i++ {
			sects[i] = i + int(start)
		}
		return sects
	} else {
		splitter := regexp.MustCompile("[, ]")
		tmp := splitter.Split(nums, -1)
		rtn := make([]int, len(tmp))
		i := 0
		for _, numstr := range tmp {
			val, err := strconv.ParseInt(strings.Trim(numstr, " "), 10, 64)
			if err != nil {
				continue
			}
			rtn[i] = int(val)
			i++
		}
		return rtn[:i]
	}
}

func SectFilter(str string) (func(*Elem) bool, string) {
	var filterfunc func(el *Elem) bool
	var hstr string
	var snums []int
	fs := Re_sectnum.FindStringSubmatch(str)
	if fs[1] != "" {
		snums = SplitNums(fmt.Sprintf("range(%s)", fs[2]))
	} else {
		snums = SplitNums(fs[2])
	}
	filterfunc = func(el *Elem) bool {
		for _, snum := range snums {
			if el.Sect.Num == snum {
				return true
			}
		}
		return false
	}
	hstr = fmt.Sprintf("Sect == %v", snums)
	return filterfunc, hstr
}

func OriginalSectFilter(str string) (func(*Elem) bool, string) {
	var filterfunc func(el *Elem) bool
	var hstr string
	fs := Re_orgsectnum.FindStringSubmatch(str)
	if len(fs) < 2 {
		return nil, ""
	}
	snums := SplitNums(fs[1])
	filterfunc = func(el *Elem) bool {
		if el.Etype != WBRACE && el.Etype != SBRACE {
			return false
		}
		for _, snum := range snums {
			if el.OriginalSection().Num == snum {
				return true
			}
		}
		return false
	}
	hstr = fmt.Sprintf("Sect == %v", snums)
	return filterfunc, hstr
}

func EtypeFilter(str string) (func(*Elem) bool, string) {
	var filterfunc func(el *Elem) bool
	var hstr string
	fs := Re_etype.FindStringSubmatch(str)
	l := len(fs)
	if l >= 4 {
		var val int
		switch {
		case Re_column.MatchString(fs[l-1]):
			val = COLUMN
		case Re_girder.MatchString(fs[l-1]):
			val = GIRDER
		case Re_brace.MatchString(fs[l-1]):
			val = BRACE
		case Re_wall.MatchString(fs[l-1]):
			val = WALL
		case Re_slab.MatchString(fs[l-1]):
			val = SLAB
		}
		filterfunc = func(el *Elem) bool {
			return el.Etype == val
		}
		hstr = fmt.Sprintf("Etype == %s", ETYPES[val])
	}
	return filterfunc, hstr
}

func FilterElem(frame *Frame, els []*Elem, str string) ([]*Elem, error) {
	l := len(els)
	if els == nil || l == 0 {
		return nil, errors.New("number of input elems is zero")
	}
	parallel := regexp.MustCompile("(?i)^ *// *([xyz]{1})")
	ortho := regexp.MustCompile("^ *TT *([xyzXYZ]{1})")
	onplane := regexp.MustCompile("(?i)^ *on *([xyz]{2})")
	adjoin := regexp.MustCompile("^ *ad(j(o(in?)?)?)? (.*)")
	currentvalue := regexp.MustCompile("^ *cv *([><=!]+) *([0-9.-]+)")
	var filterfunc func(el *Elem) bool
	var hstr string
	switch {
	case parallel.MatchString(str):
		var axis []float64
		fs := parallel.FindStringSubmatch(str)
		if len(fs) < 2 {
			break
		}
		tmp := strings.ToUpper(fs[1])
		axes := [][]float64{XAXIS, YAXIS, ZAXIS}
		for i, val := range []string{"X", "Y", "Z"} {
			if tmp == val {
				axis = axes[i]
				break
			}
		}
		filterfunc = func(el *Elem) bool {
			return el.IsParallel(axis, 1e-4)
		}
		hstr = fmt.Sprintf("Parallel to %sAXIS", tmp)
	case ortho.MatchString(str):
		var axis []float64
		fs := ortho.FindStringSubmatch(str)
		if len(fs) < 2 {
			break
		}
		tmp := strings.ToUpper(fs[1])
		axes := [][]float64{XAXIS, YAXIS, ZAXIS}
		for i, val := range []string{"X", "Y", "Z"} {
			if tmp == val {
				axis = axes[i]
				break
			}
		}
		filterfunc = func(el *Elem) bool {
			return el.IsOrthogonal(axis, 1e-4)
		}
		hstr = fmt.Sprintf("Orthogonal to %sAXIS", tmp)
	case onplane.MatchString(str):
		var axis int
		fs := onplane.FindStringSubmatch(str)
		if len(fs) < 2 {
			break
		}
		tmp := strings.ToUpper(fs[1])
		for i, val := range []string{"X", "Y", "Z"} {
			if strings.Contains(tmp, val) {
				continue
			}
			axis = i
		}
		filterfunc = func(el *Elem) bool {
			if el.IsLineElem() {
				return el.Direction(false)[axis] == 0.0
			} else {
				n := el.Normal(false)
				if n == nil {
					return false
				}
				for i := 0; i < 3; i++ {
					if i == axis {
						continue
					}
					if n[i] != 0.0 {
						return false
					}
				}
				return true
			}
		}
	case Re_sectnum.MatchString(str):
		filterfunc, hstr = SectFilter(str)
	case Re_etype.MatchString(str):
		filterfunc, hstr = EtypeFilter(str)
	case adjoin.MatchString(str):
		fs := adjoin.FindStringSubmatch(str)
		if len(fs) >= 5 {
			condition := fs[4]
			var fil func(*Elem) bool
			var hst string
			switch {
			case Re_sectnum.MatchString(condition):
				fil, hst = SectFilter(condition)
			case Re_etype.MatchString(condition):
				fil, hst = EtypeFilter(condition)
			}
			if fil == nil {
				break
			}
			filterfunc = func(el *Elem) bool {
				for _, en := range el.Enod {
					for _, sel := range frame.SearchElem(en) {
						if sel.Num == el.Num {
							continue
						}
						if fil(sel) {
							return true
						}
					}
				}
				return false
			}
			hstr = fmt.Sprintf("ADJOIN TO %s", hst)
		}
	case currentvalue.MatchString(str):
		fs := currentvalue.FindStringSubmatch(str)
		var f func(float64, float64) bool
		switch fs[1] {
		case ">=":
			f = func (u, v float64) bool {
				return u >= v
			}
		case ">":
			f = func (u, v float64) bool {
				return u > v
			}
		case "<=":
			f = func (u, v float64) bool {
				return u <= v
			}
		case "<":
			f = func (u, v float64) bool {
				return u < v
			}
		case "=", "==":
			f = func (u, v float64) bool {
				return u == v
			}
		case "!=":
			f = func (u, v float64) bool {
				return u != v
			}
		default:
			return els, errors.New("no filtering")
		}
		val, err := strconv.ParseFloat(fs[2], 64)
		if err != nil {
			return els, err
		}
		filterfunc = func(el *Elem) bool {
			return f(el.CurrentValue(frame.Show, true, false), val)
		}
		hstr = fmt.Sprintf("CURRENT VALUE %s %.3f", fs[1], val)
	case strings.EqualFold(str, "confed"):
		filterfunc = func(el *Elem) bool {
			for _, en := range el.Enod {
				for _, c := range en.Conf {
					if c {
						return true
					}
				}
			}
			return false
		}
		hstr = "CONFED"
	}
	if filterfunc != nil {
		tmpels := make([]*Elem, l)
		enum := 0
		for _, el := range els {
			if el == nil {
				continue
			}
			if filterfunc(el) {
				tmpels[enum] = el
				enum++
			}
		}
		rtn := tmpels[:enum]
		fmt.Printf("FILTER: %s\n", hstr)
		return rtn, nil
	} else {
		return els, errors.New("no filtering")
	}
}

// line: (x1, y1) -> (x2, y2), dot: (dx, dy)
// provided that x1*y2-x2*y1>0
//     if rtn>0: dot is the same side as (0, 0)
//     if rtn==0: dot is on the line
//     if rtn<0: dot is the opposite side to (0, 0)
func DotLine(x1, y1, x2, y2, dx, dy float64) float64 {
	return (x1*y2 + x2*dy + dx*y1) - (x1*dy + x2*y1 + dx*y2)
}
