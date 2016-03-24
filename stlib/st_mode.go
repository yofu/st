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

type ExModer interface {
	Selector

	// UndoStack
	UseUndo(bool)
	// TagFrame
	Checkout(string) (*Frame, error)
	AddTag(*Frame, string, bool) error

	LastExCommand() string
	SetLastExCommand(string)
	Print()
	SaveAS()
	SaveFileSelected(string) error
	SearchFile(string) (string, error)
	ShapeData(Shape)
	ToggleFixRotate()
	ToggleFixMove()
	SetShowPrintRange(bool)
	ToggleShowPrintRange()
	CurrentLap(string, int, int)
	SectionData(*Sect)
	TextBox(string) TextBox
	SetAngle(float64, float64)
	SetPaperSize(uint)
	PaperSize() uint
	Pivot() bool
	DrawPivot([]*Node, chan int, chan int)
	SetColorMode(uint)
	SetConf([]bool)
}

type Fig2Moder interface {
	Selector

	// Alias
	AddSectionAlias(int, string)
	DeleteSectionAlias(int)
	ClearSectionAlias()

	LastFig2Command() string
	SetLastFig2Command(string)
	ShowCenter()
	SetColorMode(uint)
	TextBox(string) TextBox
}
