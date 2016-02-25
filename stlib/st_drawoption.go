package st

type DrawOption struct {
	rotateSpeedX float64
	rotateSpeedY float64
	moveSpeedX   float64
	moveSpeedY   float64
	scaleSpeed   float64
	fitScale     float64
	animateSpeed float64
	linedash     []int
	dashlength   int
}

func NewDrawOption() *DrawOption {
	return &DrawOption{
		rotateSpeedX: 0.01,
		rotateSpeedY: 0.01,
		moveSpeedX:   0.05,
		moveSpeedY:   0.05,
		scaleSpeed:   15.0,
		fitScale:     0.9,
		animateSpeed: 0.02,
		linedash:     []int{1},
		dashlength:   1,
	}
}

func (d *DrawOption) CanvasRotateSpeedX() float64 {
	return d.rotateSpeedX
}

func (d *DrawOption) SetCanvasRotateSpeedX(val float64) {
	d.rotateSpeedX = val
}

func (d *DrawOption) CanvasRotateSpeedY() float64 {
	return d.rotateSpeedY
}

func (d *DrawOption) SetCanvasRotateSpeedY(val float64) {
	d.rotateSpeedY = val
}

func (d *DrawOption) CanvasMoveSpeedX() float64 {
	return d.moveSpeedX
}

func (d *DrawOption) SetCanvasMoveSpeedX(val float64) {
	d.moveSpeedX = val
}

func (d *DrawOption) CanvasMoveSpeedY() float64 {
	return d.moveSpeedY
}

func (d *DrawOption) SetCanvasMoveSpeedY(val float64) {
	d.moveSpeedY = val
}

func (d *DrawOption) CanvasScaleSpeed() float64 {
	return d.scaleSpeed
}

func (d *DrawOption) SetCanvasScaleSpeed(val float64) {
	d.scaleSpeed = val
}

func (d *DrawOption) CanvasFitScale() float64 {
	return d.fitScale
}

func (d *DrawOption) SetCanvasFitScale(val float64) {
	d.fitScale = val
}

func (d *DrawOption) CanvasAnimateSpeed() float64 {
	return d.animateSpeed
}

func (d *DrawOption) SetCanvasAnimateSpeed(val float64) {
	d.animateSpeed = val
}

func (d *DrawOption) LineDash() []int {
	return d.linedash
}

func (d *DrawOption) SetLineDash(lis []int) {
	d.linedash = lis
	sum := 0
	for _, l := range lis {
		sum += l
	}
	d.dashlength = sum
}

func (d *DrawOption) LineDashProperty() ([]int, int) {
	return d.linedash, d.dashlength
}

func (d *DrawOption) LineStyle(ls int) {
	switch ls {
	case CONTINUOUS:
		d.SetLineDash([]int{1})
	case DOTTED:
		d.SetLineDash([]int{3, 3})
	case DASHED:
		d.SetLineDash([]int{10, 5})
	case DASH_DOT:
		d.SetLineDash([]int{10, 5, 2, 5})
	}
}
