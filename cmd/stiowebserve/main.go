// stiowebserve serves only the static browser viewer assets on a local network.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	listen := flag.String("listen", ":8080", "listen address (e.g. 127.0.0.1:8080 for this PC only)")
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	dir := flag.String("dir", filepath.Dir(exe), "directory containing the built viewer assets")
	flag.Parse()
	for name := range assets {
		if _, err := os.Stat(filepath.Join(*dir, name)); err != nil {
			log.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	fmt.Printf("Viewer: http://127.0.0.1:%d/\n", port)
	if ips, err := net.InterfaceAddrs(); err == nil && listener.Addr().(*net.TCPAddr).IP.IsUnspecified() {
		for _, address := range ips {
			if ip, ok := address.(*net.IPNet); ok && ip.IP.To4() != nil && !ip.IP.IsLoopback() && !ip.IP.IsLinkLocalUnicast() {
				fmt.Printf("LAN:    http://%s:%d/\n", ip.IP, port)
			}
		}
	}
	fmt.Println("Press Ctrl+C to stop.")
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, Handler: viewerHandler(*dir)}
	log.Fatal(server.Serve(listener))
}

var assets = map[string]string{"index.html": "text/html; charset=utf-8", "viewer.css": "text/css; charset=utf-8", "viewer.js": "text/javascript; charset=utf-8", "workspace.js": "text/javascript; charset=utf-8", "wasm_exec.js": "text/javascript; charset=utf-8", "stio.wasm": "application/wasm", "analysis.wasm": "application/wasm", "analysis-worker.js": "text/javascript; charset=utf-8"}

func viewerHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := r.URL.Path
		if name == "/" {
			name = "/index.html"
		}
		if name == "" || name[0] != '/' {
			http.NotFound(w, r)
			return
		}
		name = name[1:]
		content, ok := assets[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", content)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(dir, name))
	})
}
