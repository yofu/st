package arclm

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
)

type timeHistoryOutput struct {
	steps, nodes *csv.Writer
	files        []*os.File
	indices      []int
	opts         TimeHistoryOptions
}

func newTimeHistoryOutput(f *Frame, o TimeHistoryOptions) (*timeHistoryOutput, error) {
	w := &timeHistoryOutput{opts: o}
	wanted := make(map[int]bool)
	for _, n := range o.OutputNodes {
		if wanted[n] {
			return nil, fmt.Errorf("duplicate output node %d", n)
		}
		wanted[n] = true
	}
	for i, n := range f.Nodes {
		if len(o.OutputNodes) == 0 || wanted[n.Num] {
			w.indices = append(w.indices, i)
			delete(wanted, n.Num)
		}
	}
	if len(wanted) != 0 {
		return nil, fmt.Errorf("requested output nodes not found: %v", wanted)
	}
	if o.Output == "" {
		return w, nil
	}
	for _, suffix := range []string{".steps.csv", ".nodes.csv"} {
		// Exclusive creation prevents silent loss of an earlier transient run.
		file, err := os.OpenFile(o.Output+suffix, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			var paths []string
			for _, p := range w.files {
				paths = append(paths, p.Name())
			}
			w.close()
			for _, p := range paths {
				os.Remove(p)
			}
			return nil, err
		}
		w.files = append(w.files, file)
	}
	w.steps, w.nodes = csv.NewWriter(w.files[0]), csv.NewWriter(w.files[1])
	if err := w.steps.Write([]string{"step", "time_s", "substeps_total", "iterations_last", "agx_m_s2", "agy_m_s2", "residual_force", "tolerance_force", "kinetic_energy", "input_work", "internal_work_trapezoid", "damping_work", "plastic_ends", "rayleigh_mode1", "rayleigh_mode2", "rayleigh_period1_s", "rayleigh_period2_s", "rayleigh_alpha", "rayleigh_eta"}); err != nil {
		w.close()
		return nil, err
	}
	header := []string{"step", "time_s", "node"}
	for _, name := range []string{"u", "v", "a_relative", "a_absolute", "reaction"} {
		for _, axis := range []string{"x", "y", "z", "rx", "ry", "rz"} {
			header = append(header, name+"_"+axis)
		}
	}
	if err := w.nodes.Write(header); err != nil {
		w.close()
		return nil, err
	}
	return w, nil
}

func timeHistoryNumber(x float64) string { return strconv.FormatFloat(x, 'g', 17, 64) }

func (w *timeHistoryOutput) write(f *Frame) error {
	s := f.transient
	if w.steps != nil {
		plastic := 0
		for _, e := range f.Elems {
			for _, p := range e.Phinge {
				if p {
					plastic++
				}
			}
		}
		row := []string{strconv.Itoa(s.Step), timeHistoryNumber(s.Time), strconv.Itoa(s.Substeps), strconv.Itoa(s.Iterations)}
		for _, v := range []float64{s.Ground[0], s.Ground[1], s.Residual, s.Tolerance, s.KineticEnergy, s.InputWork, s.InternalWork, s.DampingWork} {
			row = append(row, timeHistoryNumber(v))
		}
		row = append(row, strconv.Itoa(plastic))
		row = append(row, strconv.Itoa(s.RayleighModes[0]), strconv.Itoa(s.RayleighModes[1]))
		for _, v := range []float64{s.RayleighPeriods[0], s.RayleighPeriods[1], s.RayleighAlpha, s.RayleighEta} {
			row = append(row, timeHistoryNumber(v))
		}
		if err := w.steps.Write(row); err != nil {
			return err
		}
		for _, i := range w.indices {
			n := f.Nodes[i]
			row = []string{strconv.Itoa(s.Step), timeHistoryNumber(s.Time), strconv.Itoa(n.Num)}
			for _, values := range [][]float64{n.Disp, s.Velocity[6*i : 6*i+6], s.Acceleration[6*i : 6*i+6]} {
				for _, v := range values {
					row = append(row, timeHistoryNumber(v))
				}
			}
			for j := 0; j < 6; j++ {
				v := s.Acceleration[6*i+j]
				if j < 2 {
					v += s.Ground[j]
				}
				row = append(row, timeHistoryNumber(v))
			}
			for _, v := range n.Reaction {
				row = append(row, timeHistoryNumber(v))
			}
			if err := w.nodes.Write(row); err != nil {
				return err
			}
		}
		w.steps.Flush()
		w.nodes.Flush()
		if err := errors.Join(w.steps.Error(), w.nodes.Error()); err != nil {
			return err
		}
	}
	if w.opts.Output != "" && w.opts.SaveEvery > 0 && s.Step%w.opts.SaveEvery == 0 {
		file, err := os.OpenFile(fmt.Sprintf("%s@%d.otp", w.opts.Output, s.Step), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			return err
		}
		_, writeErr := f.WriteTo(file)
		return errors.Join(writeErr, file.Close())
	}
	return nil
}

func (w *timeHistoryOutput) close() error {
	if w == nil {
		return nil
	}
	var result error
	for _, c := range []*csv.Writer{w.steps, w.nodes} {
		if c != nil {
			c.Flush()
			result = errors.Join(result, c.Error())
		}
	}
	for _, f := range w.files {
		result = errors.Join(result, f.Close())
	}
	w.files = nil
	return result
}
