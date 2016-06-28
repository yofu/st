package st

type Amount map[string]float64

func NewAmount() Amount {
	var a Amount = make(map[string]float64)
	return a
}
