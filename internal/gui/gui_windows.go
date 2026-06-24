//go:build windows

// Package gui opens the Tallywell UI in the system browser on Windows.
// CGO_ENABLED=0 cross-compilation from Linux is preserved this way until
// a CGO-capable Windows runner is added to CI.
package gui

import (
	"os/exec"
	"sync"
)

const (
	windowWidth  = 1280
	windowHeight = 800
)

// Open starts Tallywell in the default Windows browser and returns two
// functions. run blocks until quit is called. quit may be called from any
// goroutine (safe to call multiple times).
func Open(url, title string) (run func(), quit func()) {
	done := make(chan struct{})
	var once sync.Once
	quit = func() { once.Do(func() { close(done) }) }
	run = func() {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		<-done
	}
	return
}
