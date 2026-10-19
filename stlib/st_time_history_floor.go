package st

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"

	"github.com/yofu/st/arclm"
)

var timeHistoryFloorSuffixes = []string{".floors.csv", ".stories.csv", ".peaks.csv"}

type transientFloor struct {
	nodes []int
	z     float64
}
type transientCutMember struct {
	elem  int
	lower int
	axes  [2][6]float64
}
type transientStory struct {
	height, cut float64
	members     []transientCutMember
}
type transientPeakValue struct {
	value, time float64
	step, node  int
}
type transientPeak struct {
	kind, quantity, axis, unit string
	index                      int
	have                       bool
	max, min, abs              transientPeakValue
}

func (p *transientPeak) add(value float64, s *arclm.TimeHistoryState, node int) {
	v := transientPeakValue{value, s.Time, s.Step, node}
	if !p.have || value > p.max.value {
		p.max = v
	}
	if !p.have || value < p.min.value {
		p.min = v
	}
	if !p.have || math.Abs(value) > math.Abs(p.abs.value) {
		p.abs = v
	}
	p.have = true
}

// TimeHistoryFloorOutput aggregates every requested output step independently
// of GUI drawing. Membership/axes are cached from the input model. Floor means
// are arithmetic means of model nodes included in the analysis (as in the
// existing pushover summary), not rigid-diaphragm/center-of-mass estimates.
// Files are opened lazily at the first successful t=0 response.
type TimeHistoryFloorOutput struct {
	af                       *arclm.Frame
	prefix                   string
	floors                   []transientFloor
	stories                  []transientStory
	initialDisp              [][2]float64
	peaks                    []transientPeak
	floorPeaks               [][2][2]int // mean and all-node absolute acceleration, X/Y
	storyPeaks               [][3][2]int // total drift, incremental drift, member shear
	lastStep                 int
	lastTime                 float64
	files                    []*os.File
	floorWriter, storyWriter *csv.Writer
	closed                   bool
}

func transientFinite(x float64) bool   { return !math.IsNaN(x) && !math.IsInf(x, 0) }
func transientNumber(x float64) string { return strconv.FormatFloat(x, 'g', 17, 64) }

