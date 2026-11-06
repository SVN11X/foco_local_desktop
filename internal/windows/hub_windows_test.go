package windows

import "testing"

func TestParseMonths(t *testing.T) {
	for _, tc := range []struct {
		input string
		mask  uint16
		valid bool
	}{
		{"", 0, true}, {"1,3", 5, true}, {"4-6", 56, true},
		{"1,12", 2049, true}, {"0", 0, false}, {"13", 0, false}, {"3-2", 0, false},
	} {
		m, e := parseMonths(tc.input)
		if (e == nil) != tc.valid {
			t.Errorf("%q validity: %v", tc.input, e)
		}
		if tc.valid && m != tc.mask {
			t.Errorf("%q: got %d want %d", tc.input, m, tc.mask)
		}
	}
}
