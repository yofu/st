package stgio

import (
	"gioui.org/widget"
	st "github.com/yofu/st/stlib"
)

func (s *interactiveCommand) CommandPrompt(title, initial string) chan st.CommandText {
	answer := make(chan st.CommandText, 1)
	respond := func(value st.CommandText) {
		select {
		case answer <- value:
		default:
		}
		if s.Window.interaction.Load() == s {
			s.Window.interactionFocus.Store(true)
		}
		s.QueueRedrawAll()
	}
	d := &pathDialog{title: title, textOnly: true, owner: s, ed: widget.Editor{SingleLine: true, Submit: true}, focus: true}
	// The editor is initialized before publication and used only by the UI after.
	d.ed.SetText(initial)
	d.ed.SetCaret(d.ed.Len(), d.ed.Len())
	d.apply = func(text string) { respond(st.CommandText{Text: text}) }
	d.cancel = func() { respond(st.CommandText{Cancelled: true}) }
	select {
	case s.ui.pathRequests <- d:
	case <-s.cancel:
		respond(st.CommandText{Cancelled: true})
	case <-s.ui.done:
		respond(st.CommandText{Cancelled: true})
	}
	s.QueueRedrawAll()
	return answer
}
