package arclm

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
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

// vecs are assumed to be part of orthonomal basis
func OrthoNormalBasis(size int, vecs ...[]float64) ([][]float64, error) {
	rtn := make([][]float64, size)
	num := 0
	for _, v := range vecs {
		if len(v) != size {
			return nil, fmt.Errorf("size error")
		}
		rtn[num] = make([]float64, size)
		for i := 0; i < size; i++ {
			rtn[num][i] = v[i]
		}
		num++
		if num >= size {
			return rtn, nil
		}
	}
	var wg sync.WaitGroup
	for ind := num; ind < size; ind++ {
		wg.Add(1)
		go func(i int) {
			rtn[i] = make([]float64, size)
			for j := 0; j < size; j++ {
				rtn[i][j] = rand.Float64()
			}
			// NORMALIZATION
			sum := 0.0
			for j := 0; j < size; j++ {
				sum += rtn[i][j] * rtn[i][j]
			}
			sum = math.Sqrt(sum)
			for j := 0; j < size; j++ {
				rtn[i][j] /= sum
			}
			wg.Done()
		}(ind)
	}
	wg.Wait()
	for i := num; i < size; i++ {
		// GRAM-SCHMIDT ORTHONORMALIZATION
		for j := 0; j < i; j++ {
			sum := 0.0
			for k := 0; k < size; k++ {
				sum += rtn[i][k]*rtn[j][k]
			}
			for k := 0; k < size; k++ {
				rtn[i][k] -= sum*rtn[j][k]
			}
		}
	}
	return rtn, nil
}
