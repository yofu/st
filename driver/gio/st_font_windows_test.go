package stgio

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"golang.org/x/image/math/fixed"
)

// Unique Japanese characters must have distinct glyphs, rather than all being
// replaced by the same missing-glyph box. Use the actual popup title style.
func TestPopupJapaneseTitleGlyphs(t *testing.T) {
	h := newUIHarness(t)
	w := h.w
	label := w.popupTitle("回転角方向倍率基準線名座標入力参照点指定取消")
	seenRune := map[rune]bool{}
	seenGlyph := map[text.GlyphID]rune{}
	for _, r := range label.Text {
		if seenRune[r] {
			continue
		}
		seenRune[r] = true
		label.Shaper.LayoutString(text.Parameters{Font: label.Font, PxPerEm: fixed.I(30), MaxWidth: 200}, string(r))
		found := false
		for g, ok := label.Shaper.NextGlyph(); ok; g, ok = label.Shaper.NextGlyph() {
			if g.Advance <= 0 {
				continue
			}
			found = true
			if prev, ok := seenGlyph[g.ID]; ok && prev != r {
				t.Errorf("Japanese glyphs %c and %c both use missing/repeated glyph %v", prev, r, g.ID)
			}
			seenGlyph[g.ID] = r
		}
		if !found {
			t.Errorf("no glyph for Japanese character %c", r)
		}
	}
}

func TestPopupJapaneseWindowsRendering(t *testing.T) {
	size := image.Pt(1000, 550)
	gpu, err := headless.NewWindow(size.X, size.Y)
	if err != nil {
		t.Skipf("Direct3D unavailable: %v", err)
	}
	defer gpu.Release()
	w := NewWindow(nil)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1.5, PxPerSp: 1.5}, Constraints: layout.Exact(size)}
	paint.Fill(&ops, w.Theme.Bg)
	w.modalBox(gtx, "回転角[度]（参照点から指定する場合はR）", "方向ベクトル X,Y,Z ／ 中心座標 ／ 基準線名 ／ 倍率", func(c layout.Context) layout.Dimensions { return w.uiLabel(c, "日本語入力・確定・取消") })
	if err := gpu.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rectangle{Max: size})
	if err := gpu.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	if out := os.Getenv("ST_GIO_FONT_RENDER_DIR"); out != "" {
		f, err := os.Create(filepath.Join(out, "gio-popup-japanese.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
