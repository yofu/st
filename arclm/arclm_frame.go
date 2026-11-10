package arclm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yofu/st/matrix"
)

type Frame struct {
	Sects            []*Sect
	Nodes            []*Node
	Elems            []*Elem
	EigenValue       []float64
	EigenVector      [][]float64
	Pivot            chan int
	Lapch            chan int
	Endch            chan error
	Output           io.Writer
	running          bool
	cancel           context.CancelFunc
	runMu            sync.Mutex
	plasticLoad      float64
	plasticLoadValid bool
	loading          *plasticLoadState
	transient        *TimeHistoryState
	geometry         *geometryState
	staticResultCase int
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
	Conf             [][]bool
	Disp             [][]float64
	Reaction         [][]float64
	Stress           [][]float64
	Energy           [][2]float64
	plastic          []plasticElementState
	valid            []bool
	plasticLoad      float64
	plasticLoadValid bool
	loading          *plasticLoadState
	transient        *TimeHistoryState
	geometry         *geometryState
}

func NewFrameState(nnode, nelem int) *FrameState {
	fs := new(FrameState)
	fs.Conf = make([][]bool, nnode)
	fs.Disp = make([][]float64, nnode)
	fs.Reaction = make([][]float64, nnode)
	fs.Stress = make([][]float64, nelem)
	fs.Energy = make([][2]float64, nelem)
	fs.plastic = make([]plasticElementState, nelem)
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
	af.runMu.Lock()
	defer af.runMu.Unlock()
	return af.running
}

func (af *Frame) Stop() error {
	af.runMu.Lock()
	cancel := af.cancel
	running := af.running
	af.runMu.Unlock()
	if !running || cancel == nil {
		return fmt.Errorf("not running")
	}
	cancel()
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
	frame.plasticLoad, frame.plasticLoadValid = 0, false
	frame.loading = nil
	frame.transient = nil
	frame.geometry = nil
	for _, el := range frame.Elems {
		el.geometryAxes = nil
	}
	for _, n := range frame.Nodes {
		for i := 0; i < 6; i++ {
			n.Disp[i] = 0.0
			n.Reaction[i] = 0.0
		}
	}
	for _, el := range frame.Elems {
		el.resetPlasticState()
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
	fs.valid = make([]bool, nelem)
	fs.plasticLoad, fs.plasticLoadValid = frame.plasticLoad, frame.plasticLoadValid
	fs.loading = copyPlasticLoading(frame.loading)
	fs.transient = copyTimeHistoryState(frame.transient)
	fs.geometry = copyGeometryState(frame.geometry)
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
		fs.plastic[i] = el.savePlasticState()
		fs.valid[i] = el.IsValid
		fs.Energy[i] = [2]float64{el.Energy, el.Energyb}
	}
	return fs
}

func (frame *Frame) RestoreState(fs *FrameState) {
	frame.plasticLoad, frame.plasticLoadValid = fs.plasticLoad, fs.plasticLoadValid
	frame.loading = copyPlasticLoading(fs.loading)
	frame.transient = copyTimeHistoryState(fs.transient)
	frame.geometry = copyGeometryState(fs.geometry)
	frame.restoreGeometryAxes()
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
		if i < len(fs.valid) {
			el.IsValid = fs.valid[i]
		}
		if i < len(fs.Energy) {
			el.Energy, el.Energyb = fs.Energy[i][0], fs.Energy[i][1]
		}
		// Preserve compatibility with states containing only nodal data/stress.
		if i < len(fs.plastic) && fs.plastic[i].bonds != nil {
			el.restorePlasticState(fs.plastic[i])
		}
	}
}

