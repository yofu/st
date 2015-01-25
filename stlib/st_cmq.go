package st

import (
	"errors"
)

type Cmq struct {
	Ci  float64
	Cj  float64
	Mc0 float64
	Qi0 float64
	Qj0 float64
}

func Concentration(P float64, L0, L1, L2 float64) (*Cmq, error) {
	if L0 != L1+L2 {
		return nil, errors.New("CMQ Concentration: Length mismatched")
	}
	c := new(Cmq)
	c.Ci = -P * L1 * L2 * L2 / (L0 * L0)
	c.Cj = P * L1 * L1 * L2 / (L0 * L0)
	c.Qi0 = P * L2 / L0
	c.Qj0 = P * L1 / L0
	if L1 >= L2 {
		c.Mc0 = 0.5 * P * L2
	} else {
		c.Mc0 = 0.5 * P * L1
	}
	return c, nil
}

/*            w0          */
/*            .           */
/*          .::           */
/*        .::::           */
/* ......::::::.......... */
/*   L1    L2      L3     */
/*          L0            */ // by Jun SATO
func RightAngledTriangle(w float64, L0, L1, L2, L3 float64) (*Cmq, error) {
	if L0 != L1+L2+L3 {
		return nil, errors.New("CMQ RightAngledTriangle: Length mismatched")
	}
	c := new(Cmq)
	c.Ci = -1.0 / 60.0 * w * L2 / (L0 * L0) * (5.0*(L2*L2+4.0*L2*L3+6.0*L3*L3)*L1 + 2.0*(L2*L2+5.0*L2*L3+10.0*L3*L3)*L2)
	c.Cj = 1.0 / 60.0 * w * L2 / (L0 * L0) * (5.0*(6.0*L1*L1+8.0*L1*L2+3.0*L2*L2)*L3 + (10.0*L1*L1+10.0*L1*L2+3.0*L2*L2)*L2)
	c.Qi0 = 1.0 / 6.0 * w * L2 / L0 * (L2 + 3.0*L3)
	c.Qj0 = 1.0 / 6.0 * w * L2 / L0 * (3.0*L1 + 2.0*L2)
	if 0.5*L0 <= L1 {
		c.Mc0 = 0.5 * L0 * c.Qi0
	} else if 0.5*L0 < L1+L2 {
		c.Mc0 = 1.0 / 48.0 * w / L2 * (3.0*L2*L2*(L1+L2+3.0*L3) + (L1-L3)*(L1-L3)*(L1-3.0*L2-L3))
	} else {
		c.Mc0 = 0.5 * L0 * c.Qj0
	}
	return c, nil
}

func Uniform(L float64, w float64) (*Cmq, error) {
	c0, err := RightAngledTriangle(w, L, 0.0, L, 0.0)
	if err != nil {
		return nil, err
	}
	c := new(Cmq)
	c.Ci = c0.Ci - c0.Cj
	c.Cj = c0.Cj - c0.Ci
	c.Mc0 = 2.0 * c0.Mc0
	c.Qi0 = c0.Qi0 + c0.Qj0
	c.Qj0 = c0.Qj0 + c0.Qi0
	return c, nil
}

// TODO: implement
func Polygon(elem *Elem) (*Cmq, error) {
	return nil, nil
}
