package stgio

import (
	"os"
	"path/filepath"
	"sync"

	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"gioui.org/widget/material"
)

const defaultIPAMinchoPath = `C:\Users\yofu\.st\fonts\ipam.ttf`

var (
	ipaMinchoOnce sync.Once
	ipaMinchoData []byte
	ipaMinchoPath string
)

func ipaMinchoCandidates() []string {
	var paths []string
	add := func(p string) {
		if p == "" {
			return
		}
		p = filepath.Clean(p)
		for _, existing := range paths {
			if existing == p {
				return
			}
		}
		paths = append(paths, p)
	}
	add(os.Getenv("ST_FONT"))
	if home := os.Getenv("HOME"); home != "" {
		add(filepath.Join(home, ".st", "fonts", "ipam.ttf"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		add(filepath.Join(home, ".st", "fonts", "ipam.ttf"))
	}
	add(defaultIPAMinchoPath)
	return paths
}

func loadIPAMinchoBytes() ([]byte, string) {
	ipaMinchoOnce.Do(func() {
		for _, path := range ipaMinchoCandidates() {
			data, err := os.ReadFile(path)
			if err == nil && len(data) > 0 {
				ipaMinchoData = data
				ipaMinchoPath = path
				return
			}
		}
	})
	return ipaMinchoData, ipaMinchoPath
}

func parseFontCollectionWithWeights(data []byte) ([]font.FontFace, error) {
	faces, err := opentype.ParseCollection(data)
	if err != nil || len(faces) == 0 {
		return nil, err
	}
	base := faces[0]
	out := make([]font.FontFace, 0, len(faces)+3)
	out = append(out, faces...)
	// Provide Medium/SemiBold/Bold aliases using the same face so material
	// buttons and headings stay in IPA Mincho instead of falling back to sans-serif.
	for _, weight := range []font.Weight{font.Medium, font.SemiBold, font.Bold} {
		alias := base
		alias.Font.Weight = weight
		out = append(out, alias)
	}
	return out, nil
}

func newWindowTheme() *material.Theme {
	th := material.NewTheme()
	data, _ := loadIPAMinchoBytes()
	if len(data) == 0 {
		return th
	}
	collection, err := parseFontCollectionWithWeights(data)
	if err != nil || len(collection) == 0 {
		return th
	}
	th.Shaper = text.NewShaper(text.WithCollection(collection))
	th.Face = collection[0].Font.Typeface
	return th
}

// Keep popup headings on the active Japanese-capable regular face.
func (w *Window) popupTitle(title string) material.LabelStyle {
	label := material.H6(w.Theme, title)
	label.Font.Weight = font.Normal
	if w.Theme.Face != "" {
		label.Font.Typeface = w.Theme.Face
	} else {
		label.Font.Typeface = "Yu Gothic UI, Meiryo, Yu Gothic, MS Gothic, Noto Sans CJK JP, Hiragino Sans, sans-serif"
	}
	return label
}
