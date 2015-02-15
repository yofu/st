package arclm

import (
	"math"
)

func Normalize(vec []float64) []float64 {
	sum := 0.0
	for _, val := range vec {
		sum += val * val
	}
	sum = math.Sqrt(sum)
	if sum == 0.0 {
		return vec
	}
	for i := range vec {
		vec[i] /= sum
	}
	return vec
}

func Dot(x, y []float64, size int) float64 {
	rtn := 0.0
	for i := 0; i < size; i++ {
		rtn += x[i] * y[i]
	}
	return rtn
}

func Cross(x, y []float64) []float64 {
	rtn := make([]float64, 3)
	rtn[0] = x[1]*y[2] - x[2]*y[1]
	rtn[1] = x[2]*y[0] - x[0]*y[2]
	rtn[2] = x[0]*y[1] - x[1]*y[0]
	return rtn
}
