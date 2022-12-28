package arclm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/yofu/st/matrix"
)

type Frame struct {
	Sects       []*Sect
	Nodes       []*Node
	Elems       []*Elem
	EigenValue  []float64
	EigenVector [][]float64
	Pivot       chan int
	Lapch       chan int
	Endch       chan error
	Output      io.Writer
	running     bool
	cancel      context.CancelFunc
}

func NewFrame() *Frame {
	af := new(Frame)
	af.Sects = make([]*Sect, 0)
	af.Nodes = make([]*Node, 0)
	af.Elems = make([]*Elem, 0)
	af.Pivot = make(chan int)
	af.Lapch = make(chan int)
	af.Endch = make(chan error)
	af.Output = os.Stdout
	return af
}

type FrameState struct {
	Conf     [][]bool
	Disp     [][]float64
	Reaction [][]float64
	Stress   [][]float64
}

func NewFrameState(nnode, nelem int) *FrameState {
	fs := new(FrameState)
	fs.Conf = make([][]bool, nnode)
	fs.Disp = make([][]float64, nnode)
	fs.Reaction = make([][]float64, nnode)
	fs.Stress = make([][]float64, nelem)
	for i := 0; i < nnode; i++ {
		fs.Conf[i] = make([]bool, 6)
		fs.Disp[i] = make([]float64, 6)
		fs.Reaction[i] = make([]float64, 6)
	}
	for i := 0; i < nelem; i++ {
		fs.Stress[i] = make([]float64, 12)
	}
	return fs
}

func (af *Frame) Running() bool {
	return af.running
}

func (af *Frame) Stop() error {
	if af.cancel == nil {
		return fmt.Errorf("not running")
	}
	af.cancel()
	af.running = false
	return nil
}

func (af *Frame) ReadInput(filename string) error {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var lis []string
	if ok := strings.HasSuffix(string(f), "\r\n"); ok {
		lis = strings.Split(string(f), "\r\n")
	} else {
		lis = strings.Split(string(f), "\n")
	}
	split := func(str string) []string {
		var words []string
		for _, k := range strings.Split(str, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		return words
	}
	words := split(lis[0])
	nums := make([]int, 3)
	for i := 0; i < 3; i++ {
		num, err := strconv.ParseInt(words[i], 10, 64)
		if err != nil {
			return err
		}
		nums[i] = int(num)
	}
	af.Nodes = make([]*Node, nums[0])
	af.Elems = make([]*Elem, nums[1])
	af.Sects = make([]*Sect, nums[2])
	// Sect
	for i, j := range lis[1 : 1+nums[2]] {
		words := split(j)
		if len(words) == 0 {
			continue
		}
		s, err := ParseArclmSect(words)
		if err != nil {
			return err
		}
		af.Sects[i] = s
	}
	// Node1
	for i, j := range lis[1+nums[2] : 1+nums[2]+nums[0]] {
		words := split(j)
		if len(words) == 0 {
			continue
		}
		n, err := ParseArclmNode(words)
		if err != nil {
			return err
		}
		n.Index = i
		af.Nodes[i] = n
	}
	// Elem
	for i, j := range lis[1+nums[2]+nums[0] : 1+nums[2]+nums[0]+nums[1]] {
		words := split(j)
		if len(words) == 0 {
			continue
		}
		el, err := ParseArclmElem(words, af.Sects, af.Nodes)
		if err != nil {
			return err
		}
		af.Elems[i] = el
	}
	// Node2
	for i, j := range lis[1+nums[2]+nums[0]+nums[1] : 1+nums[2]+nums[0]+nums[1]+nums[0]] {
		words := split(j)
		if len(words) == 0 {
			continue
		}
		num, err := strconv.ParseInt(words[0], 10, 64)
		if err != nil {
			return err
		}
		nnum := int(num)
		if af.Nodes[i].Num != nnum {
			return errors.New(fmt.Sprintf("ARCLM: NODE %d: format error", nnum))
		}
		af.Nodes[i].Parse(words)
	}
	return nil
}

func (frame *Frame) SaveInput(fn string) error {
	var otp bytes.Buffer
	otp.WriteString(fmt.Sprintf("%5d %5d %5d\n", len(frame.Nodes), len(frame.Elems), len(frame.Sects)))
	// Sect
	for _, s := range frame.Sects {
		otp.WriteString(s.InlString())
	}
	// Node: Coord
	for _, n := range frame.Nodes {
		otp.WriteString(n.InlCoordString())
	}
	// Elem
	for _, el := range frame.Elems {
		otp.WriteString(el.InlString())
	}
	// Node: Boundary Condition
	for _, n := range frame.Nodes {
		otp.WriteString(n.InlConditionString())
	}
	// Write
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	var withcr bytes.Buffer
	withcr.WriteString(strings.Replace(otp.String(), "\n", "\r\n", -1))
	withcr.WriteTo(w)
	return nil
}

func (frame *Frame) Initialise() {
	for _, n := range frame.Nodes {
		for i := 0; i < 6; i++ {
			n.Disp[i] = 0.0
			n.Reaction[i] = 0.0
		}
	}
	for _, el := range frame.Elems {
		for i := 0; i < 12; i++ {
			el.Stress[i] = el.Cmq[i]
		}
		el.Energy = 0.0
		el.Energyb = 0.0
	}
}

func (frame *Frame) SaveState() *FrameState {
	nnode := len(frame.Nodes)
	nelem := len(frame.Elems)
	fs := NewFrameState(nnode, nelem)
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			fs.Conf[i][j] = n.Conf[j]
			fs.Disp[i][j] = n.Disp[j]
			fs.Reaction[i][j] = n.Reaction[j]
		}
	}
	for i, el := range frame.Elems {
		for j := 0; j < 12; j++ {
			fs.Stress[i][j] = el.Stress[j]
		}
	}
	return fs
}

