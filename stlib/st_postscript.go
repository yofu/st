package st

import (
	"github.com/yofu/ps"
	"os"
)


func (frame *Frame) PostScript (fn string) error {
	cvs := ps.NewCanvas()
	for _, el := range frame.Elems {
		el.PostScript(cvs, frame.Show)
	}
	w, err := os.Create(fn)
	if err != nil {
		return err
	}
	cvs.WriteTo(w)
	return nil
}

func (elem *Elem) PostScript (cvs *ps.Canvas, show *Show) {
	if elem.IsLineElem() {
		cvs.FLine(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1], elem.Enod[1].Pcoord[0], elem.Enod[1].Pcoord[1])
	} else {
		lines := make([]string, elem.Enods)
		lines[0] = ps.FMoveTo(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1])
		for i:=1; i<elem.Enods; i++ {
			lines[i] = ps.FLineTo(elem.Enod[i].Pcoord[0], elem.Enod[i].Pcoord[1])
		}
		cvs.Fill(true, lines...)
	}
}
