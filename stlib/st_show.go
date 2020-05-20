package st

import (
	"fmt"
	"strings"
)

var (
	NODECAPTIONS    = []string{"NC_NUM", "NC_WEIGHT", "NC_ZCOORD", "NC_DX", "NC_DY", "NC_DZ", "NC_TX", "NC_TY", "NC_TZ", "NC_RX", "NC_RY", "NC_RZ", "NC_MX", "NC_MY", "NC_MZ", "NC_PILE"}
	NODECAPTIONNAME = []string{"節点番号", "節点重量", "Z座標", "X方向変位", "Y方向変位", "Z方向変位", "X軸回り変形角[rad]", "Y軸回り変形角[rad]", "Z軸回り変形角[rad]",
		"X方向反力", "Y方向反力", "Z方向反力", "X軸回り反モーメント", "Y軸回り反モーメント", "Z軸回り反モーメント", "杭番号"}
	ELEMCAPTIONS    = []string{"EC_NUM", "EC_SECT", "EC_RATE_L", "EC_RATE_S", "EC_PREST", "EC_STIFF_X", "EC_STIFF_Y", "EC_DRIFT_X", "EC_DRIFT_Y", "EC_TENSILE_X", "EC_TENSILE_Y", "EC_WIDTH", "EC_HEIGHT"}
	ELEMCAPTIONNAME = []string{"部材番号", "断面番号", "断面検定比", "断面検定比", "プレストレス", "X方向水平剛性", "Y方向水平剛性", "X方向層間変形角[rad]", "Y方向層間変形角[rad]", "幅", "高さ"}
	SRCANS          = []string{"SRCAN_L", "SRCAN_S", "SRCAN_Q", "SRCAN_M"}
)

const ( // NodeCaption
	NC_NUM = 1 << iota
	NC_WEIGHT
	NC_ZCOORD
	NC_DX
	NC_DY
	NC_DZ
	NC_TX
	NC_TY
	NC_TZ
	NC_RX
	NC_RY
	NC_RZ
	NC_MX
	NC_MY
	NC_MZ
	NC_PILE
)
const ( // ElemCaption
	EC_NUM = 1 << iota
	EC_SECT
	EC_RATE_L
	EC_RATE_S
	EC_PREST
	EC_STIFF_X
	EC_STIFF_Y
	EC_DRIFT_X
	EC_DRIFT_Y
	EC_TENSILE_X
	EC_TENSILE_Y
	EC_WIDTH
	EC_HEIGHT
)
const (
	SRCAN_L = 1 << iota
	SRCAN_S
	SRCAN_Q
	SRCAN_M
)

const ( // Rate
	RATE_L = 1 << iota
	RATE_S
	RATE_Q
	RATE_M
)

const ( // PlotState
	PLOT_UNDEFORMED = 1 << iota
	PLOT_DEFORMED
)

const (
	RANGE_MAX = 1000.0
	RANGE_MIN = -100.0
)

// Line Color
var ( // Boundary for Rainbow (length should be <= 6)
	RateBoundary   = []float64{0.5, 0.6, 0.7, 0.71428, 0.9, 1.0}
	HeightBoundary = []float64{0.5, 1.0, 1.5, 2.0, 2.5, 3.0}
	EnergyBoundary = []float64{1.1, 2.0, 3.0, 4.0, 5.0, 6.0}
)
var (
	// ECOLORS = []string{ "WHITE", "BLACK", "BY SECTION", "BY RATE", "BY HEIGHT", "BY N" }
	ECOLORS = []string{"WHITE", "BLACK", "BY SECTION", "BY RATE", "BY N", "BY STRONG", "BY ENERGY"}
	PERIODS = []string{"L", "X", "Y"}
)

const (
	ECOLOR_WHITE = iota
	ECOLOR_BLACK
	ECOLOR_SECT
	ECOLOR_RATE
	// ECOLOR_HEIGHT
	ECOLOR_N
	ECOLOR_STRONG
	ECOLOR_ENERGY
)

type Show struct {
	Frame *Frame

	Unit     []float64
	UnitName []string

	ColorMode uint

	NodeCaption uint
	ElemCaption uint
	SrcanRate   uint

	Energy bool

	GlobalAxis      bool
	GlobalAxisSize  float64
	ElementAxis     bool
	ElementAxisSize float64

	NodeNormal     bool
	NodeNormalSize float64
	ElemNormal     bool
	ElemNormalSize float64
	CmqLine        bool

	Conf     bool
	ConfSize float64

	Bond     bool
	Phinge   bool
	BondSize float64

	Draw     map[int]bool
	DrawSize float64

	Period string

	PlotState int
	Dfact     float64

	PointedLoad bool
	Pfact       float64

	Rfact float64

	Stress        map[int]uint
	NoMomentValue bool
	NoShearValue  bool
	MomentFigure  bool
	ShearArrow    bool
	Mfact         float64
	Qfact         float64

	NoLegend       bool
	LegendPosition []int
	LegendSize     int
	LegendLineSep  float64

	YieldFunction bool

	Fes      bool
	MassSize float64

	Kijun     bool
	KijunSize float64

	Measure bool

	Arc bool

	Select bool

	Sect  map[int]bool
	Etype map[int]bool

	Xrange []float64
	Yrange []float64
	Zrange []float64

	DefaultTextAlignment int
	Formats              map[string]string

	CanvasFontColor  int
	PlateEdgeColor   int
	ConfColor        int
	BondColor        int
	DeformationColor int
	KijunColor       int
	MeasureColor     int
	MomentColor      int
	StressTextColor  int
	YieldedTextColor int
	BrittleTextColor int
}

