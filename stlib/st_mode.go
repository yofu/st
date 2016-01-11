package st

// Paper Size
const (
	A4_TATE = iota
	A4_YOKO
	A3_TATE
	A3_YOKO
)

type ExModer interface {
	LastExCommand() string
	SetLastExCommand(string)
	History(string)
	ErrorMessage(error)
	CompleteFileName(string) string
	Cwd() string
	HomeDir() string
	Print()
	PrintSVG(string) error
	IsChanged() bool
	Yn(string, string) bool
	Yna(string, string, string) int
	SaveAS()
	SaveFile(string) error
	SaveFileSelected(string) error
	SearchFile(string) (string, error)
	OpenFile(string, bool) error
	Reload()
	Close(bool)
	Checkout(string) error
	AddTag(string, bool) error
	Copylsts(string)
	ReadFile(string) error
	ReadAll()
	ReadPgp(string) error
	ReadFig2(string) error
	CheckFrame()
	SelectElem([]*Elem)
	SelectNode([]*Node)
	ElemSelected() bool
	NodeSelected() bool
	SelectedElems() []*Elem
	SelectedNodes() []*Node
	SelectConfed()
	Deselect()
	Rebase(string)
	ShowRecently()
	ShapeData(Shape)
	Snapshot()
	UseUndo(bool)
	EPS() float64
	SetEPS(float64)
	CanvasFitScale() float64
	SetCanvasFitScale(float64)
	CanvasAnimateSpeed() float64
	SetCanvasAnimateSpeed(float64)
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
	Redraw()
	SetConf([]bool)
}

type Fig2Moder interface {
	LastFig2Command() string
	SetLastFig2Command(string)
	History(string)
	ErrorMessage(error)
	SetLabel(string, string)
	DisableLabel(string)
	EnableLabel(string)
	GetCanvasSize() (int, int)
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
	EPS() float64
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
	AddSectionAliase(int, string)
	DeleteSectionAliase(int)
	ClearSectionAliase()
	SelectElem([]*Elem)
	IncrementPeriod(int)
	TextBox(string) TextBox
}
