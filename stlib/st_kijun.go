package st

import (
    "github.com/visualfc/go-iup/cd"
    "math"
)

type Kijun struct {
    Name string
    Start  []float64
    End    []float64
    Pstart []float64
    Pend   []float64

    Hide bool
}

func NewKijun () *Kijun {
    k := new(Kijun)
    k.Start  = make([]float64, 3)
    k.End    = make([]float64, 3)
    k.Pstart = make([]float64, 2)
    k.Pend   = make([]float64, 2)
    return k
}

func (k *Kijun) Length() float64 {
    sum := 0.0
    for i:=0; i<3; i++ {
        sum += math.Pow((k.End[i]-k.Start[i]),2)
    }
    return math.Sqrt(sum)
}

func (k *Kijun) Direction() []float64 {
    vec := make([]float64,3)
    var l float64
    l = k.Length()
    for i:=0; i<3; i++ {
        vec[i] = (k.End[i]-k.Start[i])/l
    }
    return vec
}

func (k *Kijun) PLength() float64 {
    sum := 0.0
    for i:=0; i<2; i++ {
        sum += math.Pow((k.Pend[i]-k.Pstart[i]),2)
    }
    return math.Sqrt(sum)
}

func (k *Kijun) PDirection(normalize bool) []float64 {
    vec := make([]float64,2)
    var l float64
    if normalize {
        l = k.PLength()
    } else {
        l = 1.0
    }
    for i:=0; i<2; i++ {
        vec[i] = (k.Pend[i]-k.Pstart[i])/l
    }
    return vec
}

func (k *Kijun) Draw (cvs *cd.Canvas, show *Show) {
    d := k.PDirection(true)
    if (math.Abs(d[0]) <= 1e-6 && math.Abs(d[1]) <= 1e-6) { return }
    cvs.LineStyle(cd.CD_DASH_DOT)
    cvs.FLine(k.Pstart[0], k.Pstart[1], k.Pend[0], k.Pend[1])
    cvs.LineStyle(cd.CD_CONTINUOUS)
    cvs.FCircle(k.Pstart[0]-d[0]*show.KijunSize, k.Pstart[1]-d[1]*show.KijunSize, show.KijunSize*2)
    if k.Name[0] == '_' {
        cvs.FText(k.Pstart[0]-d[0]*show.KijunSize, k.Pstart[1]-d[1]*show.KijunSize, k.Name[1:])
    } else {
        cvs.FText(k.Pstart[0]-d[0]*show.KijunSize, k.Pstart[1]-d[1]*show.KijunSize, k.Name)
    }
}