func NewShow(frame *Frame) *Show {
	sects := make(map[int]bool)
	for snum := range frame.Sects {
		sects[snum] = true
	}
	etypes := make(map[int]bool)
	draws := make(map[int]bool)
	for i, _ := range ETYPES {
		if i == WBRACE || i == SBRACE {
			etypes[i] = false
		} else {
			etypes[i] = true
		}
		draws[i] = false
	}
	formats := make(map[string]string)
	formats["STRESS"] = "%.3f"
	formats["RATE"] = "%.3f"
	formats["DISP"] = "%.3f"
	formats["THETA"] = "%.3e"
	formats["REACTION"] = "%.3f"
	return &Show{
		Frame:                frame,
		Unit:                 []float64{1.0, 1.0},
		UnitName:             []string{"tf", "m"},
		ColorMode:            ECOLOR_SECT,
		NodeCaption:          0,
		ElemCaption:          0,
		SrcanRate:            0,
		Energy:               false,
		GlobalAxis:           true,
		GlobalAxisSize:       1.0,
		ElementAxis:          false,
		ElementAxisSize:      0.5,
		NodeNormal:           false,
		NodeNormalSize:       0.2,
		ElemNormal:           false,
		ElemNormalSize:       0.2,
		CmqLine:              false,
		Conf:                 true,
		ConfSize:             9.0,
		Bond:                 true,
		Phinge:               true,
		BondSize:             3.0,
		Draw:                 draws,
		DrawSize:             1.0,
		Period:               "L",
		PlotState:            PLOT_UNDEFORMED,
		Dfact:                100.0,
		PointedLoad:          false,
		Pfact:                1.0,
		Rfact:                0.5,
		Stress:               map[int]uint{COLUMN: 0, GIRDER: 0, BRACE: 0, WBRACE: 0, SBRACE: 0},
		NoMomentValue:        false,
		NoShearValue:         false,
		MomentFigure:         true,
		ShearArrow:           false,
		Mfact:                0.5,
		Qfact:                0.5,
		NoLegend:             false,
		LegendPosition:       []int{30, 30},
		LegendSize:           10,
		LegendLineSep:        1.3,
		YieldFunction:        false,
		Fes:                  false,
		MassSize:             0.5,
		Kijun:                false,
		KijunSize:            12.0,
		Measure:              true,
		Arc:                  true,
		Select:               false,
		Sect:                 sects,
		Etype:                etypes,
		Xrange:               []float64{RANGE_MIN, RANGE_MAX},
		Yrange:               []float64{RANGE_MIN, RANGE_MAX},
		Zrange:               []float64{RANGE_MIN, RANGE_MAX},
		DefaultTextAlignment: WEST,
		Formats:              formats,
		CanvasFontColor:      GRAY,
		PlateEdgeColor:       GRAY,
		ConfColor:            GRAY,
		BondColor:            GRAY,
		DeformationColor:     GRAY,
		KijunColor:           GRAY,
		MeasureColor:         GRAY,
		MomentColor:          DARK_MAGENTA,
		StressTextColor:      GRAY,
		YieldedTextColor:     YELLOW,
		BrittleTextColor:     RED,
	}
}

func (show *Show) Copy() *Show {
	s := NewShow(show.Frame)
	for i := 0; i < 2; i++ {
		s.Unit[i] = show.Unit[i]
		s.UnitName[i] = show.UnitName[i]
		s.Xrange[i] = show.Xrange[i]
		s.Yrange[i] = show.Yrange[i]
		s.Zrange[i] = show.Zrange[i]
	}
	s.ColorMode = show.ColorMode
	s.NodeCaption = show.NodeCaption
	s.ElemCaption = show.ElemCaption
	s.SrcanRate = show.SrcanRate
	s.GlobalAxis = show.GlobalAxis
	s.GlobalAxisSize = show.GlobalAxisSize
	s.ElementAxis = show.ElementAxis
	s.ElementAxisSize = show.ElementAxisSize
	s.NodeNormal = show.NodeNormal
	s.NodeNormalSize = show.NodeNormalSize
	s.ElemNormal = show.ElemNormal
	s.ElemNormalSize = show.ElemNormalSize
	s.Bond = show.Bond
	s.BondSize = show.BondSize
	s.Conf = show.Conf
	s.ConfSize = show.ConfSize
	s.Period = show.Period
	s.PlotState = show.PlotState
	s.Dfact = 100.0
	s.PointedLoad = false
	s.Pfact = 1.0
	for _, et := range []int{COLUMN, GIRDER, BRACE, WBRACE, SBRACE} {
		s.Stress[et] = show.Stress[et]
	}
	s.Mfact = show.Mfact

	s.Kijun = show.Kijun
	s.KijunSize = show.KijunSize

	s.Arc = show.Arc

	s.Select = show.Select

	s.Sect = make(map[int]bool)
	s.Etype = make(map[int]bool)
	for i, _ := range ETYPES {
		s.Etype[i] = show.Etype[i]
	}
	for k, v := range show.Sect {
		s.Sect[k] = v
	}

	s.Formats["STRESS"] = show.Formats["STRESS"]
	s.Formats["RATE"] = show.Formats["RATE"]
	s.Formats["DISP"] = show.Formats["DISP"]
	s.Formats["REACTION"] = show.Formats["REACTION"]

	return s
}

