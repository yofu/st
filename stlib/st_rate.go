package st

import (
    "errors"
    "fmt"
    "math"
)

const (
    E_LAMBDA_B = 1.29099444874 // 1.0/math.Sqrt(6)
    P_LAMBDA_B = 0.3
)
// Material
type Material interface {
}

type Steel struct {
    Name string
    F    float64
    Fu   float64
    E    float64
    Poi  float64
}

type Concrete struct {
    Name string
    Fc   float64
    E    float64
    Poi  float64
}

type Reinforce struct {
    Name string
    Fy   float64
    Fu   float64
    E    float64
    Poi  float64
}

type Wood struct {
    Name float64
    Fc   float64
    Ft   float64
    Fb   float64
    Fs   float64
    E    float64
    Poi  float64
}

var (
    SN400 = Steel{"SN400", 2.4, 4.0, 2100.0, 0.3}
    SN490 = Steel{"SN490", 3.3, 5.0, 2100.0, 0.3}

    FC24 = Concrete{"Fc24", 0.240, 210.0, 0.166666}
    FC36 = Concrete{"Fc36", 0.360, 210.0, 0.166666}

    SD295 = Reinforce{"SD295", 2.0, 3.0, 2100.0, 0.3}
    SD345 = Reinforce{"SD345", 2.2, 3.5, 2100.0, 0.3}
)

// Section
type SectionRate interface {
    String() string
    Factor() float64
    Na(*Condition)  float64
    Qa(*Condition) float64
    Ma(*Condition) float64
    Mza(*Condition) float64
    Rate([]float64, *Condition) []float64
}

type Shape interface {
    String() string
    Description() string
    A()  float64
    Asx() float64
    Asy() float64
    Ix() float64
    Iy() float64
    J()  float64
    Iw() float64
    Torsion() float64
    Zx() float64
    Zy() float64
}


