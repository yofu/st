package st

type Axis struct {
	Frame     *Frame
	Origin    []float64
	Direction [][]float64
	Current   int
}

func (axis *Axis) Project(view *View) ([]float64, [][]float64) {
	start := view.ProjectCoord(axis.Origin)
	end := make([][]float64, 3)
	for i := 0; i < 3; i++ {
		tmp := make([]float64, 3)
		for j := 0; j < 3; j++ {
			tmp[j] = axis.Origin[j] + axis.Direction[i][j]
		}
		end[i] = view.ProjectCoord(tmp)
	}
	return start, end
}
