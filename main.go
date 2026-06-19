// Command tallywell runs the local web app: it binds to loopback only, opens
// the default browser, and serves the UI. All logic lives in internal/*; this
// file is thin wiring.
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/tallywell/tallywell/internal/app"
	"github.com/tallywell/tallywell/internal/server"
)

func main() {
	dir, err := dataDir()
	if err != nil {
		log.Fatalf("tallywell: data dir: %v", err)
	}
	a, err := app.New(dir)
	if err != nil {
		log.Fatalf("tallywell: init: %v", err)
	}
	srv, err := server.New(a, server.DefaultAutoLock)
	if err != nil {
		log.Fatalf("tallywell: server: %v", err)
	}

	// Loopback only — never exposed to the network.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("tallywell: listen: %v", err)
	}
	url := fmt.Sprintf("http://%s/", ln.Addr().String())
	fmt.Printf("Tallywell is running at %s\n", url)
	openBrowser(url)

	if err := http.Serve(ln, srv.Handler()); err != nil {
		log.Fatalf("tallywell: serve: %v", err)
	}
}

// dataDir returns the per-user local data directory for Tallywell. On macOS
// this is ~/Library/Application Support/Tallywell.
func dataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Tallywell"), nil
}

// openBrowser best-effort opens url in the user's default browser.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
