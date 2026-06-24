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
	"math"
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
		// SetTemplateIcon: first arg is the macOS template icon (auto-adapts
		// dark/light mode); second arg is the full-colour icon used on Windows
		// and Linux, where template rendering is not available.
		systray.SetTemplateIcon(makeTemplateIcon(), makeColorIcon())
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

// makeTemplateIcon generates a 22×22 "T" glyph as a PNG — black on
// transparent, suitable for macOS template icon rendering (auto-adapts
// dark/light mode). Not used on Windows/Linux where template rendering is
// unavailable.
func makeTemplateIcon() []byte {
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
		return nil
	}
	return buf.Bytes()
}

// makeColorIcon generates a 32×32 full-colour tally-marks icon for Windows
// and Linux system trays. The design mirrors the app icon: green rounded
// square, four white vertical bars, diagonal strike-through.
func makeColorIcon() []byte {
	const sz = 32
	img := image.NewNRGBA(image.Rect(0, 0, sz, sz))
	green := color.NRGBA{R: 0x3b, G: 0x6d, B: 0x57, A: 0xff}
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	// Rounded square background (corner radius ≈ 22% of size, matching app icon).
	const r = 7.0
	for py := 0; py < sz; py++ {
		for px := 0; px < sz; px++ {
			if rrContains(float64(px), float64(py), sz, r) {
				img.SetNRGBA(px, py, green)
			}
		}
	}

	// Four vertical tally bars (x positions proportional to the 100×100 SVG).
	for _, bx := range []int{7, 12, 17, 22} {
		for py := 9; py < 23; py++ {
			for dx := 0; dx < 3; dx++ {
				img.SetNRGBA(bx+dx, py, white)
			}
		}
	}

	// Diagonal strike-through (scaled from SVG: (13,78)→(87,22) in 100×100).
	drawLine(img, 4, 25, 28, 7, 3.5, white)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

// rrContains reports whether pixel (px, py) lies inside a rounded rectangle
// of size sz×sz with corner radius r.
func rrContains(px, py float64, sz int, r float64) bool {
	s := float64(sz - 1)
	inCornerX := px < r || px > s-r
	inCornerY := py < r || py > s-r
	if !inCornerX || !inCornerY {
		return true
	}
	// Nearest corner arc centre
	cx := r
	if px > s-r {
		cx = s - r
	}
	cy := r
	if py > s-r {
		cy = s - r
	}
	return math.Hypot(px-cx, py-cy) <= r
}

// drawLine anti-aliases a thick line segment from (x1,y1) to (x2,y2) onto img.
func drawLine(img *image.NRGBA, x1, y1, x2, y2, width float64, c color.NRGBA) {
	b := img.Bounds()
	hw := width / 2
	dx, dy := x2-x1, y2-y1
	lenSq := dx*dx + dy*dy
	for py := b.Min.Y; py < b.Max.Y; py++ {
		for px := b.Min.X; px < b.Max.X; px++ {
			rx, ry := float64(px)-x1, float64(py)-y1
			t := (rx*dx + ry*dy) / lenSq
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			ex := x1 + t*dx - float64(px)
			ey := y1 + t*dy - float64(py)
			dist := math.Sqrt(ex*ex + ey*ey)
			a := hw + 0.5 - dist
			if a <= 0 {
				continue
			}
			if a > 1 {
				a = 1
			}
			dst := img.NRGBAAt(px, py)
			img.SetNRGBA(px, py, color.NRGBA{
				R: uint8(float64(c.R)*a + float64(dst.R)*(1-a)),
				G: uint8(float64(c.G)*a + float64(dst.G)*(1-a)),
				B: uint8(float64(c.B)*a + float64(dst.B)*(1-a)),
				A: uint8(math.Min(float64(dst.A)+a*255, 255)),
			})
		}
	}
}
