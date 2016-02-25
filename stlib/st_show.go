package st

var (
	NODECAPTIONS = []string{"NC_NUM", "NC_WEIGHT", "NC_ZCOORD", "NC_DX", "NC_DY", "NC_DZ", "NC_TX", "NC_TY", "NC_TZ", "NC_RX", "NC_RY", "NC_RZ", "NC_MX", "NC_MY", "NC_MZ", "NC_PILE"}
	ELEMCAPTIONS = []string{"EC_NUM", "EC_SECT", "EC_RATE_L", "EC_RATE_S", "EC_PREST", "EC_STIFF_X", "EC_STIFF_Y", "EC_DRIFT_X", "EC_DRIFT_Y", "EC_WIDTH", "EC_HEIGHT"}
	SRCANS       = []string{"SRCAN_L", "SRCAN_S", "SRCAN_Q", "SRCAN_M"}
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

type Show struct { // {{{
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

	PlateEdgeColor int

	Conf      bool
	ConfSize  float64
	ConfColor int

	Bond      bool
	BondSize  float64
	BondColor int
	Phinge    bool

	DeformationColor int
	KijunColor       int
	MeasureColor     int
	CanvasFontColor  int

	Draw     map[int]bool
	DrawSize float64

	Period string

	Deformation bool
	Dfact       float64

	Rfact float64

	Stress        map[int]uint
	NoMomentValue bool
	NoShearValue  bool
	MomentFigure  bool
	ShearArrow    bool

	NoLegend       bool
	LegendPosition []int
	LegendSize     int
	LegendLineSep  float64

	YieldFunction bool

	Mfact            float64
	Qfact            float64
	MomentColor      int
	StressTextColor  int
	YieldedTextColor int
	BrittleTextColor int

	DefaultTextAlignment int

	Fes      bool
	MassSize float64

	Kijun     bool
	KijunSize float64

	Measure bool

	Select bool

	Sect  map[int]bool
	Etype map[int]bool

	Xrange []float64
	Yrange []float64
	Zrange []float64

	Formats map[string]string
}

func NewShow(frame *Frame) *Show {
	s := new(Show)

	s.Frame = frame

	s.Unit = []float64{1.0, 1.0}
	s.UnitName = []string{"tf", "m"}

	s.ColorMode = ECOLOR_SECT

	s.NodeCaption = NC_NUM
	s.ElemCaption = 0
	s.SrcanRate = 0

	s.GlobalAxis = true
	s.GlobalAxisSize = 1.0
	s.ElementAxis = false
	s.ElementAxisSize = 0.5

	s.NodeNormal = false
	s.NodeNormalSize = 0.2
	s.ElemNormal = false
	s.ElemNormalSize = 0.2

	s.Bond = true
	s.Phinge = true
	s.BondSize = 3.0
	s.Conf = true
	s.ConfSize = 9.0

	s.Period = "L"

	s.Deformation = false
	s.Dfact = 100.0

	s.Rfact = 0.5

	s.Stress = map[int]uint{COLUMN: 0, GIRDER: 0, BRACE: 0, WBRACE: 0, SBRACE: 0}
	s.NoMomentValue = false
	s.NoShearValue = false
	s.MomentFigure = true
	s.ShearArrow = false
	s.Mfact = 0.5
	s.Qfact = 0.5

	s.NoLegend = false
	s.LegendPosition = []int{30, 30}
	s.LegendSize = 10
	s.LegendLineSep = 1.3

	s.Fes = false
	s.MassSize = 0.5

	s.Kijun = false
	s.KijunSize = 12.0

	s.Measure = true

	s.Select = false

	s.Sect = make(map[int]bool)
	for snum := range frame.Sects {
		s.Sect[snum] = true
	}
	s.Etype = make(map[int]bool)
	s.Draw = make(map[int]bool)
	s.DrawSize = 1.0
	for i, _ := range ETYPES {
		if i == WBRACE || i == SBRACE {
			s.Etype[i] = false
		} else {
			s.Etype[i] = true
		}
		s.Draw[i] = false
	}

	s.Xrange = []float64{-100.0, 1000.0}
	s.Yrange = []float64{-100.0, 1000.0}
	s.Zrange = []float64{-100.0, 1000.0}

	s.Formats = make(map[string]string)

	s.Formats["STRESS"] = "%.3f"
	s.Formats["RATE"] = "%.3f"
	s.Formats["DISP"] = "%.3f"
	s.Formats["THETA"] = "%.3e"
	s.Formats["REACTION"] = "%.3f"

	s.CanvasFontColor = GRAY
	s.PlateEdgeColor = GRAY
	s.ConfColor = GRAY
	s.BondColor = GRAY
	s.DeformationColor = GRAY
	s.KijunColor = GRAY
	s.MeasureColor = GRAY
	s.MomentColor = DARK_MAGENTA
	s.StressTextColor = GRAY
	s.YieldedTextColor = YELLOW
	s.BrittleTextColor = RED

	return s
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
	s.Deformation = false
	s.Dfact = 100.0
	for _, et := range []int{COLUMN, GIRDER, BRACE, WBRACE, SBRACE} {
		s.Stress[et] = show.Stress[et]
	}
	s.Mfact = show.Mfact

	s.Kijun = show.Kijun
	s.KijunSize = show.KijunSize

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
	show.Xrange[0] = -100.0
	show.Xrange[1] = 1000.0
	show.Yrange[0] = -100.0
	show.Yrange[1] = 1000.0
	show.Zrange[0] = -100.0
	show.Zrange[1] = 1000.0
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

// }}}
