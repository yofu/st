package matrix

func MatrixMatrix (a, b [][]float64) [][]float64 {
    size := len(a)
    rtn := make([][]float64, size)
    for i:=0; i<size; i++ {
        rtn[i] = make([]float64, size)
        for j:=0; j<size; j++ {
            for k:=0; k<size; k++ {
                rtn[i][j] += a[i][k] * b[k][j]
            }
        }
    }
    return rtn
}

func MatrixTranspose (m [][]float64) [][]float64 {
    size := len(m)
    rtn := make([][]float64, size)
    for i:=0; i<size; i++ {
        rtn[i] = make([]float64, size)
        for j:=0; j<size; j++ {
            rtn[i][j] = m[j][i]
        }
    }
    return rtn
}