func (frame *Frame) RestoreState(fs *FrameState) {
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			n.Conf[j] = fs.Conf[i][j]
			n.Disp[j] = fs.Disp[i][j]
			n.Reaction[j] = fs.Reaction[i][j]
		}
	}
	for i, el := range frame.Elems {
		for j := 0; j < 12; j++ {
			el.Stress[j] = fs.Stress[i][j]
		}
	}
}

func (frame *Frame) AssemGlobalVector(safety float64) (int, []bool, []float64, error) {
	var err error
	var tmatrix [][]float64
	size := 6 * len(frame.Nodes)
	gvct := make([]float64, size)
	for _, el := range frame.Elems {
		tmatrix, err = el.TransMatrix()
		if err != nil {
			return 0, nil, nil, err
		}
		gvct = el.AssemCMQ(tmatrix, gvct, safety)
	}
	csize, conf, vec := frame.AssemConf(gvct, safety)
	return csize, conf, vec, nil
}

func (frame *Frame) AssemGlobalMatrix(matf func(*Elem) ([][]float64, error), vecf func(*Elem, [][]float64, []float64, float64) []float64, safety float64) (*matrix.COOMatrix, []float64, error) { // TODO: UNDER CONSTRUCTION
	var err error
	var tmatrix, stiff [][]float64
	size := 6 * len(frame.Nodes)
	gmtx := matrix.NewCOOMatrix(size)
	gvct := make([]float64, size)
	for _, el := range frame.Elems {
		tmatrix, err = el.TransMatrix()
		if err != nil {
			return nil, nil, err
		}
		stiff, err = matf(el)
		if err != nil {
			return nil, nil, err
		}
		if stiff == nil {
			continue
		}
		stiff, err = el.ModifyHinge(stiff)
		if err != nil {
			return nil, nil, err
		}
		stiff = Transformation(stiff, tmatrix)
		for n1 := 0; n1 < 2; n1++ {
			for i := 0; i < 6; i++ {
				row := 6*el.Enod[n1].Index + i
				for n2 := 0; n2 < 2; n2++ {
					for j := 0; j < 6; j++ {
						col := 6*el.Enod[n2].Index + j
						val := stiff[6*n1+i][6*n2+j]
						if val != 0.0 {
							gmtx.Add(row, col, val)
						}
					}
				}
			}
		}
		el.ModifyCMQ()
		gvct = vecf(el, tmatrix, gvct, safety)
	}
	return gmtx, gvct, nil
}

func (frame *Frame) AssemMassMatrix() *matrix.COOMatrix {
	size := 6 * len(frame.Nodes)
	mmtx := matrix.NewCOOMatrix(size)
	for _, n := range frame.Nodes {
		for i := 0; i < 6; i++ {
			if n.Conf[i] {
				continue
			}
			row := 6*n.Index + i
			mmtx.Add(row, row, n.Mass)
		}
	}
	return mmtx
}

func (frame *Frame) KE(safety float64) (*matrix.COOMatrix, []float64, error) { // TODO: UNDER CONSTRUCTION
	matf := func(elem *Elem) ([][]float64, error) {
		return elem.StiffMatrix()
	}
	vecf := func(elem *Elem, tmatrix [][]float64, gvct []float64, safety float64) []float64 {
		return elem.AssemCMQ(tmatrix, gvct, safety)
	}
	return frame.AssemGlobalMatrix(matf, vecf, safety)
}

// TODO: implement
func (frame *Frame) KP(safety float64) (*matrix.COOMatrix, []float64, error) { // TODO: UNDER CONSTRUCTION
	matf := func(elem *Elem) ([][]float64, error) {
		estiff, err := elem.StiffMatrix()
		if err != nil {
			return nil, err
		}
		pstiff, err := elem.PlasticMatrix(estiff)
		if err != nil {
			return nil, err
		}
		return pstiff, nil
	}
	vecf := func(elem *Elem, tmatrix [][]float64, gvct []float64, safety float64) []float64 {
		return elem.AssemCMQ(tmatrix, gvct, safety)
	}
	return frame.AssemGlobalMatrix(matf, vecf, safety)
}

func (frame *Frame) KG(safety float64) (*matrix.COOMatrix, []float64, error) { // TODO: UNDER CONSTRUCTION
	matf := func(elem *Elem) ([][]float64, error) {
		return elem.GeoStiffMatrix()
	}
	vecf := func(elem *Elem, tmatrix [][]float64, gvct []float64, safety float64) []float64 {
		gvct = elem.AssemCMQ(tmatrix, gvct, safety)
		gvct = elem.ModifyTrueForce(tmatrix, gvct)
		return gvct
	}
	return frame.AssemGlobalMatrix(matf, vecf, safety)
}

func (frame *Frame) KEKG(safety float64) (*matrix.COOMatrix, []float64, error) { // TODO: UNDER CONSTRUCTION
	matf := func(elem *Elem) ([][]float64, error) {
		if !elem.IsValid {
			return nil, nil
		}
		estiff, err := elem.StiffMatrix()
		if err != nil {
			return nil, err
		}
		gstiff, err := elem.GeoStiffMatrix()
		if err != nil {
			return nil, err
		}
		stiff := make([][]float64, 12)
		for i := 0; i < 12; i++ {
			stiff[i] = make([]float64, 12)
		}
		for i := 0; i < 12; i++ {
			for j := 0; j < 12; j++ {
				stiff[i][j] = estiff[i][j] + gstiff[i][j]
			}
		}
		return stiff, nil
	}
	vecf := func(elem *Elem, tmatrix [][]float64, gvct []float64, safety float64) []float64 {
		if !elem.IsValid {
			return gvct
		}
		gvct = elem.AssemCMQ(tmatrix, gvct, safety)
		gvct = elem.ModifyTrueForce(tmatrix, gvct)
		return gvct
	}
	return frame.AssemGlobalMatrix(matf, vecf, safety)
}