func (show *Show) All() {
	for i, _ := range ETYPES {
		if i == WBRACE || i == SBRACE {
			continue
		}
		show.Etype[i] = true
	}
	for i, _ := range show.Sect {
		show.Sect[i] = true
	}
	show.Xrange[0] = RANGE_MIN
	show.Xrange[1] = RANGE_MAX
	show.Yrange[0] = RANGE_MIN
	show.Yrange[1] = RANGE_MAX
	show.Zrange[0] = RANGE_MIN
	show.Zrange[1] = RANGE_MAX
}

func (show *Show) NodeCaptionOn(val uint) {
	show.NodeCaption |= val
}
func (show *Show) NodeCaptionOff(val uint) {
	show.NodeCaption &= ^val
}
func (show *Show) ElemCaptionOn(val uint) {
	show.ElemCaption |= val
}
func (show *Show) ElemCaptionOff(val uint) {
	show.ElemCaption &= ^val
}
func (show *Show) SrcanRateOn(val uint) {
	show.SrcanRate |= val
}
func (show *Show) SrcanRateOff(val uint) {
	show.SrcanRate &= ^val
}

func (show *Show) Dataline() []string {
	first := make([]string, 0)
	num1 := 0
	if show.Conf {
		first = append(first, "支点")
		num1++
	}
	if show.PlotState&PLOT_DEFORMED != 0 {
		first = append(first, "変形図")
		num1++
	}
	if show.PointedLoad {
		first = append(first, "節点荷重")
		num1++
	}
	for i, nc := range []uint{NC_NUM, NC_WEIGHT, NC_ZCOORD, NC_DX, NC_DY, NC_DZ, NC_TX, NC_TY, NC_TZ, NC_RX, NC_RY, NC_RZ, NC_MX, NC_MY, NC_MZ, NC_PILE} {
		if show.NodeCaption&nc != 0 {
			if nc == NC_DX || nc == NC_DY || nc == NC_DZ {
				first = append(first, NODECAPTIONNAME[i]+"[cm]")
			} else if nc == NC_RX || nc == NC_RY || nc == NC_RZ {
				first = append(first, NODECAPTIONNAME[i]+fmt.Sprintf("[%s]", show.UnitName[0]))
			} else if nc == NC_MX || nc == NC_MY || nc == NC_MZ {
				first = append(first, NODECAPTIONNAME[i]+fmt.Sprintf("[%s%s]", show.UnitName[0], show.UnitName[1]))
			} else {
				first = append(first, NODECAPTIONNAME[i])
			}
			num1++
		}
	}
	for i, ec := range []uint{EC_NUM, EC_SECT, EC_PREST, EC_STIFF_X, EC_STIFF_Y, EC_DRIFT_X, EC_DRIFT_Y, EC_TENSILE_X, EC_TENSILE_Y, EC_WIDTH, EC_HEIGHT} {
		if show.ElemCaption&ec != 0 {
			if ec == EC_PREST {
				first = append(first, ELEMCAPTIONNAME[i]+fmt.Sprintf("[%s]", show.UnitName[0]))
			} else if ec == EC_STIFF_X || ec == EC_STIFF_Y {
				first = append(first, ELEMCAPTIONNAME[i]+fmt.Sprintf("[%s/%s]", show.UnitName[0], show.UnitName[1]))
			} else if ec == EC_WIDTH || ec == EC_HEIGHT {
				first = append(first, ELEMCAPTIONNAME[i]+fmt.Sprintf("[%s]", show.UnitName[1]))
			} else {
				first = append(first, ELEMCAPTIONNAME[i])
			}
			num1++
		}
	}
	if show.SrcanRate != 0 || show.ColorMode == ECOLOR_RATE {
		first = append(first, "断面検定比")
	}
	second := make([]string, 0)
	num2 := 0
	for i, _ := range ETYPES {
		if i == 0 {
			continue
		}
		if show.Etype[i] {
			second = append(second, ETYPENAME[i])
			num2++
		}
	}
	return []string{strings.Join(first[:num1], ", "), fmt.Sprintf("表示部材 : %s", strings.Join(second[:num2], ", "))}
}
