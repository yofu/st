package st

type Font interface {
	Face()  string
	SetFace(string)
	Size()  int
	SetSize(int)
	Color() int
	SetColor(int)
}