func (frame *Frame) AssemConf(gvct []float64, safety float64) (int, []bool, []float64) {
	size := 6 * len(frame.Nodes)
	csize := 0
	conf := make([]bool, size)
	vec := make([]float64, size)
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			if n.Conf[j] {
				conf[6*i+j] = true
				csize++
			} else {
				vec[6*i+j-csize] = gvct[6*i+j] + safety*n.Force[j]
			}
		}
	}
	return csize, conf, vec[:size-csize]
}

func (frame *Frame) FillConf(vec []float64) []float64 {
	ind := 0
	rtn := make([]float64, 6*len(frame.Nodes))
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			if n.Conf[j] {
				ind++
				continue
			}
			rtn[6*i+j] = vec[6*i+j-ind]
		}
	}
	return rtn
}

func (frame *Frame) RemoveConf(vec []float64) []float64 {
	ind := 0
	rtn := make([]float64, 6*len(frame.Nodes))
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			if n.Conf[j] {
				continue
			}
			rtn[ind] = vec[6*i+j]
			ind++
		}
	}
	return rtn[:ind]
}

func (frame *Frame) UpdateStress(vec []float64) ([][]float64, error) {
	rtn := make([][]float64, len(frame.Elems))
	for enum, el := range frame.Elems {
		if !el.IsValid {
			for i := 0; i < 12; i++ {
				el.Stress[i] = 0.0
			}
			continue
		}
		gdisp := make([]float64, 12)
		for i := 0; i < 2; i++ {
			for j := 0; j < 6; j++ {
				gdisp[6*i+j] = vec[6*el.Enod[i].Index+j]
			}
		}
		df, err := el.ElemStress(gdisp)
		if err != nil {
			return nil, err
		}
		rtn[enum] = df
	}
	return rtn, nil
}

func (frame *Frame) UpdateStressEnergy(vec []float64) ([][]float64, []float64, error) {
	rtn := make([][]float64, len(frame.Elems))
	eng := make([]float64, len(frame.Elems))
	for enum, el := range frame.Elems {
		gdisp := make([]float64, 12)
		for i := 0; i < 2; i++ {
			for j := 0; j < 6; j++ {
				gdisp[6*i+j] = vec[6*el.Enod[i].Index+j]
			}
		}
		df, err := el.ElemStress(gdisp)
		if err != nil {
			return nil, nil, err
		}
		rtn[enum] = df
		e, err := el.BucklingEnergy(gdisp)
		if err != nil {
			return nil, nil, err
		}
		eng[enum] = e
	}
	return rtn, eng, nil
}

// TODO: implement
func (frame *Frame) UpdateStressPlastic(vec []float64) ([][]float64, error) {
	rtn := make([][]float64, len(frame.Elems))
	for enum, el := range frame.Elems {
		gdisp := make([]float64, 12)
		for i := 0; i < 2; i++ {
			for j := 0; j < 6; j++ {
				gdisp[6*i+j] = vec[6*el.Enod[i].Index+j]
			}
		}
		df, err := el.ElemStress(gdisp)
		if err != nil {
			return nil, err
		}
		rtn[enum] = df
	}
	return rtn, nil
}

func (frame *Frame) UpdateReaction(gmtx *matrix.COOMatrix, vec []float64) []float64 {
	rtn := make([]float64, 6*len(frame.Nodes))
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			if n.Conf[j] {
				val := 0.0
				for k := 0; k < gmtx.Size; k++ {
					stiff := gmtx.Query(6*i+j, k)
					val += stiff * vec[k]
				}
				n.Reaction[j] += val
				rtn[6*i+j] = val
			}
		}
	}
	return rtn
}

func (frame *Frame) UpdateForm(vec []float64) {
	for i, n := range frame.Nodes {
		for j := 0; j < 6; j++ {
			n.Disp[j] += vec[6*i+j]
		}
	}
}

func (frame *Frame) WriteTo(w io.Writer) (int64, error) {
	var otp bytes.Buffer
	var rea bytes.Buffer
	otp.WriteString("\n\n** FORCES OF MEMBER\n\n")
	otp.WriteString("  NO   KT NODE         N        Q1        Q2        MT        M1        M2\n\n")
	for _, el := range frame.Elems {
		otp.WriteString(el.OutputStress())
	}
	otp.WriteString("\n\n** DISPLACEMENT OF NODE\n\n")
	otp.WriteString("  NO          U          V          W         KSI         ETA       OMEGA\n\n")
	rea.WriteString("\n\n** REACTION\n\n")
	rea.WriteString("  NO  DIRECTION              R    NC\n\n")
	for _, n := range frame.Nodes {
		otp.WriteString(fmt.Sprintf("%4d", n.Num))
		for j := 0; j < 3; j++ {
			otp.WriteString(fmt.Sprintf(" %10.6f", n.Disp[j]))
			if n.Conf[j] {
				rea.WriteString(fmt.Sprintf("%4d %10d %14.6f     1\n", n.Num, j+1, n.Reaction[j]))
			}
		}
		for j := 3; j < 6; j++ {
			otp.WriteString(fmt.Sprintf(" %11.7f", n.Disp[j]))
			if n.Conf[j] {
				rea.WriteString(fmt.Sprintf("%4d %10d %14.6f     1\n", n.Num, j+1, n.Reaction[j]))
			}
		}
		otp.WriteString("\n")
	}
	var rtn, tmp int64
	var err error
	rtn, err = otp.WriteTo(w)
	if err != nil {
		return rtn, err
	}
	tmp, err = rea.WriteTo(w)
	return rtn + tmp, err
}