// NewTimeHistoryFloorOutput uses Ai.Boundary and FloorNodes. It does not use
// node/element selection or Show.Unit, so exports are independent of GUI state.
// Shears are restoring member-end forces; Rayleigh viscous forces are excluded.
func NewTimeHistoryFloorOutput(frame *Frame, af *arclm.Frame, prefix string) (*TimeHistoryFloorOutput, error) {
	if frame == nil || af == nil || frame.Ai == nil || frame.Ai.Nfloor < 2 || len(frame.Ai.Boundary) != frame.Ai.Nfloor+1 {
		return nil, fmt.Errorf("timehistory floors: define at least two levels and Nfloor+1 boundaries")
	}
	for i, z := range frame.Ai.Boundary {
		if !transientFinite(z) || i > 0 && z <= frame.Ai.Boundary[i-1] {
			return nil, fmt.Errorf("timehistory floors: boundaries must increase strictly")
		}
	}
	if prefix == "" {
		return nil, fmt.Errorf("timehistory floors: output prefix is empty")
	}
	for _, suffix := range timeHistoryFloorSuffixes {
		if _, err := os.Stat(prefix + suffix); err == nil {
			return nil, fmt.Errorf("floor response already exists: %s", prefix+suffix)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	w := &TimeHistoryFloorOutput{af: af, prefix: prefix, lastStep: -1}
	index := make(map[int]int, len(af.Nodes))
	for i, n := range af.Nodes {
		if n == nil || n.Index != i || len(n.Coord) != 3 || len(n.Disp) != 6 {
			return nil, fmt.Errorf("timehistory floors: invalid analysis node index")
		}
		if _, ok := index[n.Num]; ok {
			return nil, fmt.Errorf("duplicate analysis node %d", n.Num)
		}
		index[n.Num] = i
	}
	for i, nodes := range frame.FloorNodes(nil, nil) {
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].Num < nodes[j].Num })
		floor := transientFloor{}
		for _, n := range nodes {
			if k, ok := index[n.Num]; ok {
				floor.nodes = append(floor.nodes, k)
				floor.z += n.Coord[2]
			}
		}
		if len(floor.nodes) == 0 {
			return nil, fmt.Errorf("timehistory floors: level %d has no model nodes in the analysis", i)
		}
		floor.z /= float64(len(floor.nodes))
		if !transientFinite(floor.z) || i > 0 && floor.z <= w.floors[i-1].z {
			return nil, fmt.Errorf("timehistory floors: mean level elevations must increase")
		}
		w.floors = append(w.floors, floor)
	}
	for i := 1; i < len(w.floors); i++ {
		w.stories = append(w.stories, transientStory{height: w.floors[i].z - w.floors[i-1].z, cut: frame.Ai.Boundary[i]})
	}
	// Use every valid analysis element, including automatically generated
	// members. Choose its geometrically lower end, independent of node order.
	order := make([]int, len(af.Elems))
	for i := range order {
		if af.Elems[i] == nil {
			return nil, fmt.Errorf("timehistory floors: nil analysis element")
		}
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool { return af.Elems[order[i]].Num < af.Elems[order[j]].Num })
	for _, eindex := range order {
		e := af.Elems[eindex]
		if e == nil {
			return nil, fmt.Errorf("timehistory floors: nil analysis element")
		}
		if !e.IsValid {
			continue
		}
		if len(e.Enod) != 2 {
			return nil, fmt.Errorf("timehistory floors: element %d has invalid ends", e.Num)
		}
		lower := 0
		if e.Enod[1].Coord[2] < e.Enod[0].Coord[2] {
			lower = 1
		}
		lo, hi := e.Enod[lower].Coord[2], e.Enod[1-lower].Coord[2]
		if lo == hi {
			continue
		}
		el := *e
		el.Enod = make([]*arclm.Node, 2)
		for j, n := range e.Enod {
			c := *n
			c.Disp = make([]float64, 6)
			el.Enod[j] = &c
		}
		t, err := el.TransMatrix()
		if err != nil {
			return nil, err
		}
		member := transientCutMember{elem: eindex, lower: lower}
		for axis := 0; axis < 2; axis++ {
			for j := 0; j < 6; j++ {
				member.axes[axis][j] = t[6*lower+j][6*lower+axis]
			}
		}
		for i := range w.stories {
			if lo < w.stories[i].cut && hi >= w.stories[i].cut {
				w.stories[i].members = append(w.stories[i].members, member)
			}
		}
	}
	addPeak := func(kind string, index int, quantity, axis, unit string) int {
		n := len(w.peaks)
		w.peaks = append(w.peaks, transientPeak{kind: kind, index: index, quantity: quantity, axis: axis, unit: unit})
		return n
	}
	w.floorPeaks = make([][2][2]int, len(w.floors))
	for i := range w.floors {
		for q, name := range []string{"mean_absolute_acceleration", "node_absolute_acceleration"} {
			for axis, label := range []string{"X", "Y"} {
				w.floorPeaks[i][q][axis] = addPeak("floor", i, name, label, "cm/s2")
			}
		}
	}
	w.storyPeaks = make([][3][2]int, len(w.stories))
	for i := range w.stories {
		for q, name := range []string{"drift", "incremental_drift", "member_shear"} {
			for axis, label := range []string{"X", "Y"} {
				unit := "rad"
				if q == 2 {
					unit = "model_force"
				}
				w.storyPeaks[i][q][axis] = addPeak("story", i+1, name, label, unit)
			}
		}
	}
	return w, nil
}