func (frame *Frame) CheckSingularNode(e error) error {
	var singular *matrix.SingularPivotError
	if !errors.As(e, &singular) {
		return e
	}
	dof := singular.Index
	names := [...]string{"UX", "UY", "UZ", "RX", "RY", "RZ"}
	for _, n := range frame.Nodes {
		for i, fixed := range n.Conf {
			if fixed {
				continue
			}
			if dof == 0 {
				return fmt.Errorf("matrix singular at NODE %d[%d] (%s): %w", n.Num, i, names[i], e)
			}
			dof--
		}
	}
	return e
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
	return frame.assemGlobalMatrix(func(el *Elem) ([][]float64, error) {
		stiff, err := matf(el)
		if err != nil || stiff == nil {
			return stiff, err
		}
		return el.ModifyHinge(stiff)
	}, vecf, safety, false)
}

func (frame *Frame) assemGlobalMatrix(matf func(*Elem) ([][]float64, error), vecf func(*Elem, [][]float64, []float64, float64) []float64, safety float64, material bool) (*matrix.COOMatrix, []float64, error) {
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
		connection := el
		if material {
			connection = el.plasticConnection()
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
		connection.ModifyCMQ()
		gvct = vecf(connection, tmatrix, gvct, safety)
	}
	return gmtx, gvct, nil
}

