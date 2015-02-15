package matrix

func MatrixVector(mat [][]float64, vec []float64) []float64 {
	size := len(mat)
	rtn := make([]float64, size)
	for i := 0; i < size; i++ {
		rtn[i] = 0.0
		for j := 0; j < size; j++ {
			rtn[i] += mat[i][j] * vec[j]
		}
	}
	return rtn
}

func MatrixMatrix(a, b [][]float64) [][]float64 {
	size := len(a)
	rtn := make([][]float64, size)
	for i := 0; i < size; i++ {
		rtn[i] = make([]float64, size)
		for j := 0; j < size; j++ {
			for k := 0; k < size; k++ {
				rtn[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return rtn
}

func MatrixTranspose(m [][]float64) [][]float64 {
	size := len(m)
	rtn := make([][]float64, size)
	for i := 0; i < size; i++ {
		rtn[i] = make([]float64, size)
		for j := 0; j < size; j++ {
			rtn[i][j] = m[j][i]
		}
	}
	return rtn
}