func (w *TimeHistoryFloorOutput) open() error {
	for _, suffix := range timeHistoryFloorSuffixes {
		f, err := os.OpenFile(w.prefix+suffix, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			var paths []string
			for _, created := range w.files {
				paths = append(paths, created.Name())
			}
			w.Close()
			for _, p := range paths {
				os.Remove(p)
			}
			return err
		}
		w.files = append(w.files, f)
	}
	w.floorWriter, w.storyWriter = csv.NewWriter(w.files[0]), csv.NewWriter(w.files[1])
	if err := w.floorWriter.Write([]string{"step", "time_s", "floor", "elevation_m", "nodes", "mean_displacement_x_m", "mean_displacement_y_m", "mean_absolute_acc_x_cm_s2", "mean_absolute_acc_y_cm_s2", "node_max_abs_acc_x_cm_s2", "node_max_abs_acc_x_signed_cm_s2", "node_max_abs_acc_x_node", "node_max_abs_acc_y_cm_s2", "node_max_abs_acc_y_signed_cm_s2", "node_max_abs_acc_y_node"}); err != nil {
		return err
	}
	return w.storyWriter.Write([]string{"step", "time_s", "story", "lower_floor", "upper_floor", "height_m", "cut_elevation_m", "crossing_members", "drift_x_rad", "drift_y_rad", "incremental_drift_x_rad", "incremental_drift_y_rad", "member_shear_x", "member_shear_y"})
}

// Write is suitable for TimeHistoryOptions.OnStep. Initial static deformation
// is retained in total drift; incremental drift is measured from the t=0 state.
// Acceleration means and envelopes use absolute (relative + ground) acceleration.
func (w *TimeHistoryFloorOutput) Write(s *arclm.TimeHistoryState) error {
	if w.closed || s == nil || len(s.Acceleration) != 6*len(w.af.Nodes) || s.Step != w.lastStep+1 || !transientFinite(s.Time) || w.lastStep < 0 && s.Time != 0 || w.lastStep >= 0 && s.Time <= w.lastTime {
		return fmt.Errorf("timehistory floors: invalid/nonconsecutive response state")
	}
	for _, v := range s.Acceleration {
		if !transientFinite(v) {
			return fmt.Errorf("non-finite floor acceleration")
		}
	}
	for _, v := range s.Ground {
		if !transientFinite(v) {
			return fmt.Errorf("non-finite ground acceleration")
		}
	}
	meanU, meanA := make([][2]float64, len(w.floors)), make([][2]float64, len(w.floors))
	nodeMax := make([][2]transientPeakValue, len(w.floors))
	for i, floor := range w.floors {
		for k, j := range floor.nodes {
			n := w.af.Nodes[j]
			for axis := 0; axis < 2; axis++ {
				u := n.Disp[axis]
				if !transientFinite(u) {
					return fmt.Errorf("non-finite floor displacement at node %d", n.Num)
				}
				acc := 100 * (s.Acceleration[6*j+axis] + s.Ground[axis])
				if !transientFinite(acc) {
					return fmt.Errorf("non-finite floor absolute acceleration")
				}
				meanU[i][axis] += u
				meanA[i][axis] += acc
				if k == 0 || math.Abs(acc) > math.Abs(nodeMax[i][axis].value) {
					nodeMax[i][axis] = transientPeakValue{value: acc, node: n.Num}
				}
				w.peaks[w.floorPeaks[i][1][axis]].add(acc, s, n.Num)
			}
		}
		for axis := 0; axis < 2; axis++ {
			meanU[i][axis] /= float64(len(floor.nodes))
			meanA[i][axis] /= float64(len(floor.nodes))
			if !transientFinite(meanU[i][axis]) || !transientFinite(meanA[i][axis]) {
				return fmt.Errorf("non-finite mean floor response")
			}
			w.peaks[w.floorPeaks[i][0][axis]].add(meanA[i][axis], s, 0)
		}
	}
	if w.lastStep < 0 {
		w.initialDisp = append([][2]float64(nil), meanU...)
		if err := w.open(); err != nil {
			return err
		}
	}
	for i, floor := range w.floors {
		row := []string{strconv.Itoa(s.Step), transientNumber(s.Time), strconv.Itoa(i), transientNumber(floor.z), strconv.Itoa(len(floor.nodes)), transientNumber(meanU[i][0]), transientNumber(meanU[i][1]), transientNumber(meanA[i][0]), transientNumber(meanA[i][1])}
		for axis := 0; axis < 2; axis++ {
			v := nodeMax[i][axis]
			row = append(row, transientNumber(math.Abs(v.value)), transientNumber(v.value), strconv.Itoa(v.node))
		}
		if err := w.floorWriter.Write(row); err != nil {
			return err
		}
	}
	for i, story := range w.stories {
		var shear, drift, increment [2]float64
		for _, member := range story.members {
			e := w.af.Elems[member.elem]
			if len(e.Stress) != 12 {
				return fmt.Errorf("invalid story member stress")
			}
			for axis := 0; axis < 2; axis++ {
				for j := 0; j < 6; j++ {
					v := e.Stress[6*member.lower+j]
					if !transientFinite(v) {
						return fmt.Errorf("non-finite story member stress")
					}
					shear[axis] -= member.axes[axis][j] * v
				}
			}
		}
		for axis := 0; axis < 2; axis++ {
			drift[axis] = (meanU[i+1][axis] - meanU[i][axis]) / story.height
			increment[axis] = drift[axis] - (w.initialDisp[i+1][axis]-w.initialDisp[i][axis])/story.height
			for q, v := range []float64{drift[axis], increment[axis], shear[axis]} {
				if !transientFinite(v) {
					return fmt.Errorf("non-finite story response")
				}
				w.peaks[w.storyPeaks[i][q][axis]].add(v, s, 0)
			}
		}
		row := []string{strconv.Itoa(s.Step), transientNumber(s.Time), strconv.Itoa(i + 1), strconv.Itoa(i), strconv.Itoa(i + 1), transientNumber(story.height), transientNumber(story.cut), strconv.Itoa(len(story.members))}
		for _, v := range []float64{drift[0], drift[1], increment[0], increment[1], shear[0], shear[1]} {
			row = append(row, transientNumber(v))
		}
		if err := w.storyWriter.Write(row); err != nil {
			return err
		}
	}
	w.floorWriter.Flush()
	w.storyWriter.Flush()
	if err := errors.Join(w.floorWriter.Error(), w.storyWriter.Error()); err != nil {
		return err
	}
	if err := w.writePeaks(); err != nil {
		return err
	}
	w.lastStep, w.lastTime = s.Step, s.Time
	return nil
}

