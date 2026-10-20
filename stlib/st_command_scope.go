package st

// InteractiveScope lets a driver serialize asynchronous command steps with its
// renderer. Drivers without this optional interface retain existing behavior.
type InteractiveScope interface {
	Wait(...any)
	WaitDirect(...any)
	Resume()
	Close()
}

type idleCommandScope struct{}

func (idleCommandScope) Wait(...any)       {}
func (idleCommandScope) WaitDirect(...any) {}
func (idleCommandScope) Resume()           {}
func (idleCommandScope) Close()            {}

func commandScope(stw Commander) InteractiveScope {
	if host, ok := stw.(interface{ BeginInteractiveCommand() InteractiveScope }); ok {
		return host.BeginInteractiveCommand()
	}
	return idleCommandScope{}
}

// A second-stage element receive must also yield the model lock, and must be
// cancellable even when no further element is clicked.
func receiveCommandElem(scope InteractiveScope, ch chan *Elem, quit chan bool) (*Elem, bool) {
	scope.WaitDirect(ch)
	var el *Elem
	cancelled := false
	select {
	case el = <-ch:
	case <-quit:
		cancelled = true
	}
	scope.Resume()
	return el, cancelled
}
