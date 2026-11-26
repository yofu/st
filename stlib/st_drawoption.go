package st

import "sort"

type ElemPen struct {
	Sect  int
	Etype int
}

// ElemPenProvider is optionally implemented by drivers (via embedded *DrawOption)
// to supply default section and ETYPE for interactive member creation commands.
type ElemPenProvider interface {
	LinePen() ElemPen
	SetLinePen(sect, etype int)
	PlatePen() ElemPen
	SetPlatePen(sect, etype int)
}

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
	linePen      ElemPen
	platePen     ElemPen
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

func (d *DrawOption) LinePen() ElemPen {
	return d.linePen
}

func (d *DrawOption) SetLinePen(sect, etype int) {
	d.linePen = ElemPen{Sect: sect, Etype: etype}
}

func (d *DrawOption) PlatePen() ElemPen {
	return d.platePen
}

func (d *DrawOption) SetPlatePen(sect, etype int) {
	d.platePen = ElemPen{Sect: sect, Etype: etype}
}

func (d *DrawOption) ResetElemPens() {
	d.linePen = ElemPen{}
	d.platePen = ElemPen{}
}

// InferEtypeFromSect infers a default ETYPE for sectNum. It first checks the
// most common valid ETYPE already assigned to sectNum in frame, then falls back
// to ST's section numbering bands (100s=COLUMN, 200-400s=GIRDER, 500-600s=BRACE,
// 700s=WALL, 800s=SLAB).
func InferEtypeFromSect(frame *Frame, sectNum int, isPlate bool) int {
	if frame != nil && sectNum > 0 {
		counts := map[int]int{}
		for _, el := range frame.Elems {
			if el == nil || el.Sect == nil || el.Sect.Num != sectNum {
				continue
			}
			if isPlate {
				if !el.IsLineElem() && (el.Etype == WALL || el.Etype == SLAB) {
					counts[el.Etype]++
				}
			} else {
				if el.IsLineElem() && (el.Etype == COLUMN || el.Etype == GIRDER || el.Etype == BRACE) {
					counts[el.Etype]++
				}
			}
		}
		bestEtype, bestCount := NULL, 0
		for _, cand := range []int{COLUMN, GIRDER, BRACE, WALL, SLAB} {
			if counts[cand] > bestCount {
				bestEtype, bestCount = cand, counts[cand]
			}
		}
		if bestCount > 0 {
			return bestEtype
		}
	}
	if isPlate {
		switch {
		case sectNum >= 700 && sectNum <= 799:
			return WALL
		case sectNum >= 800 && sectNum <= 899:
			return SLAB
		default:
			return NULL
		}
	}
	switch {
	case sectNum >= 100 && sectNum <= 199:
		return COLUMN
	case sectNum >= 200 && sectNum <= 499:
		return GIRDER
	case sectNum >= 500 && sectNum <= 699:
		return BRACE
	default:
		return NULL
	}
}

func firstSectInRange(frame *Frame, minNum, maxNum int) *Sect {
	if frame == nil || len(frame.Sects) == 0 {
		return nil
	}
	nums := make([]int, 0, len(frame.Sects))
	for num := range frame.Sects {
		if num >= minNum && num <= maxNum {
			nums = append(nums, num)
		}
	}
	if len(nums) == 0 {
		return nil
	}
	sort.Ints(nums)
	return frame.Sects[nums[0]]
}

func ResolveLinePen(stw any, frame *Frame) (*Sect, int) {
	var pen ElemPen
	if p, ok := stw.(ElemPenProvider); ok && p != nil {
		pen = p.LinePen()
	}
	var sec *Sect
	if frame != nil && pen.Sect > 0 {
		sec = frame.Sects[pen.Sect]
	}
	if sec == nil && frame != nil {
		sec = firstSectInRange(frame, 100, 699)
		if sec == nil {
			sec = frame.DefaultSect()
		}
	}
	etype := NULL
	switch pen.Etype {
	case NULL, COLUMN, GIRDER, BRACE:
		etype = pen.Etype
	}
	return sec, etype
}

func ResolvePlatePen(stw any, frame *Frame) (*Sect, int) {
	var pen ElemPen
	if p, ok := stw.(ElemPenProvider); ok && p != nil {
		pen = p.PlatePen()
	}
	var sec *Sect
	if frame != nil && pen.Sect > 0 {
		sec = frame.Sects[pen.Sect]
	}
	if sec == nil && frame != nil {
		sec = firstSectInRange(frame, 700, 899)
		if sec == nil {
			sec = frame.DefaultSect()
		}
	}
	etype := NULL
	switch pen.Etype {
	case NULL, WALL, SLAB:
		etype = pen.Etype
	}
	return sec, etype
}