func (frame *Frame) WriteBclngTo(w io.Writer) (int64, error) {
	var otp bytes.Buffer
	otp.WriteString(fmt.Sprintf("NODES=%d ELEMS=%d SECTS=%d\n", len(frame.Nodes), len(frame.Elems), len(frame.Sects)))
	for i := 0; i < len(frame.EigenValue); i++ {
		otp.WriteString(fmt.Sprintf("EIGEN VALUE %d=%12.5E\n", i+1, frame.EigenValue[i]))
		for j, n := range frame.Nodes {
			otp.WriteString(fmt.Sprintf("NODE: %4d {dU}=", n.Num))
			for k := 0; k < 6; k++ {
				otp.WriteString(fmt.Sprintf(" %12.5E", frame.EigenVector[i][6*j+k]))
			}
			otp.WriteString("\n")
		}
	}
	otp.WriteString("COMPLETED.\n")
	return otp.WriteTo(w)
}

type AnalysisCondition struct {
	init   bool
	solver string
	otp    []string
	extra  [][]float64

	nlgeometry bool
	nlmaterial bool

	postprocess func(*Frame, [][]float64, []float64, []float64) (float64, bool)

	nlap  int
	delta float64
	start float64
	max   float64
	eps   float64
}

func NewAnalysisCondition() *AnalysisCondition {
	return &AnalysisCondition{
		init:        true,
		solver:      "LLS",
		extra:       nil,
		nlgeometry:  false,
		nlmaterial:  false,
		postprocess: nil,
		nlap:        1,
		delta:       1.0,
		start:       0.0,
		max:         1.0,
		eps:         1e-12,
	}
}

func (cond *AnalysisCondition) Nlap() int {
	return cond.nlap
}
func (cond *AnalysisCondition) Delta() float64 {
	return cond.delta
}
func (cond *AnalysisCondition) SetInit(i bool) {
	cond.init = i
}
func (cond *AnalysisCondition) SetSolver(s string) {
	cond.solver = s
}
func (cond *AnalysisCondition) SetOutput(o []string) {
	cond.otp = o
}
func (cond *AnalysisCondition) Output() []string {
	return cond.otp
}
func (cond *AnalysisCondition) SetExtra(e [][]float64) {
	cond.extra = e
}
func (cond *AnalysisCondition) SetNlgeometry(n bool) {
	cond.nlgeometry = n
}
func (cond *AnalysisCondition) SetNlmaterial(n bool) {
	cond.nlmaterial = n
}
func (cond *AnalysisCondition) SetNlap(l int) {
	cond.nlap = l
}
func (cond *AnalysisCondition) SetDelta(d float64) {
	cond.delta = d
}
func (cond *AnalysisCondition) SetStart(s float64) {
	cond.start = s
}
func (cond *AnalysisCondition) SetMax(m float64) {
	cond.max = m
}
func (cond *AnalysisCondition) SetEps(e float64) {
	cond.eps = e
}
func (cond *AnalysisCondition) SetPostprocess(f func(*Frame, [][]float64, []float64, []float64) (float64, bool)) {
	cond.postprocess = f
}

func (cond *AnalysisCondition) String() string {
	var rtn bytes.Buffer
	rtn.WriteString(fmt.Sprintf("INITIALIZE  : %t\n", cond.init))
	rtn.WriteString(fmt.Sprintf("OUTPUT FILE : %v\n", cond.otp))
	rtn.WriteString(fmt.Sprintf("SOLVER      : %s\n", cond.solver))
	rtn.WriteString(fmt.Sprintf("EPS         : %.3E\n", cond.eps))
	rtn.WriteString(fmt.Sprintf("EXTRA LOAD  : %d\n", len(cond.extra)))
	rtn.WriteString("NON-LINEAR\n")
	rtn.WriteString(fmt.Sprintf("  GEOMETRY  : %t\n", cond.nlgeometry))
	rtn.WriteString(fmt.Sprintf("  MATERIAL  : %t\n", cond.nlmaterial))
	rtn.WriteString("STEP\n")
	rtn.WriteString(fmt.Sprintf("  NLAP      : %d\n", cond.nlap))
	rtn.WriteString(fmt.Sprintf("  DELTA     : %.3f\n", cond.delta))
	rtn.WriteString(fmt.Sprintf("  START     : %.3f\n", cond.start))
	rtn.WriteString(fmt.Sprintf("  MAX       : %.3f\n", cond.max))
	rtn.WriteString(fmt.Sprintf("POST PROCESS: %t", cond.postprocess != nil))
	return rtn.String()
}

func (cond *AnalysisCondition) NonLinear() bool {
	return cond.nlgeometry || cond.nlmaterial
}

