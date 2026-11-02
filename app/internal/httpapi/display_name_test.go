package httpapi

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateDisplayName(t *testing.T) {
	if got := truncateDisplayName("short", 128); got != "short" {
		t.Fatalf("a short name is kept: %q", got)
	}
	// 3-byte characters: a cut at 128 bytes would split one and PostgreSQL refuses the result.
	long := strings.Repeat("日", 200)
	got := truncateDisplayName(long, 128)
	if !utf8.ValidString(got) || utf8.RuneCountInString(got) != 128 {
		t.Fatalf("expected 128 whole characters, got %d runes, valid=%v", utf8.RuneCountInString(got), utf8.ValidString(got))
	}
	emoji := strings.Repeat("a😀", 100)
	if got := truncateDisplayName(emoji, 128); !utf8.ValidString(got) || utf8.RuneCountInString(got) != 128 {
		t.Fatalf("mixed width characters: %d runes", utf8.RuneCountInString(got))
	}
}
