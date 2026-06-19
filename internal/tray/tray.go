// Package tray manages the system-tray icon (macOS menu bar, Windows taskbar
// tray, Linux desktop tray). It blocks the calling goroutine via systray.Run;
// on macOS this must be the main goroutine (Cocoa requires UI on the main
// thread), so the HTTP server must already be running in a goroutine before
// Run is called.
package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/signal"
	"syscall"

	"fyne.io/systray"
)

// Run starts the system-tray icon and blocks until the user clicks Quit or
// the process receives SIGINT/SIGTERM. openURL is called when the user clicks
// "Open Tallywell"; onQuit is called after the tray has exited so the caller
// can shut down the HTTP server.
func Run(url string, openURL func(string), onQuit func()) {
	// Forward OS signals so Ctrl-C in a terminal also quits gracefully.
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		systray.Quit()
	}()

	systray.Run(func() {
		icon := makeIcon()
		// SetTemplateIcon lets macOS auto-adapt the icon to dark/light mode.
		systray.SetTemplateIcon(icon, icon)
		systray.SetTooltip("Tallywell")

		mOpen := systray.AddMenuItem("Open Tallywell", "Open Tallywell in your browser")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit Tallywell", "Stop the Tallywell server")

		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					openURL(url)
				case <-mQuit.ClickedCh:
					systray.Quit()
					return
				}
			}
		}()
	}, onQuit)
}

// Quit programmatically stops the tray loop. Safe to call from any goroutine;
// used by the web UI Quit button to trigger a clean shutdown.
func Quit() { systray.Quit() }

// makeIcon generates a 22×22 "T" glyph as a PNG — black on transparent,
// suitable for macOS template icon rendering (auto-adapts dark/light mode).
func makeIcon() []byte {
	const size = 22
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	black := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	// Horizontal bar
	for y := 4; y <= 7; y++ {
		for x := 3; x <= 18; x++ {
			img.SetNRGBA(x, y, black)
		}
	}
	// Vertical stem
	for y := 7; y <= 18; y++ {
		for x := 9; x <= 12; x++ {
			img.SetNRGBA(x, y, black)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		// bytes.Buffer.Write never returns an error; this branch is unreachable.
		return nil
	}
	return buf.Bytes()
}
