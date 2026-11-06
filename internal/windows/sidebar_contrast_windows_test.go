package windows

import (
	"math"
	"testing"
)

// COLORREF is 0x00bbggrr. Check contrast with the actual Win32 color order.
func relativeLuminance(c uint32) float64 {
	linear := func(byteValue uint32) float64 {
		v := float64(byteValue) / 255.0
		if v <= .04045 {
			return v / 12.92
		}
		return math.Pow((v+.055)/1.055, 2.4)
	}
	return .2126*linear(c&255) + .7152*linear((c>>8)&255) + .0722*linear((c>>16)&255)
}
func contrastRatio(foreground, background uint32) float64 {
	a, b := relativeLuminance(foreground), relativeLuminance(background)
	if a < b {
		a, b = b, a
	}
	return (a + .05) / (b + .05)
}
func TestSidebarTextContrastInAllThemes(t *testing.T) {
	for _, tc := range []struct {
		name string
		bg   uint32
	}{
		{"light", lightPalette[1]}, {"dark", darkPalette[1]},
		{"light-action", lightPalette[2]}, {"dark-action", darkPalette[2]},
		{"selected", sidebarSelected}, {"selected-pressed", sidebarSelectedActive},
	} {
		if actual := contrastRatio(sidebarInk, tc.bg); actual < 4.5 {
			t.Errorf("%s: sidebar primary contrast %.2f < 4.5", tc.name, actual)
		}
	}
	for _, tc := range []struct {
		name string
		bg   uint32
	}{
		{"light", lightPalette[1]}, {"dark", darkPalette[1]},
	} {
		if actual := contrastRatio(sidebarInkSecondary, tc.bg); actual < 4.5 {
			t.Errorf("%s: sidebar secondary contrast %.2f < 4.5", tc.name, actual)
		}
	}
}
