package st

import (
	"fmt"
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

func (k *Kijun) String() string {
	fmt.Println(k.Name)
	fmt.Println(k.Start)
	fmt.Println(k.End)
	return fmt.Sprintf("%5s %8.3f %8.3f %8.3f %8.3f %8.3f %8.3f\n", k.Name, k.Start[0], k.Start[1], k.Start[2], k.End[0], k.End[1], k.End[2])
}

func (k *Kijun) Hide() {
	k.hide = true
}

func (k *Kijun) Show() {
	k.hide = false
}

func (k *Kijun) IsHidden(show *Show) bool {
	if k.hide {
		return true
	}
	d := k.Direction()
	if math.Abs(d[0]) < 1e-4 {
		if k.Start[0] < show.Xrange[0] || show.Xrange[1] < k.Start[0] {
			return true
		}
		if k.End[0] < show.Xrange[0] || show.Xrange[1] < k.End[0] {
			return true
		}
	}
	if math.Abs(d[1]) < 1e-4 {
		if k.Start[1] < show.Yrange[0] || show.Yrange[1] < k.Start[1] {
			return true
		}
		if k.End[1] < show.Yrange[0] || show.Yrange[1] < k.End[1] {
			return true
		}
	}
	if math.Abs(d[2]) < 1e-4 {
		if k.Start[2] < show.Zrange[0] || show.Zrange[1] < k.Start[2] {
			return true
		}
		if k.End[2] < show.Zrange[0] || show.Zrange[1] < k.End[2] {
			return true
		}
	}
	return false
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
