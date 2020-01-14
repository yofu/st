package st

// Paper Size
const (
	A4_TATE = iota
	A4_YOKO
	A3_TATE
	A3_YOKO
)

const (
	W0 = 210.0
	H0 = 297.0
)

func PaperSizemm(name uint) (float64, float64) {
	switch name {
	case A4_TATE:
		return W0, H0
	case A4_YOKO:
		return H0, W0
	case A3_TATE:
		return H0, W0 * 2.0
	case A3_YOKO:
		return W0 * 2.0, H0
	default:
		return 0.0, 0.0
	}
}

func PaperName(name uint) (string, string) {
	switch name {
	case A4_TATE:
		return "A4", "Portrait"
	case A4_YOKO:
		return "A4", "Landscape"
	case A3_TATE:
		return "A3", "Portrait"
	case A3_YOKO:
		return "A3", "Landscape"
	default:
		return "", ""
	}
}