func (w *TimeHistoryFloorOutput) writePeaks() error {
	var buffer bytes.Buffer
	c := csv.NewWriter(&buffer)
	if err := c.Write([]string{"location", "index", "quantity", "direction", "unit", "maximum", "maximum_time_s", "maximum_step", "maximum_node", "minimum", "minimum_time_s", "minimum_step", "minimum_node", "absolute_maximum", "absolute_maximum_signed", "absolute_maximum_time_s", "absolute_maximum_step", "absolute_maximum_node"}); err != nil {
		return err
	}
	for _, p := range w.peaks {
		row := []string{p.kind, strconv.Itoa(p.index), p.quantity, p.axis, p.unit}
		for _, v := range []transientPeakValue{p.max, p.min} {
			row = append(row, transientNumber(v.value), transientNumber(v.time), strconv.Itoa(v.step), strconv.Itoa(v.node))
		}
		v := p.abs
		row = append(row, transientNumber(math.Abs(v.value)), transientNumber(v.value), transientNumber(v.time), strconv.Itoa(v.step), strconv.Itoa(v.node))
		if err := c.Write(row); err != nil {
			return err
		}
	}
	c.Flush()
	if err := c.Error(); err != nil {
		return err
	}
	if _, err := w.files[2].WriteAt(buffer.Bytes(), 0); err != nil {
		return err
	}
	return w.files[2].Truncate(int64(buffer.Len()))
}

func (w *TimeHistoryFloorOutput) Close() error {
	if w == nil || w.closed {
		return nil
	}
	w.closed = true
	var err error
	for _, c := range []*csv.Writer{w.floorWriter, w.storyWriter} {
		if c != nil {
			c.Flush()
			err = errors.Join(err, c.Error())
		}
	}
	for _, f := range w.files {
		err = errors.Join(err, f.Close())
	}
	w.files = nil
	return err
}
