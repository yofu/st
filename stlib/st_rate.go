package st

// Material
type Material interface {
}

type Steel struct {
    name string
    Fy   float64
    Fu   float64
    E    float64
    poi  float64
}

type Concrete struct {
    name string
    Fc   float64
    E    float64
    poi  float64
}

type Reinforce struct {
    name string
    Fy   float64
    Fu   float64
    E    float64
    poi  float64
}

type Wood struct {
    name float64
    Fc   float64
    Ft   float64
    Fb   float64
    Fs   float64
    E    float64
    poi  float64
}

var (
    SN400B = &Steel{"SN400B", 2.4, 4.0, 2100.0, 0.3}
    SN490B = &Steel{"SN490B", 3.3, 5.0, 2100.0, 0.3}

    FC24 = &Concrete{"Fc24", 0.240, 210.0, 0.166666}
    FC36 = &Concrete{"Fc36", 0.360, 210.0, 0.166666}

    SD295 = &Reinforce{"SD295", 2.0, 3.0, 2100.0, 0.3}
    SD345 = &Reinforce{"SD345", 2.2, 3.5, 2100.0, 0.3}
)

// Section
type SectionRate interface {
    Na(*Condition)  float64
    Qxa(*Condition) float64
    Qya(*Condition) float64
    Mxa(*Condition) float64
    Mya(*Condition) float64
    Mza(*Condition) float64
}

type Shape interface {
    A()  float64
    IX() float64
    IY() float64
    J()  float64
}

type SColumn struct {
    Steel
}

type HKYOU struct {
    H, B, tw, tf float64
}

type HWEAK struct {
    H, B, tw, tf float64
}

type RPIPE struct {
    H, B, tw, tf float64
}

type CPIPE struct {
    D, t float64
}

type SGirder struct {
    Steel
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
}
