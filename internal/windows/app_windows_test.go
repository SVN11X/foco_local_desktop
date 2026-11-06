package windows

import "testing"

func TestFormatDuration(t *testing.T) {
	for _, tc := range []struct {
		millis int64
		want   string
	}{
		{-1, "00:00"}, {0, "00:00"}, {999, "00:00"},
		{5000, "00:05"}, {62000, "01:02"}, {120000, "02:00"},
	} {
		if got := formatDuration(tc.millis); got != tc.want {
			t.Errorf("formatDuration(%d): got %q want %q", tc.millis, got, tc.want)
		}
	}
}
