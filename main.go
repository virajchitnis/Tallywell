// Command tallywell runs the local web app: it binds to loopback only, opens
// a native application window (WKWebView on macOS, WebKitGTK on Linux, default
// browser on Windows), and serves the UI until the user closes the window or
// clicks Quit. All logic lives in internal/*; this file is thin wiring.
//
// Set TALLYWELL_NO_TRAY=1 to skip the GUI window and run in headless mode
// (useful for E2E tests and other non-interactive environments).
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/tallywell/tallywell/internal/app"
	"github.com/tallywell/tallywell/internal/gui"
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

	if os.Getenv("TALLYWELL_NO_TRAY") == "1" {
		// Headless mode: block until Ctrl-C, SIGTERM, or web UI Quit.
		done := make(chan struct{})
		var once sync.Once
		srv.SetQuitFunc(func() { once.Do(func() { close(done) }) })

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		select {
		case <-ch:
		case <-done:
		}
		shutdown()
		return
	}

	// GUI mode: native window. run blocks the main goroutine (required on macOS
	// by Cocoa/WKWebView; on Linux by GTK). quit closes the window from any goroutine.
	run, quit := gui.Open(url, "Tallywell")
	srv.SetQuitFunc(quit)
	run()
	shutdown()
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