// S COLUMN
type SColumn struct {
    Steel
    Shape
    BBLength []float64
    BTLength []float64
    BBFactor []float64
    BTFactor []float64
}
func (sc *SColumn) String() string {
    return fmt.Sprintf("S COLUMN\n%s %s %s", sc.Shape.String(), sc.Steel.Name, sc.Shape.Description())
}
func (sc *SColumn) Factor(p string) float64 {
    switch p {
    default:
        return 1e16
    case "L":
        return 1.5
    case "X", "Y", "S":
        return 1.0
    }
}
func (sc *SColumn) Na(cond *Condition) float64 {
    if cond.Compression {
        return 0.0
    } else {
        return sc.F*sc.A()/sc.Factor(cond.Period)
    }
}
func (sc *SColumn) Qa(cond *Condition) float64 {
    if cond.Strong {
        return sc.F/math.Sqrt(3)*sc.Asx()/sc.Factor(cond.Period)
    } else {
        return sc.F/math.Sqrt(3)*sc.Asy()/sc.Factor(cond.Period)
    }
}
func (sc *SColumn) Lk (length float64, strong bool) float64 {
    var ind int
    if strong { ind = 0 } else { ind = 1 }
    if sc.BBLength[ind] > 0.0 {
        return sc.BBLength[ind]
    } else if sc.BBFactor[ind] > 0.0 {
        return length * sc.BBFactor[ind]
    } else {
        return length
    }
}
func (sc *SColumn) Lb (length float64, strong bool) float64 {
    var ind int
    if strong { ind = 0 } else { ind = 1 }
    if sc.BTLength[ind] > 0.0 {
        return sc.BTLength[ind]
    } else if sc.BTFactor[ind] > 0.0 {
        return length * sc.BTFactor[ind]
    } else {
        return length
    }
}
func (sc *SColumn) Fb (cond *Condition) float64 {
    l := sc.Lb(cond.Length, cond.Strong)
    fbnew := func () float64 {
                 me := sc.Me(l, 1.0, cond)
                 my := sc.My(cond)
                 lambda_b := math.Sqrt(my/me)
                 nu := 1.5 + math.Pow(lambda_b/E_LAMBDA_B, 2.0)/1.5
                 if lambda_b <= P_LAMBDA_B {
                     return sc.F/nu/sc.Factor(cond.Period)
                 } else if lambda_b <= E_LAMBDA_B {
                     return (1.0-0.4*(lambda_b-P_LAMBDA_B)/(E_LAMBDA_B-P_LAMBDA_B))*sc.F/nu/sc.Factor(cond.Period)
                 } else {
                     return sc.F/math.Pow(lambda_b, 2.0)/2.17/sc.Factor(cond.Period)
                 }
             }
    if cond.Strong {
        if hk, ok := sc.Shape.(HKYOU); ok {
            if cond.FbOld {
                return 900.0/(l*hk.H/(hk.B*hk.Tf))
            } else {
                return fbnew()
            }
        } else {
            return sc.F
        }
    } else {
        if hw, ok := sc.Shape.(HWEAK); ok {
            if cond.FbOld {
                return 900.0/(l*hw.H/(hw.B*hw.Tf))
            } else {
                return fbnew()
            }
        } else {
            return sc.F
        }
    }
}
func (sc *SColumn) Me(length, Cb float64, cond *Condition) float64 {
    g := sc.E/(2.0*(1+sc.Poi))
    var I float64
    Ix := sc.Ix(); Iy := sc.Iy()
    if Ix >= Iy { I = Iy } else { I = Ix }
    return Cb*math.Sqrt(math.Pow(math.Pi,4.0)*sc.E*I*sc.E*sc.Iw()/math.Pow(length, 4.0) + math.Pow(math.Pi, 2.0)*sc.E*I*g*sc.J()/math.Pow(length, 2.0))*0.01 // [tfm]
}
func (sc *SColumn) My(cond *Condition) float64 {
    if cond.Strong {
        return sc.F*sc.Zx()/sc.Factor(cond.Period)*0.01 // [tfm]
    } else {
        return sc.F*sc.Zy()/sc.Factor(cond.Period)*0.01 // [tfm]
    }
}
func (sc *SColumn) Ma(cond *Condition) float64 {
    if cond.Strong {
        return sc.Fb(cond)*sc.Zx()/sc.Factor(cond.Period)*0.01 // [tfm]
    } else {
        return sc.Fb(cond)*sc.Zy()/sc.Factor(cond.Period)*0.01 // [tfm]
    }
}
func (sc *SColumn) Mza(cond *Condition) float64 {
    return sc.F/math.Sqrt(3)*sc.Torsion()/sc.Factor(cond.Period)*0.01 // [tfm]
}
func (sc *SColumn) Rate(stress []float64, cond *Condition) ([]float64, error) {
    if len(stress) < 6 { return nil, errors.New("Rate: Not enough number of Stress") }
    rate := make([]float64, 6)
    cond.Rate = rate
    na := sc.Na(cond)
    if na != 0.0 { rate[0] = stress[0]/na } else { rate[0] = -1.0 }
    cond.Strong = true
    qax := sc.Qa(cond)
    if qax != 0.0 { rate[1] = stress[1]/qax } else { rate[1] = -1.0 }
    max := sc.Ma(cond)
    if max != 0.0 { rate[4] = stress[4]/max } else { rate[4] = -1.0 }
    cond.Strong = false
    qay := sc.Qa(cond)
    if qay != 0.0 { rate[2] = stress[2]/qay } else { rate[2] = -1.0 }
    may := sc.Ma(cond)
    if may != 0.0 { rate[5] = stress[5]/may } else { rate[5] = -1.0 }
    maz := sc.Mza(cond)
    if maz != 0.0 { rate[3] = stress[3]/maz } else { rate[3] = -1.0 }
    return rate, nil
}