// AssemMassMatrix assigns the lumped nodal mass to free translations.
// Rotations stay in the stiffness system with zero mass.
func (frame *Frame) AssemMassMatrix() *matrix.COOMatrix {
	size := 6 * len(frame.Nodes)
	mmtx := matrix.NewCOOMatrix(size)
	for _, n := range frame.Nodes {
		if n.Mass == 0 {
			continue
		}
		for i := 0; i < 3; i++ {
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

func (frame *Frame) KP(safety float64) (*matrix.COOMatrix, []float64, error) {
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
	return frame.assemGlobalMatrix(matf, vecf, safety, true)
}

func (frame *Frame) KG(safety float64) (*matrix.COOMatrix, []float64, error) { // TODO: UNDER CONSTRUCTION
	matf := func(elem *Elem) ([][]float64, error) {
		return elem.bucklingGeometricStiffness()
	}
	vecf := func(elem *Elem, tmatrix [][]float64, gvct []float64, safety float64) []float64 {
		gvct = elem.AssemCMQ(tmatrix, gvct, safety)
		gvct = elem.ModifyTrueForce(tmatrix, gvct)
		return gvct
	}
	return frame.assemGlobalMatrix(matf, vecf, safety, false)
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

// UpdateStressPlastic commits element return-mapping responses only after all
// elements have succeeded, so a local failure cannot partially update the frame.
func (frame *Frame) UpdateStressPlastic(vec []float64) ([][]float64, error) {
	if len(vec) != 6*len(frame.Nodes) {
		return nil, fmt.Errorf("plastic displacement vector has size %d, want %d", len(vec), 6*len(frame.Nodes))
	}
	responses := make([]*PlasticResponse, len(frame.Elems))
	for i, el := range frame.Elems {
		if !el.IsValid {
			continue
		}
		gdisp := make([]float64, 12)
		for end := 0; end < 2; end++ {
			copy(gdisp[6*end:6*end+6], vec[6*el.Enod[end].Index:6*el.Enod[end].Index+6])
		}
		r, err := el.TrialPlasticResponse(gdisp)
		if err != nil {
			return nil, err
		}
		responses[i] = r
	}
	rtn := make([][]float64, len(frame.Elems))
	for i, r := range responses {
		if r != nil {
			frame.Elems[i].commitPlasticResponse(r)
			rtn[i] = r.Increment
		}
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
	otp.WriteString("  NO   KT NODE         N        Q1        Q2        MT        M1        M2        PHINGE\n\n")
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
	init                bool
	solver              string
	otp                 []string
	extra               [][]float64
	extraFrames         []*Frame
	staticMaterial      bool
	floorContact        *float64
	compressionLimit    float64
	compressionSections map[int]bool

	nlgeometry bool
	nlmaterial bool

	postprocess func(*Frame, [][]float64, []float64, []float64) (float64, bool)

	nlap                int
	delta               float64
	start               float64
	max                 float64
	eps                 float64
	newton              NewtonOptions
	startSet            bool
	maxSet              bool
	geometryIncremental bool
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
	cond.startSet = true
}
func (cond *AnalysisCondition) SetMax(m float64) {
	cond.max = m
	cond.maxSet = true
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
	rtn.WriteString(fmt.Sprintf("EXTRA LOAD  : %d\n", len(cond.extra)+len(cond.extraFrames)))
	rtn.WriteString("NON-LINEAR\n")
	rtn.WriteString(fmt.Sprintf("  GEOMETRY  : %t\n", cond.nlgeometry))
	rtn.WriteString(fmt.Sprintf("  MATERIAL  : %t\n", cond.nlmaterial))
	rtn.WriteString("STEP\n")
	rtn.WriteString(fmt.Sprintf("  NLAP      : %d\n", cond.nlap))
	rtn.WriteString(fmt.Sprintf("  DELTA     : %.3f\n", cond.delta))
	rtn.WriteString(fmt.Sprintf("  START     : %.3f\n", cond.start))
	rtn.WriteString(fmt.Sprintf("  MAX       : %.3f\n", cond.max))
	rtn.WriteString(fmt.Sprintf("POST PROCESS: %t", cond.hasStaticPost()))
	return rtn.String()
}

func (cond *AnalysisCondition) NonLinear() bool {
	return cond.nlgeometry || cond.nlmaterial
}

func (frame *Frame) StaticAnalysis(cancel context.CancelFunc, cond *AnalysisCondition) error {
	if cond == nil {
		return fmt.Errorf("StaticAnalysis: nil condition")
	}
	if cond.NonLinear() && cond.hasStaticPost() {
		return fmt.Errorf("StaticAnalysis: POST is supported only with the linear element formulation")
	}
	if cond.NonLinear() && (len(cond.extra) > 0 || len(cond.extraFrames) > 0) {
		return fmt.Errorf("StaticAnalysis: extra load cases require linear analysis")
	}
	if cond.nlgeometry {
		return frame.staticGeometry(cancel, cond)
	}
	if cond.nlmaterial {
		c := *cond
		c.staticMaterial, c.maxSet = true, true
		return frame.arclm101Newton(cancel, &c)
	}
	return frame.staticLinear(cancel, cond)
}

// Arclm101 performs material-nonlinear load control. With nlgeometry it uses
// finite rotations and the combined physical-force plastic return; otherwise
// the original small-displacement material solver is used.
func (frame *Frame) Arclm101(cancel context.CancelFunc, cond *AnalysisCondition) error {
	if cond != nil && cond.nlgeometry {
		c := *cond
		c.nlmaterial = true
		c.geometryIncremental = true
		return frame.staticGeometry(cancel, &c)
	}
	return frame.arclm101Newton(cancel, cond)
}

// notifyLap publishes a completed LAP and waits for its acknowledgement.
func (frame *Frame) notifyLap(lap int) {
	if frame.Lapch != nil {
		frame.Lapch <- lap
		<-frame.Lapch
	}
}

// beginLinearIteration prepares Arclm301/Arclm401. Both solve the full load
// on the reference geometry at every LAP, so a state saved by a nonlinear
// solver cannot be continued here.
func (frame *Frame) beginLinearIteration(name string, init bool) error {
	if init {
		frame.Initialise()
		return nil
	}
	if frame.geometry != nil || frame.transient != nil || (frame.loading != nil && frame.loading.steps != 0) {
		return fmt.Errorf("%s: saved state requires its original nonlinear solver", name)
	}
	return nil
}

// solveLinearIteration solves the full load for the current supports and
// element operators, and stores displacements, member forces and reactions.
func (frame *Frame) solveLinearIteration(elems []linearElement, load linearLoads) ([]float64, error) {
	size := 6 * len(frame.Nodes)
	k, conf, csize := frame.linearSystem(elems, size)
	u := make([]float64, size)
	if csize < size {
		rhs := frame.linearRHS(elems, load, 1, conf, make([]float64, size))
		answers, err := k.ToLLS(csize, conf).SolveRelative(frame.Pivot, linearPivotTolerance, rhs)
		if err != nil {
			return nil, frame.CheckSingularNode(err)
		}
		u = frame.FillConf(answers[0])
	}
	for i, v := range u {
		if !finitePlastic(v) {
			return nil, fmt.Errorf("non-finite displacement at DOF %d", i)
		}
	}
	frame.linearResult(elems, load, 1, conf, u)
	return u, nil
}

func (frame *Frame) writeLinearIteration(otp string) error {
	frame.loading = nil
	frame.plasticLoadValid = false
	if otp == "" {
		otp = "hogtxt.otp"
	}
	return writeStaticResult(frame, otp)
}

// soilSpringElement returns the operator of a soil spring whose axial
// stiffness follows the secant of its current length change y [cm]:
// the area is A up to y = 0.1 and A/(3.16 sqrt(y)) beyond.
func soilSpringElement(member *Elem, u []float64) (linearElement, error) {
	e := linearElementGeometry(member, true)
	length := e.Length0()
	deformed := 0.0
	for i := 0; i < 3; i++ {
		d := e.Enod[1].Coord[i] + u[6*e.Enod[1].Index+i] - e.Enod[0].Coord[i] - u[6*e.Enod[0].Index+i]
		deformed += d * d
	}
	if y := math.Abs(math.Sqrt(deformed)-length) * 100; y > 0.1 {
		sect := *e.Sect
		sect.Value = append([]float64(nil), sect.Value...)
		sect.Value[0] /= math.Sqrt(y) * 3.16
		e.Sect = &sect
	}
	return newLinearElement(&e)
}

// Arclm301 solves the full load with nonlinear soil springs. The members of
// the sections in sects take a secant axial stiffness from the previous
// LAP's displacement; iteration stops when the displacement change is
// below eps.
func (frame *Frame) Arclm301(otp string, init bool, sects []int, eps float64) error {
	if err := frame.beginLinearIteration("Arclm301", init); err != nil {
		return err
	}
	start := time.Now()
	laptime := func(message string) {
		fmt.Fprintf(frame.Output, "%s: %fsec\n", message, time.Since(start).Seconds())
	}
	soil := make(map[int]bool, len(sects))
	for _, sec := range sects {
		soil[sec] = true
	}
	elems, err := newLinearElements(frame, true)
	if err != nil {
		return err
	}
	load, err := frame.linearLoad(frame, true)
	if err != nil {
		return err
	}
	const maxLaps = 1000
	last := make([]float64, 6*len(frame.Nodes))
	for lap := 1; ; lap++ {
		if lap > maxLaps {
			return fmt.Errorf("Arclm301: soil springs did not converge in %d LAPs", maxLaps)
		}
		if lap > 1 {
			for i, member := range frame.Elems {
				if member.IsValid && soil[member.Sect.Num] {
					if elems[i], err = soilSpringElement(member, last); err != nil {
						return err
					}
				}
			}
		}
		laptime("Assem")
		u, err := frame.solveLinearIteration(elems, load)
		if err != nil {
			return err
		}
		norm := 0.0
		for i, v := range u {
			norm = math.Hypot(norm, v-last[i])
		}
		last = u
		laptime(fmt.Sprintf("LAP = %d NORM = %.5E", lap, norm))
		frame.notifyLap(lap)
		if norm < eps {
			break
		}
	}
	return frame.writeLinearIteration(otp)
}

// Arclm401 solves the full load with supports that cannot carry vertical
// tension. A support whose vertical reaction is tensile is released in all
// six directions; a released support that moves below its own elevation is
// restored. The result is written with the settled support set, and the
// frame's own supports are restored on return.
//
// wgtdict gives, by node number, a downward load that acts on the support
// but is not among the frame's nodal loads. It is added to the vertical
// reaction for the tension test only. Reactions already include the nodal
// loads applied at supports, so those must not be listed again.
func (frame *Frame) Arclm401(otp string, init bool, eps float64, wgtdict map[int]float64) error {
	if err := frame.beginLinearIteration("Arclm401", init); err != nil {
		return err
	}
	start := time.Now()
	laptime := func(message string) {
		fmt.Fprintf(frame.Output, "%s: %fsec\n", message, time.Since(start).Seconds())
	}
	type support struct {
		node     *Node
		conf     [6]bool
		released bool
	}
	var supports []*support
	for _, n := range frame.Nodes {
		if n.Conf[2] {
			s := &support{node: n}
			copy(s.conf[:], n.Conf)
			supports = append(supports, s)
		}
	}
	defer func() {
		for _, s := range supports {
			copy(s.node.Conf, s.conf[:])
		}
	}()
	elems, err := newLinearElements(frame, true)
	if err != nil {
		return err
	}
	load, err := frame.linearLoad(frame, true)
	if err != nil {
		return err
	}
	maxLaps := 2*len(supports) + 10
	for lap := 1; ; lap++ {
		if lap > maxLaps {
			return fmt.Errorf("Arclm401: supports did not settle in %d LAPs", maxLaps)
		}
		u, err := frame.solveLinearIteration(elems, load)
		if err != nil {
			return fmt.Errorf("Arclm401: LAP %d: %w", lap, err)
		}
		scale := 0.0
		for _, v := range u {
			scale = math.Max(scale, math.Abs(v))
		}
		changed := 0
		for _, s := range supports {
			n := s.node
			if !s.released {
				reaction := n.Reaction[2] + wgtdict[n.Num]
				if reaction < -1e-10*math.Max(1, math.Abs(n.Reaction[2])) {
					s.released = true
					clear(n.Conf)
					changed++
				}
			} else if n.Disp[2] < -1e-10*scale {
				s.released = false
				copy(n.Conf, s.conf[:])
				changed++
			}
		}
		laptime(fmt.Sprintf("LAP = %d CHANGED SUPPORTS = %d", lap, changed))
		frame.notifyLap(lap)
		if changed == 0 {
			break
		}
	}
	var nodes bytes.Buffer
	nodes.WriteString(":node")
	for _, s := range supports {
		if s.released {
			fmt.Fprintf(&nodes, " %d", s.node.Num)
		}
	}
	fmt.Fprintln(frame.Output, nodes.String())
	if err := frame.writeLinearIteration(otp); err != nil {
		return err
	}
	laptime("End")
	return nil
}

// bisectBuckling locates the rank-th positive buckling eigenvalue in reciprocal
// coordinates. It checks KE is positive definite after removing constrained DOFs.
// The returned factor belongs to the upper bracket, not necessarily to mu:
// the midpoint can be an exact eigenvalue and therefore singular.
func bisectBuckling(ke, kg *matrix.COOMatrix, csize int, conf []bool, rank int, eps, right float64, pivot chan int, progress func(float64, int)) (float64, *matrix.SymLDLTFactor, error) {
	if ke == nil || rank < 1 || rank > ke.Size-csize {
		return 0, nil, fmt.Errorf("invalid buckling search parameters")
	}
	search, err := newBucklingSearch(ke, kg, csize, conf, eps, right, pivot)
	if err != nil {
		return 0, nil, err
	}
	return search.eigenvalue(rank, progress)
}

func (frame *Frame) Bclng001(otp string, init bool, n int, eps float64, right float64) error {
	return frame.Bclng001WithDiagnostics(otp, init, n, eps, right, nil)
}

// Bclng001WithDiagnostics optionally records profiles and matched-shift legacy
// comparisons. CompareOnly assembles the real model and skips eigenvalue search.
func (frame *Frame) Bclng001WithDiagnostics(otp string, init bool, n int, eps float64, right float64, options *BucklingDiagnosticsOptions) (analysisErr error) { // TODO: speed up
	if n < 1 || eps <= 0 || math.IsNaN(eps) || math.IsInf(eps, 0) || right <= 0 || math.IsNaN(right) || math.IsInf(right, 0) {
		return fmt.Errorf("invalid buckling search parameters")
	}
	d, diagErr := startBucklingDiagnostics(options, frame.Output)
	if diagErr != nil {
		return diagErr
	}
	if d != nil {
		defer func() { analysisErr = errors.Join(analysisErr, d.close(analysisErr)) }()
	}
	defer frame.RestoreState(frame.SaveState())
	if init {
		frame.Initialise()
	}
	start := time.Now()
	laptime := func(message string) {
		end := time.Now()
		fmt.Fprintf(frame.Output, "%s: %fsec\n", message, (end.Sub(start)).Seconds())
	}
	measure := func(phase, backend string, rank int, fn func() error) error {
		if d == nil {
			return fn()
		}
		return d.measure(phase, backend, nil, rank, func(map[string]interface{}) error { return fn() })
	}
	notify := func(lap, rank int, phase string) {
		if frame.Lapch != nil {
			measure(phase, "ui", rank, func() error { frame.Lapch <- lap; <-frame.Lapch; return nil })
		}
	}
	var elems []linearElement
	var load linearLoads
	var kemtx, kgmtx *matrix.COOMatrix
	var geometric [][][]float64
	var csize int
	var conf []bool
	size := 6 * len(frame.Nodes)
	if err := measure("assemble_ke", "assembly", 0, func() error {
		var err error
		elems, err = newLinearElements(frame, false)
		if err != nil {
			return err
		}
		load, err = frame.linearLoad(frame, false)
		if err != nil {
			return err
		}
		kemtx, conf, csize = frame.linearSystem(elems, size)
		return nil
	}); err != nil {
		return err
	}
	if csize == size {
		return fmt.Errorf("buckling has no free DOFs")
	}
	solver := LLS(frame, func(string) {})
	var u []float64
	if err := measure("initial_static", "legacy_static", 0, func() error {
		rhs := frame.linearRHS(elems, load, 1, conf, make([]float64, size))
		answers, err := solver.Solve(kemtx, csize, conf, rhs)
		if err != nil {
			return frame.CheckSingularNode(err)
		}
		u = frame.FillConf(answers[0])
		return nil
	}); err != nil {
		return err
	}
	if err := measure("assemble_kg", "assembly", 0, func() error {
		var err error
		kgmtx, geometric, err = frame.bucklingPreloadGeometric(elems, load, u)
		return err
	}); err != nil {
		return err
	}
	laptime("initial analysis solved")
	if d != nil {
		d.log("model", map[string]interface{}{"total_dofs": kemtx.Size, "free_dofs": kemtx.Size - csize, "constrained_dofs": csize, "requested_modes": n, "eps_mu": eps, "right_mu": right})
		if err := d.explicitShifts(kemtx, kgmtx, csize, conf); err != nil {
			return err
		}
		if options.CompareOnly {
			fmt.Fprintf(frame.Output, "BCLNG DIAGNOSTICS: comparison-only complete; no eigenvalue/OTP output\n")
			return nil
		}
	}
	frame.EigenValue = make([]float64, n)
	frame.EigenVector = make([][]float64, n)
	// Validate bounds/operators once. Numerical pivot updates do not trigger
	// per-DOF GUI work; shift progress below is throttled instead.
	searchStart := time.Now()
	search, err := newBucklingSearchDiagnostics(kemtx, kgmtx, csize, conf, eps, right, nil, d)
	if err != nil {
		return err
	}
	searchDuration := time.Since(searchStart)
	if search.gershgorin.Available {
		fmt.Fprintf(frame.Output, "BUCKLING GERSHGORIN: mu=[%.16g, %.16g], search=[%.16g, %.16g]\n", search.gershgorin.Lower, search.gershgorin.Upper, search.lower, search.right)
		if lower, upper, ok := search.gershgorin.PositiveReciprocalBounds(); ok {
			fmt.Fprintf(frame.Output, "BUCKLING GERSHGORIN: positive Lambda=[%.16g, %.16g]\n", lower, upper)
		}
	} else {
		fmt.Fprintf(frame.Output, "BUCKLING GERSHGORIN: finite bounds unavailable (%s, rows=%d); using search=[0, %.16g]\n", search.gershgorin.Reason, search.gershgorin.NonDominantRows, search.right)
	}
	var vectorDuration time.Duration
	lastProgress := time.Now()
	// Cache both KE-orthonormal vectors and their KE products across modes.
	locked := make([][]float64, 0, n)
	klocked := make([][]float64, 0, n)
	for i := 0; i < n; i++ {
		lap := 0
		searchStart = time.Now()
		mu, C, err := search.eigenvalue(i+1, func(mu float64, neg int) {
			lap++
			if time.Since(lastProgress) < 200*time.Millisecond {
				return
			}
			lastProgress = time.Now()
			if neg > i {
				fmt.Fprintf(frame.Output, "LAMBDA<%.14f\n", 1.0/mu)
			} else {
				fmt.Fprintf(frame.Output, "LAMBDA>%.14f\n", 1.0/mu)
			}
			notify(lap+1, i+1, "progress_wait")
		})
		searchDuration += time.Since(searchStart)
		if err != nil {
			return err
		}
		vectorStart := time.Now()
		var answer []float64
		if d == nil {
			mu, answer, err = search.inverseVector(C, mu, eps, locked, klocked, i+1, nil)
		} else {
			mu, answer, err = d.inverse(search, C, mu, eps, locked, klocked, i+1)
		}
		vectorDuration += time.Since(vectorStart)
		if err != nil {
			return fmt.Errorf("buckling mode %d: %w", i+1, err)
		}
		locked = append(locked, answer)
		klocked = append(klocked, search.elastic.MulV(answer))
		lastvec := frame.FillConf(answer)
		laptime(fmt.Sprintf("\nEIG %d: %.14f", i+1, 1.0/mu))
		frame.EigenValue[i] = 1.0 / mu
		frame.EigenVector[i] = lastvec
		if frame.Lapch != nil {
			frame.previewEigenMode(elems, geometric, conf, lastvec)
			notify(i+1, i+1, "mode_completion_wait")
		}
	}
	fmt.Fprintf(frame.Output, "BUCKLING: inertia=%d factors=%d bracket_factors=%d proposals=%d accepted=%d cached=%d search=%.3fs vectors=%.3fs\n", search.inertiaEvaluations, search.solveFactorizations, search.bracketFactorizations, search.refinementAttempts, search.refinementsAccepted, search.cacheHits, searchDuration.Seconds(), vectorDuration.Seconds())
	if d != nil {
		d.log("buckling_summary", map[string]interface{}{"inertia_evaluations": search.inertiaEvaluations, "solve_factorizations": search.solveFactorizations, "bracket_factorizations": search.bracketFactorizations, "total_factorizations": search.inertiaEvaluations + search.solveFactorizations + search.bracketFactorizations, "proposals": search.refinementAttempts, "accepted": search.refinementsAccepted, "cache_hits": search.cacheHits, "search_seconds": searchDuration.Seconds(), "vector_seconds": vectorDuration.Seconds()})
	}
	if otp == "" {
		otp = "hogtxt.otp"
	}
	w, err := os.Create(otp)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = frame.WriteBclngTo(w)
	return err
}

// VibrationalEigenAnalysis computes ascending omega squared and M-orthonormal
// modes, using the shared sparse pivoted solver and inertia-validated brackets.
func (frame *Frame) VibrationalEigenAnalysis(otp string, init bool, n int, eps float64, right float64) error {
	return frame.VibrationalEigenAnalysisWithDiagnostics(otp, init, n, eps, right, nil)
}
