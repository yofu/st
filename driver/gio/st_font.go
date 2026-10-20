package stgio

import (
	"gioui.org/font"
	"gioui.org/widget/material"
)

// Keep popup headings on Japanese-capable regular faces. H6 requests Medium,
// which can resolve to a different font/fallback than the working body text on
// Windows. A regular Japanese family avoids missing-glyph boxes in the title.
func (w *Window) popupTitle(title string) material.LabelStyle {
	label := material.H6(w.Theme, title)
	label.Font.Weight = font.Normal
	label.Font.Typeface = "Yu Gothic UI, Meiryo, Yu Gothic, MS Gothic, Noto Sans CJK JP, Hiragino Sans, sans-serif"
	return label
}
