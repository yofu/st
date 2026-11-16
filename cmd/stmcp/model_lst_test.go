package main

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	st "github.com/yofu/st/stlib"
)

func TestRCColumnPairSaveAndOverwrite(t *testing.T) {
	root := t.TempDir()
	inp := []byte("PROP 1 PNAME FC24\n MATERIAL FC24 HFACT 1 EFACT 1\nSECT 101 SNAME RC\n NFIG 1\n FIG 1 FPROP 1 AREA 0.16 IXX 0.002 IYY 0.002 VEN 0.003\nSECT 102 SNAME KEEP\n NFIG 1\n FIG 1 FPROP 1 AREA 0.16 IXX 0.002 IYY 0.002 VEN 0.003\nNODE 1 CORD 0 0 0\nNODE 2 CORD 0 0 3\nELEM 11 ESECT 101 ENODS 2 ENOD 1 2 TYPE COLUMN\n")
	keep := []byte("CODE 102 RC COLUMN KEEP\n CRECT -20 -20 20 20 FC24\n HOOPS 0.005 0.005 SD295\n# retained comment\n")
	lst := append([]byte("CODE 101 RC COLUMN OLD\n CRECT -20 -20 20 20 FC24\n HOOPS 0.005 0.005 SD295\n"), keep...)
	for name, data := range map[string][]byte{"model.inp": inp, "model.lst": lst} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	s := modelSession{root: r, parsePath: filepath.Join(t.TempDir(), "model.inp")}
	if _, err = s.open("model.inp"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.previewRCColumn(rcColumnEdit{Section: 101, Width: 300, Depth: 300, Bars: "8-D19", Steel: "SD345", Cover: 40, Hoop: "2x2-D10@100", HoopSteel: "SD295"}); err != nil {
		t.Fatal(err)
	}
	id := s.pending.ID
	if !bytes.Contains(s.pending.lst, keep) {
		t.Fatal("untouched LST block changed")
	}
	if _, err = s.save(id, "out.inp"); err != nil {
		t.Fatal(err)
	}
	// A changed LST must prevent writing either original.
	outside := append(bytes.Clone(lst), []byte("# concurrent change\n")...)
	if err = os.WriteFile(filepath.Join(root, "model.lst"), outside, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.saveFiles(id, "", true); err == nil || !strings.Contains(err.Error(), "changed since opening") {
		t.Fatalf("expected LST conflict, got %v", err)
	}
	original, err := s.readEditInput("model.inp")
	if err != nil || !bytes.Equal(inp, original) {
		t.Fatal("INP changed on conflict", err)
	}
	if err = os.WriteFile(filepath.Join(root, "model.lst"), lst, 0600); err != nil {
		t.Fatal(err)
	}
	v, err := s.saveFiles(id, "", true)
	if err != nil {
		t.Fatal(err)
	}
	backups := v.(map[string]any)["backups"].([]string)
	if len(backups) != 2 {
		t.Fatal(backups)
	}
	for i, want := range [][]byte{inp, lst} {
		got, err := s.readEditInput(backups[i])
		if err != nil || !bytes.Equal(want, got) {
			t.Fatalf("incorrect backup %s: %v", backups[i], err)
		}
	}
	for _, name := range []string{"model.inp", "out.inp"} {
		if _, err = s.open(name); err != nil {
			t.Fatal(err)
		}
		sec := s.frame.Sects[101]
		col, ok := sec.Allow.(*st.RCColumn)
		if !ok || len(col.Reins) != 8 || col.Breadth(true) != 30 || col.Height(true) != 30 || math.Abs(sec.Figs[0].Value["AREA"]-.09) > 1e-12 {
			t.Fatalf("bad reloaded RC column: %+v", sec)
		}
	}
}