// HKYOU// {{{
type HKYOU struct {
    H, B, Tw, Tf float64
}
func (hk HKYOU) String() string {
    return fmt.Sprintf("HKYOU %.1f %.1f %.1f %.1f", hk.H, hk.B, hk.Tw, hk.Tf)
}
func (hk HKYOU) Description() string {
    return fmt.Sprintf("H-%dx%dx%dx%d[mm]", int(hk.H*10), int(hk.B*10), int(hk.Tw*10), int(hk.Tf*10))
}
func (hk HKYOU) A() float64 {
    return hk.H*hk.B - (hk.H-2*hk.Tf)*(hk.B-hk.Tw)
}
func (hk HKYOU) Asx() float64 {
    return 2.0*hk.B*hk.Tf
}
func (hk HKYOU) Asy() float64 {
    return (hk.H-2*hk.Tf)*hk.Tw
}
func (hk HKYOU) Ix() float64 {
    return (hk.B*math.Pow(hk.H, 3.0) - (hk.B-hk.Tw)*math.Pow(hk.H-2*hk.Tf, 3.0))/12.0
}
func (hk HKYOU) Iy() float64 {
    return 2.0*hk.Tf*math.Pow(hk.B, 3.0)/12.0 + (hk.H-2*hk.Tf)*math.Pow(hk.Tw, 3.0)/12.0
}
func (hk HKYOU) J() float64 {
    return 2.0*hk.B*math.Pow(hk.Tf, 3.0)/3.0 + (hk.H-2*hk.Tf)*math.Pow(hk.Tw, 3.0)/3.0
}
func (hk HKYOU) Iw() float64 {
    return 0.0
}
func (hk HKYOU) Torsion() float64 {
    if hk.Tf >= hk.Tw {
        return hk.J()/hk.Tf
    } else {
        return hk.J()/hk.Tw
    }
}
func (hk HKYOU) Zx() float64 {
    return hk.Ix()/hk.H*2.0
}
func (hk HKYOU) Zy() float64 {
    return hk.Iy()/hk.B*2.0
}
// }}}


// HWEAK// {{{
type HWEAK struct {
    H, B, Tw, Tf float64
}
func (hw HWEAK) String() string {
    return fmt.Sprintf("HWEAK %.1f %.1f %.1f %.1f", hw.H, hw.B, hw.Tw, hw.Tf)
}
func (hw HWEAK) Description() string {
    return fmt.Sprintf("H-%dx%dx%dx%d[mm]", int(hw.H*10), int(hw.B*10), int(hw.Tw*10), int(hw.Tf*10))
}
func (hw HWEAK) A() float64 {
    return hw.H*hw.B - (hw.H-2*hw.Tf)*(hw.B-hw.Tw)
}
func (hw HWEAK) Asx() float64 {
    return (hw.H-2*hw.Tf)*hw.Tw
}
func (hw HWEAK) Asy() float64 {
    return 2.0*hw.B*hw.Tf
}
func (hw HWEAK) Ix() float64 {
    return 2.0*hw.Tf*math.Pow(hw.B, 3.0)/12.0 + (hw.H-2*hw.Tf)*math.Pow(hw.Tw, 3.0)/12.0
}
func (hw HWEAK) Iy() float64 {
    return (hw.B*math.Pow(hw.H, 3.0) - (hw.B-hw.Tw)*math.Pow(hw.H-2*hw.Tf, 3.0))/12.0
}
func (hw HWEAK) J() float64 {
    return 2.0*hw.B*math.Pow(hw.Tf, 3.0)/3.0 + (hw.H-2*hw.Tf)*math.Pow(hw.Tw, 3.0)/3.0
}
func (hk HWEAK) Iw() float64 {
    return 0.0
}
func (hw HWEAK) Torsion() float64 {
    if hw.Tf >= hw.Tw {
        return hw.J()/hw.Tf
    } else {
        return hw.J()/hw.Tw
    }
}
func (hw HWEAK) Zx() float64 {
    return hw.Ix()/hw.B*2.0
}
func (hw HWEAK) Zy() float64 {
    return hw.Iy()/hw.H*2.0
}
// }}}


// RPIPE
type RPIPE struct {
    H, B, Tw, Tf float64
}


// CPIPE
type CPIPE struct {
    D, T float64
}


type SGirder struct {
    SColumn
}


type RCColumn struct {
    Concrete
}

type RCGirder struct {
    Concrete
}

type RCWall struct {
    Concrete
}


// Condition
type Condition struct {
    Period string
    Length float64
    Compression bool
    Strong bool
    FbOld bool
    Rate []float64
}
