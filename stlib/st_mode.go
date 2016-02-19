package st

// Log Level
var (
	LOGLEVEL = []string{"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"}
)

const (
	DEBUG = iota
	INFO
	WARNING
	ERROR
	CRITICAL
)

// Paper Size
const (
	A4_TATE = iota
	A4_YOKO
	A3_TATE
	A3_YOKO
)

type ExModer interface {
	Selector

	// UndoStack
	UseUndo(bool)
	// TagFrame
	Checkout(string) (*Frame, error)
	AddTag(*Frame, string, bool) error

	LastExCommand() string
	SetLastExCommand(string)
	CompleteFileName(string) string
	Print()
	Yn(string, string) bool
	Yna(string, string, string) int
	SaveAS()
	SaveFileSelected(string) error
	SearchFile(string) (string, error)
	Close(bool)
	Copylsts(string)
	ReadPgp(string) error
	ReadFig2(string) error
	CheckFrame()
	ShapeData(Shape)
	ToggleFixRotate()
	ToggleFixMove()
	ToggleAltSelectNode()
	AltSelectNode() bool
	SetShowPrintRange(bool)
	ToggleShowPrintRange()
	CurrentLap(string, int, int)
	SectionData(*Sect)
	TextBox(string) TextBox
	AxisRange(int, float64, float64, bool)
	NextFloor()
	PrevFloor()
	SetAngle(float64, float64)
	SetPaperSize(uint)
	PaperSize() uint
	SetPeriod(string)
	Pivot() bool
	DrawPivot([]*Node, chan int, chan int)
	SetColorMode(uint)
	SetConf([]bool)
}

type Fig2Moder interface {
	Window

	LastFig2Command() string
	SetLastFig2Command(string)
	SetLabel(string, string)
	DisableLabel(string)
	EnableLabel(string)
	ShowCenter()
	ShowEtype(int)
	HideEtype(int)
	ShowSection(int)
	HideSection(int)
	HideAllSection()
	HideNotSelected()
	ElemCaptionOn(string)
	ElemCaptionOff(string)
	NodeCaptionOn(string)
	NodeCaptionOff(string)
	SetColorMode(uint)
	AxisRange(int, float64, float64, bool)
	SrcanRateOn(...string)
	SrcanRateOff(...string)
	StressOn(int, uint)
	StressOff(int, uint)
	DeformationOn()
	DeformationOff()
	DispOn(int)
	DispOff(int)
	SetPeriod(string)
	AddSectionAlias(int, string)
	DeleteSectionAlias(int)
	ClearSectionAlias()
	SelectElem([]*Elem)
	IncrementPeriod(int)
	TextBox(string) TextBox
}
