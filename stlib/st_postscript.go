package st

import (
	"fmt"
	"github.com/yofu/ps"
	"os"
)

func (frame *Frame) CentringTo(paper ps.Paper) {
	w, h := paper.Size()
	if paper.Portrait {
		frame.View.Center[0] = float64(w) * 0.5
		frame.View.Center[1] = float64(h) * 0.5
	} else {
		frame.View.Center[0] = float64(h) * 0.5
		frame.View.Center[1] = float64(w) * 0.5
	}
	fmt.Println(frame.View.Center)
}

func (frame *Frame) PrintPostScript(fn string, paper ps.Paper) error {
	doc := ps.NewDoc(fn)
	doc.SetPaperSize(paper)
	doc.Canvas.NewPage("1", paper)
	frame.PostScript(doc.Canvas)
	w, err := os.Create(fn)
	defer w.Close()
	if err != nil {
		return err
	}
	doc.WriteTo(w)
	return nil
}

func (frame *Frame) PostScript(cvs *ps.Canvas) {
	for _, n := range frame.Nodes {
		frame.View.ProjectNode(n)
		if frame.Show.Deformation {
			frame.View.ProjectDeformation(n, frame.Show)
		}
	}
	for _, el := range frame.Elems {
		if el.IsHidden(frame.Show) {
			continue
		}
		el.PostScript(cvs, frame.Show)
	}
}

func (elem *Elem) PostScript(cvs *ps.Canvas, show *Show) {
	if elem.IsLineElem() {
		cvs.FLine(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1], elem.Enod[1].Pcoord[0], elem.Enod[1].Pcoord[1])
	} else {
		lines := make([]string, elem.Enods)
		lines[0] = ps.FMoveTo(elem.Enod[0].Pcoord[0], elem.Enod[0].Pcoord[1])
		for i := 1; i < elem.Enods; i++ {
			lines[i] = ps.FLineTo(elem.Enod[i].Pcoord[0], elem.Enod[i].Pcoord[1])
		}
		cvs.Fill(true, lines...)
	}
}
