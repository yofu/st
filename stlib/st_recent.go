package st

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

var (
	recentfn = filepath.Join(home, ".st/recent.dat")
)

type RecentFiles struct {
	size      int
	filenames []string
}

func NewRecentFiles(size int) *RecentFiles {
	return &RecentFiles{
		size:      size,
		filenames: make([]string, size),
	}
}

func (r *RecentFiles) Recent() []string {
	return r.filenames
}

func (r *RecentFiles) AddRecent(fn string) {
	abs, err := filepath.Abs(fn)
	if err == nil {
		fn = abs
	}
	fn = filepath.ToSlash(fn)
	skip := 0
	rtn := make([]string, r.size)
	for i := 1; i < r.size; i++ {
		if r.filenames[i-1] == fn {
			skip = 1
		}
		rtn[i] = r.filenames[i-1+skip]
	}
	rtn[0] = fn
	r.filenames = rtn
}

func (r *RecentFiles) ReadRecent() error {
	f, err := os.Open(recentfn)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	num := 0
	for s.Scan() {
		r.filenames[num] = s.Text()
		num++
		if num >= r.size {
			break
		}
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func (r *RecentFiles) SaveRecent() error {
	w, err := os.Create(recentfn)
	if err != nil {
		return err
	}
	defer w.Close()
	for i := 0; i < r.size; i++ {
		w.WriteString(fmt.Sprintf("%s\n", r.filenames[i]))
	}
	return nil
}
