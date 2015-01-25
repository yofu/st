package st

type Measure struct {
	Frame     *Frame
	Start     []float64
	End       []float64
	Direction []float64
	Extension float64
	Gap       float64
	ArrowSize float64
	Rotate    float64
	Text      string
	Hide      bool
}

func NewMeasure(start, end, direction []float64) *Measure {
	m := new(Measure)
	m.Start     = start
	m.End       = end
	m.Direction = direction
	m.Extension = 1.0
	m.Gap       = 0.0
	m.ArrowSize = 6.0
	m.Text      = ""
	m.Hide      = false
	return m
}
