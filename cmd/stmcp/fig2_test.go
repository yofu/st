package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/japanese"
)

func TestFig2TemplateAndNewDocument(t *testing.T) {
	dir := t.TempDir()
	template := "common\r\n    inputfile\r\nfor i in [1,2]\r\n    page a4tate\r\n    title 旧タイトル $i\r\n    height 0 $i\r\nendfor\r\n"
	raw, err := japanese.ShiftJIS.NewEncoder().String(template)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "base.fig2")
	if err = os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	s := modelSession{root: root}
	call := func(name, args string, wantError bool) map[string]any {
		t.Helper()
		value, err := s.call(name, []byte(args))
		if wantError {
			if err == nil {
				t.Fatalf("%s accepted %s", name, args)
			}
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return value.(map[string]any)
	}
	if got := call("read_fig2", `{"path":"base.fig2"}`, false); got["encoding"] != "cp932" || got["content"] != template {
		t.Fatal(got)
	}
	call("preview_fig2", `{"template":"base.fig2","replace":[{"find":"旧タイトル","text":"構造モデル"}]}`, false)
	call("save_fig2_as", `{"draft_id":1,"path":"copied.fig2"}`, false)
	call("save_fig2_as", `{"draft_id":1,"path":"base.fig2"}`, true)
	call("save_fig2_as", `{"draft_id":1,"path":"../escaped.fig2"}`, true)
	encoded, err := os.ReadFile(filepath.Join(dir, "copied.fig2"))
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := fig2Text(encoded)
	if want := "common\r\n    inputfile\r\nfor i in [1,2]\r\n    page a4tate\r\n    title 構造モデル $i\r\n    height 0 $i\r\nendfor\r\n"; err != nil || decoded != want {
		t.Fatalf("modified template: %q, %v", decoded, err)
	}
	unchanged, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(unchanged, []byte(raw)) {
		t.Fatal("original template changed", err)
	}
	call("preview_fig2", `{"content":"for i in [1,2]\npage a4tate"}`, true)
	call("preview_fig2", `{"common":["inputfile"],"pages":[{"paper":"a4yoko","title":"部材番号","commands":["elem column girder","elemcode"]}]}`, false)
	call("save_fig2_as", `{"draft_id":1,"path":"stale.fig2"}`, true)
	call("save_fig2_as", `{"draft_id":2,"path":"new.fig2"}`, false)
	if got := call("list_fig2", `{}`, false)["paths"].([]string); len(got) != 3 {
		t.Fatal(got)
	}
}