func (frame *Frame) StaticAnalysis(cancel context.CancelFunc, cond *AnalysisCondition) error {
	frame.running = true
	frame.cancel = cancel
	defer func() {
		frame.running = false
		cancel()
	}()
	if cond.init {
		frame.Initialise()
	}
	start := time.Now()
	laptime := func(message string) {
		end := time.Now()
		fmt.Fprintf(frame.Output, "%s: %fsec\n", message, (end.Sub(start)).Seconds())
	}
	var solver Solver
	switch cond.solver {
	default:
		solver = LLS(frame, laptime)
	case "CRS":
		solver = CRS_CG(cond.eps, laptime)
	case "LLS":
		solver = LLS(frame, laptime)
	case "CG":
		solver = LLS_CG(cond.eps, laptime)
	case "PCG":
		solver = LLS_PCG(cond.eps, laptime)
	}
	var err error
	var gmtx *matrix.COOMatrix
	var gvct, vec []float64
	var csize int
	var conf []bool
	var answers [][]float64
	var bnorm, rnorm, sign float64
	output := func(fns []string, ind int, l, nl int) error {
		var fn string
		if fns == nil || len(fns) <= ind {
			fn = fmt.Sprintf("hogtxt%d.otp", ind)
		} else {
			fn = fns[ind]
		}
		if l < nl {
			ext := filepath.Ext(fn)
			fn = fmt.Sprintf("%s_LAP_%d_%d%s", strings.Replace(fn, ext, "", -1), l, nl, ext)
		}
		w, err := os.Create(fn)
		if err != nil {
			return err
		}
		defer w.Close()
		frame.WriteTo(w)
		return nil
	}
	lap := 0
	total := cond.start + cond.delta
	for {
		if total > cond.max {
			total = cond.max
		}
		f0 := frame.SaveState()
		if !cond.nlgeometry || (cond.init && lap == 0) {
			gmtx, gvct, err = frame.KE(total)
			if err != nil {
				return err
			}
			csize, conf, vec = frame.AssemConf(gvct, total)
			if cond.nlgeometry { // subtract CMQ
				for _, el := range frame.Elems {
					for i := 0; i < 12; i++ {
						el.Stress[i] = 0.0
					}
				}
			}
		} else {
			gmtx, gvct, err = frame.KEKG(total)
			if err != nil {
				return err
			}
			csize, conf, vec = frame.AssemConf(gvct, total)
		}
		if lap == 0 {
			bnorm = math.Sqrt(Dot(vec, vec, len(vec)))
		} else {
			rnorm = math.Sqrt(Dot(vec, vec, len(vec)))
		}
		laptime("ASSEM")
		if cond.extra != nil && len(cond.extra) > 1 {
			vecs := make([][]float64, len(cond.extra)+1)
			vecs[0] = vec
			for i := 0; i < len(cond.extra); i++ {
				vecs[i+1] = cond.extra[i]
			}
			answers, err = solver.Solve(gmtx, csize, conf, vecs...)
		} else {
			answers, err = solver.Solve(gmtx, csize, conf, vec)
		}
		if err != nil {
			return err
		}
		sign = 0.0
		for i := 0; i < len(vec); i++ {
			sign += answers[0][i] * vec[i]
		}
		if lap == 0 {
			laptime(fmt.Sprintf("sylvester's law of inertia: LAP %d %.3f", lap, sign))
		} else {
			laptime(fmt.Sprintf("sylvester's law of inertia: LAP %d %.3f", lap, sign))
			if sign < 0.0 {
				output(cond.otp, 0, lap+1, cond.nlap)
				return errors.New(fmt.Sprintf("sylvester's law of inertia: %.3f", sign))
			}
		}
		if cond.extra != nil {
			if cond.otp == nil || len(cond.otp) < 1 {
				cond.otp = make([]string, len(cond.extra)+1)
				for i := 0; i < len(cond.extra)+1; i++ {
					cond.otp[i] = fmt.Sprintf("hogtxt_%02d.otp", i)
				}
			} else if len(cond.otp) < len(cond.extra)+1 {
				ext := filepath.Ext(cond.otp[0])
				for i := 0; i < len(cond.extra)+1-len(cond.otp); i++ {
					cond.otp = append(cond.otp, fmt.Sprintf("%s_%02d.%s", strings.TrimSuffix(cond.otp[0], ext), i, ext))
				}
			}
		}
		if lap >= cond.nlap-1 && total >= cond.max {
			for nans, ans := range answers {
				f := frame.SaveState()
				if nans >= 1 { // subtract CMQ for extra load
					for _, el := range frame.Elems {
						for i := 0; i < 12; i++ {
							el.Stress[i] -= el.Cmq[i]
						}
					}
				}
				vec := frame.FillConf(ans)
				_, err := frame.UpdateStress(vec)
				if err != nil {
					return err
				}
				frame.UpdateReaction(gmtx, vec)
				frame.UpdateForm(vec)
				laptime(fmt.Sprintf("%04d / %04d: TOTAL = %.3f NORM = %.5E", lap+1, cond.nlap, total, rnorm/bnorm))
				output(cond.otp, nans, lap+1, cond.nlap)
				frame.Lapch <- lap + 1
				<-frame.Lapch
				frame.RestoreState(f)
			}
			break
		} else {
			du := frame.FillConf(answers[0])
			df, err := frame.UpdateStress(du)
			if err != nil {
				return err
			}
			dr := frame.UpdateReaction(gmtx, du)
			frame.UpdateForm(du)
			delta := cond.delta
			next := true
			if cond.postprocess != nil {
				delta, next = cond.postprocess(frame, df, du, dr)
			}
			frame.Lapch <- lap + 1
			ret := <-frame.Lapch
			if ret != 0 {
				output(cond.otp, 0, lap+1, cond.nlap)
				return fmt.Errorf("analysis cancelled")
			}
			if !next {
				old := total
				if delta != cond.delta {
					total += delta - cond.delta
					cond.delta = delta
				}
				laptime(fmt.Sprintf("%04d / %04d: TOTAL = %.3f -> %.3f NORM = %.5E U", lap+1, cond.nlap, old, total, rnorm/bnorm))
				frame.RestoreState(f0)
			} else {
				laptime(fmt.Sprintf("%04d / %04d: TOTAL = %.3f NORM = %.5E", lap+1, cond.nlap, total, rnorm/bnorm))
				lap++
				total += cond.delta
			}
		}
	}
	laptime("End")
	return nil
}

