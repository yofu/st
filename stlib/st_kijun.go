package st

import (
	"math"
)

type Kijun struct {
	Name   string
	Start  []float64
	End    []float64
	Pstart []float64
	Pend   []float64

	hide bool
}

func NewKijun() *Kijun {
	k := new(Kijun)
	k.Start = make([]float64, 3)
	k.End = make([]float64, 3)
	k.Pstart = make([]float64, 2)
	k.Pend = make([]float64, 2)
	return k
}

func (k *Kijun) Snapshot() *Kijun {
	rtn := NewKijun()
	rtn.Name = k.Name
	for i := 0; i < 3; i++ {
		rtn.Start[i] = k.Start[i]
		rtn.End[i] = k.End[i]
	}
	for i := 0; i < 2; i++ {
		rtn.Pstart[i] = k.Pstart[i]
		rtn.Pend[i] = k.Pend[i]
	}
	rtn.hide = k.hide
	return rtn
}

func (k *Kijun) Hide() {
	k.hide = true
}

func (k *Kijun) Show() {
	k.hide = false
}

func (k *Kijun) IsHidden(show *Show) bool {
	return k.hide
}

func (k *Kijun) Length() float64 {
	sum := 0.0
	for i := 0; i < 3; i++ {
		sum += math.Pow((k.End[i] - k.Start[i]), 2)
	}
	return math.Sqrt(sum)
}

func (k *Kijun) Direction() []float64 {
	vec := make([]float64, 3)
	var l float64
	l = k.Length()
	for i := 0; i < 3; i++ {
		vec[i] = (k.End[i] - k.Start[i]) / l
	}
	return vec
}

func (k *Kijun) PLength() float64 {
	sum := 0.0
	for i := 0; i < 2; i++ {
		sum += math.Pow((k.Pend[i] - k.Pstart[i]), 2)
	}
	return math.Sqrt(sum)
}

func (k *Kijun) PDirection(normalize bool) []float64 {
	vec := make([]float64, 2)
	var l float64
	if normalize {
		l = k.PLength()
	} else {
		l = 1.0
	}
	for i := 0; i < 2; i++ {
		vec[i] = (k.Pend[i] - k.Pstart[i]) / l
	}
	return vec
}

func (k *Kijun) Distance(k2 *Kijun) float64 {
	d := k.Direction()
	rtn := 0.0
	for i := 0; i < 3; i++ {
		val := k2.Start[i] - k.Start[i]
		val2 := val * val
		rtn += val2 - val2*d[i]*d[i]
	}
	return math.Sqrt(rtn)
}
