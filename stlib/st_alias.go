package st

type Alias struct {
	section map[int]string
	command map[string]func(Commander) chan bool
}

func NewAlias() *Alias {
	return &Alias{
		section: make(map[int]string, 0),
		command: Commands,
	}
}

func (a *Alias) AddSectionAlias(key int, value string) {
	a.section[key] = value
}

func (a *Alias) DeleteSectionAlias(key int) {
	delete(a.section, key)
}

func (a *Alias) ClearSectionAlias() {
	a.section = make(map[int]string, 0)
}

func (a *Alias) SectionAlias(key int) (string, bool) {
	str, ok := a.section[key]
	return str, ok
}

func (a *Alias) AddCommandAlias(key string, value func(Commander) chan bool) {
	a.command[key] = value
}

func (a *Alias) DeleteCommandAlias(key string) {
	delete(a.command, key)
}

func (a *Alias) ClearCommandAlias() {
	a.command = make(map[string]func(Commander) chan bool, 0)
}

func (a *Alias) CommandAlias(key string) (func(Commander) chan bool, bool) {
	str, ok := a.command[key]
	return str, ok
}