// TODO: implement && merge to StaticAnalysis
func (frame *Frame) Arclm101(otp string, init bool, nlap int, dsafety float64) error { // TODO: speed up
	if init {
		frame.Initialise()
	}
	start := time.Now()
	laptime := func(message string) {
		end := time.Now()
		fmt.Fprintf(frame.Output, "%s: %fsec\n", message, (end.Sub(start)).Seconds())
	}
	solver := LLS(frame, laptime)
	var err error
	var answers [][]float64
	var gmtx *matrix.COOMatrix
	var gvct, vec []float64
	var csize int
	var conf []bool
	safety := 0.0
	for lap := 0; lap < nlap; lap++ {
		safety += dsafety
		gmtx, gvct, err = frame.KP(safety)
		csize, conf, vec = frame.AssemConf(gvct, safety)
		if err != nil {
			return err
		}
		laptime("Assem")
		answers, err = solver.Solve(gmtx, csize, conf, vec)
		if err != nil {
			return err
		}
		tmp := frame.FillConf(answers[0])
		_, err = frame.UpdateStressPlastic(tmp)
		if err != nil {
			return err
		}
		frame.UpdateReaction(gmtx, tmp)
		frame.UpdateForm(tmp)
		laptime(fmt.Sprintf("%04d / %04d: SAFETY = %.3f", lap+1, nlap, safety))
		frame.Lapch <- lap + 1
		<-frame.Lapch
	}
	if otp == "" {
		otp = "hogtxt.otp"
	}
	w, err := os.Create(otp)
	if err != nil {
		return err
	}
	defer w.Close()
	frame.WriteTo(w)
	return nil
}

// ANALYSIS FOR PILES UNDER LATERAL LOAD
// E: 70 * α * ξ [tf/m2]
// A: 3.16 * B^(-0.75) * N * B/100 * L [m2]
//    α: SAND=80 CLAY=60
//    ξ: PILE GROUPE COEFFICIENT 1.0
//    B : PILE DIAMETER[cm]
//    N : N-VALUE
//    L : SOIL SPRING PITFCH[m]
func (frame *Frame) Arclm301(otp string, init bool, sects []int, eps float64) error { // TODO: speed up
	if init {
		frame.Initialise()
	}
	start := time.Now()
	laptime := func(message string) {
		end := time.Now()
		fmt.Fprintf(frame.Output, "%s: %fsec\n", message, (end.Sub(start)).Seconds())
	}
	solver := LLS(frame, laptime)
	var err error
	var answers [][]float64
	var gmtx *matrix.COOMatrix
	var gvct, vec []float64
	var norm float64
	var csize int
	var conf []bool
	size := 6 * len(frame.Nodes)
	dlast := make([]float64, size)
	dd := make([]float64, size)
	kpile := func(safety float64, gdisp []float64) (*matrix.COOMatrix, []float64, error) {
		matf := func(elem *Elem) ([][]float64, error) {
			for _, sec := range sects {
				if elem.Sect.Num == sec {
					ld := 0.0
					for i := 0; i < 3; i++ {
						ld += math.Pow((elem.Enod[1].Coord[i] + gdisp[6*elem.Enod[1].Index+i] - elem.Enod[0].Coord[i] - gdisp[6*elem.Enod[0].Index+i]), 2)
					}
					y := math.Abs(math.Sqrt(ld)-elem.Length()) * 100
					if y <= 0.1 {
						return elem.StiffMatrix()
					} else {
						a := elem.Sect.Value[0]
						elem.Sect.Value[0] = a / math.Sqrt(y) / 3.16
						stiff, err := elem.StiffMatrix()
						elem.Sect.Value[0] = a
						return stiff, err
					}
				}
			}
			return elem.StiffMatrix()
		}
		vecf := func(elem *Elem, tmatrix [][]float64, gvct []float64, safety float64) []float64 {
			return elem.AssemCMQ(tmatrix, gvct, safety)
		}
		return frame.AssemGlobalMatrix(matf, vecf, safety)
	}
	kpilestress := func(f *Frame, vec []float64, sects []int) ([][]float64, error) {
		rtn := make([][]float64, len(f.Elems))
		for enum, el := range f.Elems {
			gdisp := make([]float64, 12)
			for i := 0; i < 2; i++ {
				for j := 0; j < 6; j++ {
					gdisp[6*i+j] = vec[6*el.Enod[i].Index+j]
				}
			}
			var df []float64
			var err error
			soil := false
			for _, sec := range sects {
				if el.Sect.Num == sec {
					ld := 0.0
					for i := 0; i < 3; i++ {
						ld += math.Pow((el.Enod[1].Coord[i] + gdisp[6+i] - el.Enod[0].Coord[i] - gdisp[i]), 2)
					}
					y := math.Abs(math.Sqrt(ld)-el.Length()) * 100
					if y <= 0.1 {
						df, err = el.ElemStress(gdisp)
					} else {
						a := el.Sect.Value[0]
						el.Sect.Value[0] = a / math.Sqrt(y) / 3.16
						df, err = el.ElemStress(gdisp)
						el.Sect.Value[0] = a
					}
					soil = true
				}
			}
			if !soil {
				df, err = el.ElemStress(gdisp)
			}
			if err != nil {
				return nil, err
			}
			rtn[enum] = df
		}
		return rtn, nil
	}
	lap := 0
	for {
		f := frame.SaveState()
		if lap == 0 {
			gmtx, gvct, err = frame.KE(1.0)
			csize, conf, vec = frame.AssemConf(gvct, 1.0)
		} else {
			gmtx, gvct, err = kpile(1.0, dlast)
			csize, conf, vec = frame.AssemConf(gvct, 1.0)
		}
		if err != nil {
			return err
		}
		laptime("Assem")
		answers, err = solver.Solve(gmtx, csize, conf, vec)
		if err != nil {
			return err
		}
		tmp := frame.FillConf(answers[0])
		_, err = kpilestress(frame, tmp, sects)
		if err != nil {
			return err
		}
		frame.UpdateReaction(gmtx, tmp)
		frame.UpdateForm(tmp)
		for i := 0; i < size; i++ {
			dd[i] = tmp[i] - dlast[i]
			dlast[i] = tmp[i]
		}
		norm = math.Sqrt(Dot(dd, dd, len(dd)))
		laptime(fmt.Sprintf("LAP = %d NORM = %.5E", lap+1, norm))
		if norm < eps {
			frame.Lapch <- lap + 1
			break
		}
		frame.Lapch <- lap + 1
		<-frame.Lapch
		lap++
		frame.RestoreState(f)
	}
	if otp == "" {
		otp = "hogtxt.otp"
	}
	w, err := os.Create(otp)
	if err != nil {
		return err
	}
	defer w.Close()
	frame.WriteTo(w)
	return nil
}

