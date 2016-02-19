package st

type Directory struct {
	home string
	cwd  string
}

func NewDirectory(home, cwd string) *Directory {
	return &Directory{
		home: home,
		cwd:  cwd,
	}
}

func (d *Directory) Home() string {
	return d.home
}

func (d *Directory) SetHome(home string) {
	d.home = home
}

func (d *Directory) Cwd() string {
	return d.cwd
}

func (d *Directory) SetCwd(cwd string) {
	d.cwd = cwd
}
