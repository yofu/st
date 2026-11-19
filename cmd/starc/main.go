// starc runs the same finite-rotation solvers as ST without restarting an
// existing GUI. It can prepare an extracted input or read an existing .inl.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/yofu/st/arclm"
	st "github.com/yofu/st/stlib"
	"math"
	"os"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	model := flag.String("model", "", "ST INP to extract (optional)")
	coords := flag.String("coordinates", "", "optional exact GUI coordinates JSON")
	inl := flag.String("inl", "", "extracted input to read/save")
	out := flag.String("out", "arc.otl", "result output")
	ds := flag.Float64("ds", .03, "initial arc length in metres")
	dsmax := flag.Float64("dsmax", .05, "maximum arc length")
	dsmin := flag.Float64("dsmin", 1e-6, "minimum arc length")
	scale := flag.Float64("loadscale", .02, "length scale per unit load factor")
	steps := flag.Int("steps", 600, "maximum accepted points")
	start := flag.Float64("start", 0, "load-controlled preload reached with increments <=0.1")
	stopUz := flag.Float64("stop-uz", -1.85, "stop when node 61 uz reaches this displacement, metres (zero disables)")
	loadEnd := flag.Float64("load-end", 0, "load-control verification target instead of arc length (zero disables)")
	stopTurn := flag.Float64("stop-after-turn", 0, "stop below this load factor on the descending branch (zero disables)")
	shape := flag.Bool("exact-h200", false, "restore H200 section 513 analytical properties after ST text serialization")
	flag.Parse()
	var f *arclm.Frame
	if *model != "" {
		b, err := os.ReadFile(*model)
		if err != nil {
			return err
		}
		sf := st.NewFrame()
		if err = sf.ReadInpSnapshot(b, *model); err != nil {
			return err
		}
		if *coords != "" {
			var ns []struct {
				ID          int       `json:"id"`
				Coordinates []float64 `json:"coordinates"`
			}
			b, err = os.ReadFile(*coords)
			if err != nil {
				return err
			}
			if err = json.Unmarshal(b, &ns); err != nil {
				return err
			}
			if len(ns) != len(sf.Nodes) {
				return fmt.Errorf("coordinate node count mismatch")
			}
			for _, n := range ns {
				v := sf.Nodes[n.ID]
				if v == nil || len(n.Coordinates) != 3 {
					return fmt.Errorf("invalid exact node %d", n.ID)
				}
				copy(v.Coord, n.Coordinates)
			}
		}
		if *shape {
			sec := sf.Sects[513]
			if sec == nil {
				return fmt.Errorf("missing section 513")
			}
			s, err := st.NewHKYOU([]string{"20", "20", "0.8", "1.2"})
			if err != nil {
				return err
			}
			sec.Figs[0].SetShapeProperty(s)
		}
		if err = sf.ExtractArclm(*out + ".wgt"); err != nil {
			return err
		}
		f = sf.Arclms["L"]
		if *inl != "" {
			if err = f.SaveInput(*inl); err != nil {
				return err
			}
		}
	} else {
		f = arclm.NewFrame()
		if err := f.ReadInput(*inl); err != nil {
			return err
		}
	}
	f.Output = os.Stdout
	f.Lapch = nil
	f.Pivot = nil
	c := arclm.NewAnalysisCondition()
	c.SetNlgeometry(true)
	c.SetOutput([]string{*out})
	if *loadEnd != 0 {
		c.SetNlap(int(math.Ceil(*loadEnd / .1)))
		c.SetDelta(.1)
		c.SetMax(*loadEnd)
		return f.StaticAnalysis(nil, c)
	}
	if *start > 0 {
		c.SetNlap(int(math.Ceil(*start / .1)))
		c.SetDelta(.1)
		c.SetMax(*start)
		ext := filepath.Ext(*out)
		c.SetOutput([]string{(*out)[:len(*out)-len(ext)] + "_preload" + ext})
		if err := f.StaticAnalysis(nil, c); err != nil {
			return err
		}
		c = arclm.NewAnalysisCondition()
		c.SetNlgeometry(true)
		c.SetInit(false)
		c.SetStart(*start)
	}
	c.SetNlap(*steps)
	c.SetOutput([]string{*out})
	peakFactor := -math.MaxFloat64
	c.SetArcLength(arclm.ArcLengthOptions{Step: *ds, MinStep: *dsmin, MaxStep: *dsmax, LoadScale: *scale, MonitorNode: 61, OnStep: func(f *arclm.Frame, p arclm.ArcLengthPoint) (bool, error) {
		if p.Factor > peakFactor {
			peakFactor = p.Factor
		}
		if *stopTurn != 0 && p.Factor < *stopTurn && peakFactor > *stopTurn {
			return true, nil
		}
		if *stopUz == 0 {
			return false, nil
		}
		for _, n := range f.Nodes {
			if n.Num == 61 && n.Disp[2] <= *stopUz {
				return true, nil
			}
		}
		return false, nil
	}})
	return f.StaticAnalysis(nil, c)
}