// TODO: implement
func (frame *Frame) Arclm401(otp string, init bool, eps float64, wgtdict map[int]float64) error {
	if init {
		frame.Initialise()
	}
	s := frame.SaveState()
	start := time.Now()
	laptime := func(message string) {
		end := time.Now()
		fmt.Fprintf(frame.Output, "%s: %fsec\n", message, (end.Sub(start)).Seconds())
	}
	solver := LLS(frame, laptime)
	confed := make([]*Node, len(frame.Nodes))
	confdata := make([][]bool, len(frame.Nodes))
	nnum := 0
	for _, n := range frame.Nodes {
		if n.Conf[2] {
			confed[nnum] = n
			confdata[nnum] = make([]bool, 6)
			for i := 0; i < 6; i++ {
				confdata[nnum][i] = n.Conf[i]
			}
			nnum++
			continue
		}
	}
	confed = confed[:nnum]
	confdata = confdata[:nnum]
	released := make([]bool, nnum)
	lap := 0
	for {
		frame.Initialise()
		gmtx, gvct, err := frame.KE(1.0)
		laptime("ASSEM")
		if err != nil {
			return err
		}
		csize, conf, vec := frame.AssemConf(gvct, 1.0)
		laptime("VEC")
		var answers [][]float64
		answers, err = solver.Solve(gmtx, csize, conf, vec)
		if err != nil {
			return err
		}
		ans := frame.FillConf(answers[0])
		_, err = frame.UpdateStress(ans)
		if err != nil {
			return err
		}
		frame.UpdateReaction(gmtx, ans)
		frame.UpdateForm(ans)
		uplift := 0
		for i, n := range confed {
			if n.Num == 711 {
				fmt.Println("NODE 711: ", n.Conf[2], n.Disp[2])
			}
			if n.Conf[2] {
				wgt := 0.0
				if v, ok := wgtdict[n.Num]; ok {
					wgt = v
				}
				if n.Num == 711 {
					fmt.Println(n.Reaction[2], wgt)
				}
				if n.Reaction[2]+wgt < 0.0 {
					uplift++
					released[i] = true
					for j := 0; j < 6; j++ {
						n.Conf[j] = false
					}
				}
			} else {
				if n.Disp[2] < 0.0 {
					uplift++
					released[i] = false
					for j := 0; j < 6; j++ {
						n.Conf[j] = confdata[i][j]
					}
				}
			}
		}
		if uplift == 0 {
			break
		}
		lap++
		frame.RestoreState(s)
		frame.Lapch <- lap
		<-frame.Lapch
	}
	var nodes bytes.Buffer
	nodes.WriteString(":node")
	for i := 0; i < nnum; i++ {
		if released[i] {
			nodes.WriteString(fmt.Sprintf(" %d", confed[i].Num))
		}
	}
	fmt.Println(nodes.String())
	if otp == "" {
		otp = "hogtxt.otp"
	}
	w, err := os.Create(otp)
	if err != nil {
		return err
	}
	defer w.Close()
	frame.WriteTo(w)
	laptime("End")
	return nil
}

// TODO: not accurate for higher order
func (frame *Frame) Bclng001(otp string, init bool, n int, eps float64, right float64) error { // TODO: speed up
	if init {
		frame.Initialise()
	}
	start := time.Now()
	laptime := func(message string) {
		end := time.Now()
		fmt.Fprintf(frame.Output, "%s: %fsec\n", message, (end.Sub(start)).Seconds())
	}
	solver := LLS(frame, func(string) {})
	var err error
	var answers [][]float64
	var kemtx, kgmtx *matrix.COOMatrix
	var gvct, vec []float64
	var csize int
	var conf []bool
	kemtx, gvct, err = frame.KE(1.0)
	if err != nil {
		return err
	}
	csize, conf, vec = frame.AssemConf(gvct, 1.0)
	for _, el := range frame.Elems {
		for i := 0; i < 12; i++ {
			el.Stress[i] = 0.0
		}
	}
	answers, err = solver.Solve(kemtx, csize, conf, vec)
	if err != nil {
		return err
	}
	tmp := frame.FillConf(answers[0])
	_, err = frame.UpdateStress(tmp)
	if err != nil {
		return err
	}
	kgmtx, _, err = frame.KG(1.0)
	if err != nil {
		return err
	}
	laptime("initial analysis solved")
	frame.EigenValue = make([]float64, n)
	frame.EigenVector = make([][]float64, n)
	EL := 0.0
	ER := right
	FB := func(C *matrix.LLSMatrix, vec []float64, size int) []float64 {
		tmp := make([]float64, size)
		for j := 0; j < size; j++ {
			tmp[j] = vec[j]
		}
		tmp = C.FELower(tmp)
		for j := 0; j < size; j++ {
			tmp[j] /= C.Query(j, j)
		}
		return C.BSUpper(tmp)
	}
	var answer []float64
	for i := 0; i < n; i++ {
		lap := 0
		lambda := 0.5 * (EL + ER)
		lastlambda := lambda
		var lastvec []float64
	bclng:
		for {
			neg := 0
			gmtx := kgmtx.AddMat(kemtx, lambda)
			mtx := gmtx.ToLLS(csize, conf)
			size := mtx.Size
			C, err := mtx.LDLT(frame.Pivot)
			if err != nil {
				return err
			}
			C.DiagUp()
			// positive definitness can be checked only by checking
			// diagonal elements of D, where D is a diagonal matrix
			// obtained by modified Cholesky factorization (LDLT),
			// according to Sylvester's law of inertia.
			for j := 0; j < len(vec); j++ {
				if C.Query(j, j) < 0.0 {
					neg++
				}
				if neg > i {
					if ER-EL < eps {
						break bclng
					}
					fmt.Fprintf(frame.Output, "LAMBDA<%.14f\n", 1.0/lambda)
					EL = lambda
					lambda = 0.5 * (EL + ER)
					continue bclng
				}
			}
			fmt.Fprintf(frame.Output, "LAMBDA>%.14f\n", 1.0/lambda)
			ans := make([]float64, len(vec))
			for j := 0; j < len(vec); j++ {
				ans[j] = rand.Float64()
			}
			answer = FB(C, ans, size)
			tmp := frame.FillConf(answer)
			tmp = Normalize(tmp)
			lastvec = tmp
			_, _, err = frame.UpdateStressEnergy(tmp)
			if err != nil {
				return err
			}
			lastlambda = lambda
			ER = lambda
			lambda = 0.5 * (EL + ER)
			lap++
			frame.Lapch <- lap + 1
			<-frame.Lapch
		}
		laptime(fmt.Sprintf("\nEIG %d: %.14f", i+1, 1.0/lastlambda))
		frame.EigenValue[i] = 1.0 / lastlambda
		frame.EigenVector[i] = lastvec
		ER = EL
		EL = 0.0
		frame.UpdateReaction(kemtx, lastvec)
		frame.UpdateForm(lastvec)
		frame.Lapch <- i + 1
		<-frame.Lapch
	}
	if otp == "" {
		otp = "hogtxt.otp"
	}
	w, err := os.Create(otp)
	if err != nil {
		return err
	}
	defer w.Close()
	frame.WriteBclngTo(w)
	return nil
}

