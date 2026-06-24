package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
)

func TestMakeTemplateIcon(t *testing.T) {
	data := makeTemplateIcon()
	if data == nil {
		t.Fatal("makeTemplateIcon returned nil")
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a valid PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 22 || b.Dy() != 22 {
		t.Errorf("expected 22×22, got %d×%d", b.Dx(), b.Dy())
	}
	// Horizontal bar pixels should be black and opaque
	r32, g32, b32, a32 := img.At(10, 5).RGBA()
	if a32 == 0 {
		t.Error("expected opaque pixel in the horizontal bar area, got transparent")
	}
	if r32 != 0 || g32 != 0 || b32 != 0 {
		t.Errorf("expected black pixel in bar, got r=%d g=%d b=%d", r32>>8, g32>>8, b32>>8)
	}
	// Corner should be transparent
	_, _, _, cornerA := img.At(0, 0).RGBA()
	if cornerA != 0 {
		t.Error("expected transparent corner pixel")
	}
}

func TestMakeColorIcon(t *testing.T) {
	data := makeColorIcon()
	if data == nil {
		t.Fatal("makeColorIcon returned nil")
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a valid PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 32 || b.Dy() != 32 {
		t.Errorf("expected 32×32, got %d×%d", b.Dx(), b.Dy())
	}
	// A background pixel not covered by any bar or the diagonal line.
	// (4,4) is inside the rounded rect, below the bars (y<9), and off the
	// diagonal which only reaches x=4 at y=25 (outside segment at y=4).
	r32, g32, b32, a32 := img.At(4, 4).RGBA()
	if a32 == 0 {
		t.Error("expected opaque background pixel at (4,4)")
	}
	// Green channel should dominate (accent colour #3b6d57 → r=59 g=109 b=87)
	if g32 <= r32 || g32 <= b32 {
		t.Errorf("background pixel (4,4) should be greenish, got r=%d g=%d b=%d", r32>>8, g32>>8, b32>>8)
	}
	// Strict corner should be transparent (outside rounded rect)
	_, _, _, cornerA := img.At(0, 0).RGBA()
	if cornerA != 0 {
		t.Error("expected transparent corner pixel (outside rounded rect)")
	}
}

func TestRRContains(t *testing.T) {
	const sz, r = 32, 7.0
	cases := []struct {
		px, py float64
		want   bool
	}{
		{16, 16, true},  // centre
		{0, 0, false},   // strict corner — outside arc
		{7, 0, true},    // start of straight top edge
		{0, 7, true},    // start of straight left edge
		{7, 7, true},    // inside corner arc centre
		{1, 1, false},   // deep in corner — outside
		{31, 31, false}, // opposite strict corner
	}
	for _, tc := range cases {
		got := rrContains(tc.px, tc.py, sz, r)
		if got != tc.want {
			t.Errorf("rrContains(%.0f, %.0f, %d, %.0f) = %v, want %v",
				tc.px, tc.py, sz, r, got, tc.want)
		}
	}
}

func TestDrawLine(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	// Horizontal line through the middle
	drawLine(img, 0, 16, 31, 16, 3, white)

	// Pixels on the line centre should be white
	for _, x := range []int{5, 15, 25} {
		_, _, _, a := img.At(x, 16).RGBA()
		if a == 0 {
			t.Errorf("pixel (%d,16) should be drawn by the line, got transparent", x)
		}
	}
	// Pixels far off the line should be untouched
	_, _, _, a := img.At(16, 0).RGBA()
	if a != 0 {
		t.Error("pixel (16,0) should not be drawn by the horizontal line")
	}
}

func TestDrawLineAntiAlias(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	c := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	drawLine(img, 4, 25, 28, 7, 3.5, c)

	// A pixel along the diagonal should have been painted
	painted := false
	for y := 7; y <= 25; y++ {
		for x := 4; x <= 28; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				painted = true
			}
		}
	}
	if !painted {
		t.Error("drawLine produced no visible pixels along the diagonal")
	}
}

func TestRRContainsSymmetry(t *testing.T) {
	const sz, r = 32, 7.0
	// The shape should be symmetric: (x,y) and (sz-1-x, sz-1-y) give same result
	for y := 0; y < sz; y++ {
		for x := 0; x < sz; x++ {
			a := rrContains(float64(x), float64(y), sz, r)
			b := rrContains(float64(sz-1-x), float64(sz-1-y), sz, r)
			if a != b {
				t.Errorf("asymmetry at (%d,%d): %v vs mirror %v", x, y, a, b)
			}
		}
	}
}

func TestMakeColorIconTallyBars(t *testing.T) {
	data := makeColorIcon()
	img, _ := png.Decode(bytes.NewReader(data))

	// Bar pixels (x≈8, y=15 — inside the first bar's column) should be bright/white
	r32, g32, b32, _ := img.At(8, 15).RGBA()
	// All channels should be high (white bar pixel)
	if r32 < 0xc000 || g32 < 0xc000 || b32 < 0xc000 {
		t.Errorf("expected bright bar pixel at (8,15), got r=%d g=%d b=%d", r32>>8, g32>>8, b32>>8)
	}
}

func TestDrawLineWidth(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	c := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	// Vertical line at x=16 with width 3 — pixels at x=15,16,17 should be hit
	drawLine(img, 16, 0, 16, 31, 3, c)
	for _, x := range []int{15, 16, 17} {
		_, _, _, a := img.At(x, 15).RGBA()
		if a == 0 {
			t.Errorf("pixel (%d,15) should be inside the 3px-wide line", x)
		}
	}
	// x=13 should be outside the line
	_, _, _, a := img.At(13, 15).RGBA()
	if a > 0 {
		// allow some anti-alias bleed but it should be very faint
		r32, _, _, _ := img.At(13, 15).RGBA()
		if r32 > 0x1000 {
			t.Errorf("pixel (13,15) should be outside the 3px-wide line, got alpha=%d", a>>8)
		}
	}
}

// Verify the math import is used (this would fail to compile otherwise).
var _ = math.Pi
