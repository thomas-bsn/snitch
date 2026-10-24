package uploads

import (
	"testing"
)

func TestParseContentRange(t *testing.T) {
	for _, tc := range []struct {
		input string
		valid bool
	}{
		{"bytes 0-7/8", true},
		{"bytes 8-15/16", true},
		{"bytes -1-7/8", false},
		{"bytes 0-8/8", false},
		{"bytes 0-7/0", false},
		{"bytes 0-7/8 extra", false},
		{"bytes 0-9223372036854775807/9223372036854775807", false},
	} {
		start, end, total, err := ParseContentRange(tc.input)
		if (err == nil) != tc.valid {
			t.Errorf("ParseContentRange(%q) = %d,%d,%d,%v; valid=%v", tc.input, start, end, total, err, tc.valid)
		}
	}
}
