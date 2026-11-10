package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestViewerStaticAssets(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"index.html", "stio.wasm", "analysis.wasm", "analysis-worker.js", "private.inp"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	handler := viewerHandler(dir)
	for _, tc := range []struct {
		method, path string
		status       int
		content      string
	}{{"GET", "/", 200, "text/html; charset=utf-8"}, {"GET", "/stio.wasm", 200, "application/wasm"}, {"HEAD", "/stio.wasm", 200, "application/wasm"}, {"GET", "/analysis.wasm", 200, "application/wasm"}, {"GET", "/analysis-worker.js", 200, "text/javascript; charset=utf-8"}, {"GET", "/private.inp", 404, ""}, {"POST", "/stio.wasm", 405, ""}} {
		r := httptest.NewRecorder()
		handler.ServeHTTP(r, httptest.NewRequest(tc.method, tc.path, nil))
		if r.Code != tc.status || (tc.content != "" && r.Header().Get("Content-Type") != tc.content) {
			t.Errorf("%s %s: status %d content type %q", tc.method, tc.path, r.Code, r.Header().Get("Content-Type"))
		}
	}
}