// TODO: not accurate for higher order
func (frame *Frame) VibrationalEigenAnalysis(otp string, init bool, n int, eps float64, right float64) error { // TODO: speed up
	if init {
		frame.Initialise()
	}
	start := time.Now()
	laptime := func(message string) {
		end := time.Now()
		fmt.Fprintf(frame.Output, "%s: %fsec\n", message, (end.Sub(start)).Seconds())
	}
	var err error
	var kemtx, mmtx *matrix.COOMatrix
	var gvct, vec []float64
	var csize int
	var conf []bool
	kemtx, gvct, err = frame.KE(1.0)
	if err != nil {
		return err
	}
	csize, conf, vec = frame.AssemConf(gvct, 1.0)
	mmtx = frame.AssemMassMatrix()
	frame.EigenValue = make([]float64, n)
	frame.EigenVector = make([][]float64, n)
	EL := 0.0
	ER := right
	FB := func(C *matrix.LLSMatrix, vec []float64, size int) []float64 {
		tmp := make([]float64, size)
		for j := 0; j < size; j++ {
			tmp[j] = vec[j]
		}
		tmp = C.FELower(tmp)
		for j := 0; j < size; j++ {
			tmp[j] /= C.Query(j, j)
		}
		return C.BSUpper(tmp)
	}
	var answer []float64
	for i := 0; i < n; i++ {
		lap := 0
		lambda := 0.5 * (EL + ER)
		lastlambda := lambda
		var lastvec []float64
	bclng:
		for {
			neg := 0
			gmtx := kemtx.AddMat(mmtx, -1.0/lambda)
			mtx := gmtx.ToLLS(csize, conf)
			size := mtx.Size
			C, err := mtx.LDLT(frame.Pivot)
			if err != nil {
				return err
			}
			C.DiagUp()
			// positive definitness can be checked only by checking
			// diagonal elements of D, where D is a diagonal matrix
			// obtained by modified Cholesky factorization (LDLT),
			// according to Sylvester's law of inertia.
			for j := 0; j < len(vec); j++ {
				if C.Query(j, j) < 0.0 {
					neg++
				}
				if neg > i {
					if ER-EL < eps {
						break bclng
					}
					fmt.Fprintf(frame.Output, "LAMBDA<%.14f\n", 1.0/lambda)
					EL = lambda
					lambda = 0.5 * (EL + ER)
					continue bclng
				}
			}
			fmt.Fprintf(frame.Output, "LAMBDA>%.14f\n", 1.0/lambda)
			ans := make([]float64, len(vec))
			for j := 0; j < len(vec); j++ {
				ans[j] = rand.Float64()
			}
			answer = FB(C, ans, size)
			tmp := frame.FillConf(answer)
			tmp = Normalize(tmp)
			lastvec = tmp
			_, _, err = frame.UpdateStressEnergy(tmp)
			if err != nil {
				return err
			}
			lastlambda = lambda
			ER = lambda
			lambda = 0.5 * (EL + ER)
			lap++
			frame.Lapch <- lap + 1
			<-frame.Lapch
		}
		laptime(fmt.Sprintf("\nEIG %d: %.14f", i+1, 1.0/lastlambda))
		frame.EigenValue[i] = 1.0 / lastlambda
		frame.EigenVector[i] = lastvec
		ER = EL
		EL = 0.0
		frame.UpdateReaction(kemtx, lastvec)
		frame.UpdateForm(lastvec)
		frame.Lapch <- i + 1
		<-frame.Lapch
	}
	if otp == "" {
		otp = "hogtxt.otp"
	}
	w, err := os.Create(otp)
	if err != nil {
		return err
	}
	defer w.Close()
	frame.WriteBclngTo(w)
	return nil
}
