package st

type Alias struct {
	section map[int]string
}

func NewAlias() *Alias {
	return &Alias{
		section: make(map[int]string, 0),
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
