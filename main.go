// Command tallywell runs the local web app: it binds to loopback only, opens
// the default browser, shows a system-tray icon, and serves the UI until the
// user clicks Quit in the tray. All logic lives in internal/*; this file is
// thin wiring.
//
// Set TALLYWELL_NO_TRAY=1 to skip the system-tray icon and run in terminal
// mode instead (useful for headless environments and integration tests).
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/tallywell/tallywell/internal/app"
	"github.com/tallywell/tallywell/internal/server"
	"github.com/tallywell/tallywell/internal/tray"
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

	// Serve in a goroutine so the main goroutine is free for the tray (macOS
	// requires Cocoa UI on the main thread).
	httpSrv := &http.Server{Handler: srv.Handler()}
	go func() {
		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("tallywell: serve: %v", err)
		}
	}()

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx)
	}

	openBrowser(url)

	if os.Getenv("TALLYWELL_NO_TRAY") == "1" {
		// Terminal / headless mode: run until Ctrl-C or SIGTERM.
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		shutdown()
		return
	}

	// tray.Run blocks until the user clicks Quit in the menu-bar icon.
	tray.Run(url, openBrowser, shutdown)
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
