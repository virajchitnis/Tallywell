package gui

import "testing"

func TestWindowDimensions(t *testing.T) {
	if windowWidth <= 0 || windowHeight <= 0 {
		t.Errorf("invalid window dimensions: %dx%d", windowWidth, windowHeight)
	}
}
