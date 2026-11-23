package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeDispatchIndexIsCurrent(t *testing.T) {
	dir := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(dir, "native_commands.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.ReplaceAll(string(data), "\r\n", "\n") != catalog(dir) {
		t.Fatal("native command catalog is stale; run go generate ./stlib")
	}
}
