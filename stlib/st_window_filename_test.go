package st

import (
	"path/filepath"
	"testing"
)

func TestCompleteFileNamePlaceholderWithSpaces(t *testing.T) {
	root := t.TempDir()
	model := filepath.Join(root, "解析 model", "sample A.inp")
	for _, tc := range []struct{ input, want string }{
		{"#0", model}, {"%", model}, {"#0:h", filepath.Dir(model)},
		{"#0<.otl", filepath.Join(root, "解析 model", "sample A.otl")},
		{"%<.otx", filepath.Join(root, "解析 model", "sample A.otx")},
	} {
		got := CompleteFileName(tc.input, model, []string{model})
		if len(got) != 1 || got[0] != tc.want {
			t.Errorf("CompleteFileName(%q)=%q, want %q", tc.input, got, tc.want)
		}
	}
	if got := CompleteFileName("#999", model, []string{model}); len(got) == 0 {
		t.Fatal("out-of-range recent reference returned no candidates")
	}
}
