package windows

import (
	"testing"
	"unsafe"
)

func TestFormatDuration(t *testing.T) {
	for _, tc := range []struct {
		millis int64
		want   string
	}{
		{-1, "00:00"}, {0, "00:00"}, {999, "00:00"},
		{5000, "00:05"}, {62000, "01:02"}, {120000, "02:00"}, {3600000, "01:00:00"},
	} {
		if got := formatDuration(tc.millis); got != tc.want {
			t.Errorf("formatDuration(%d): got %q want %q", tc.millis, got, tc.want)
		}
	}
}

func TestNativeWin64StructLayouts(t *testing.T) {
	// Windows sends raw callback memory; catch layout regressions at build time.
	if unsafe.Sizeof(drawItemStruct{}) != 64 {
		t.Fatalf("DRAWITEMSTRUCT size = %d", unsafe.Sizeof(drawItemStruct{}))
	}
	if unsafe.Sizeof(paintStruct{}) != 72 {
		t.Fatalf("PAINTSTRUCT size = %d", unsafe.Sizeof(paintStruct{}))
	}
}
