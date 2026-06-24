//go:build !windows

// Package gui opens the Tallywell UI in a native application window.
// On macOS a WKWebView window is used; on Linux a WebKitGTK window.
// The HTTP server must be running before Open is called.
package gui

import webview "github.com/webview/webview_go"

const (
	windowWidth  = 1280
	windowHeight = 800
)

// Open creates a native application window and returns two functions.
// run must be called on the main goroutine — it blocks until the window
// is closed. quit may be called from any goroutine to close the window
// programmatically (e.g. from the web UI Quit handler).
func Open(url, title string) (run func(), quit func()) {
	wv := webview.New(false)
	quit = func() { wv.Dispatch(wv.Terminate) }
	run = func() {
		defer wv.Destroy()
		wv.SetTitle(title)
		wv.SetSize(windowWidth, windowHeight, webview.HintNone)
		wv.Navigate(url)
		wv.Run()
	}
	return
}
