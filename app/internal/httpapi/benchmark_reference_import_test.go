package httpapi

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSTM(t *testing.T) {
	text := ";; a comment\n\nclip1 1 alice 0.00 2.50 <o,f0,female> Bonjour tout le monde\nclip1 1 bob 2.5 4 salut alice\n"
	got, err := parseSTM(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].speaker != "alice" || got[0].start != 0 || got[0].end != 2.5 || got[0].text != "Bonjour tout le monde" {
		t.Fatalf("label and text are told apart: %+v", got)
	}
	if got[1].speaker != "bob" || got[1].text != "salut alice" {
		t.Fatalf("a line without label keeps its whole text: %+v", got[1])
	}
	for name, bad := range map[string]struct {
		in   string
		want error
	}{
		"too few fields":   {"clip1 1 alice 0 1", errImportFormat},
		"not a number":     {"clip1 1 alice zero 1 hello", errImportFormat},
		"infinite":         {"clip1 1 alice 0 Inf hello", errImportFormat},
		"end before start": {"clip1 1 alice 3 1 hello", errImportTimes},
		"negative start":   {"clip1 1 alice -1 1 hello", errImportTimes},
		"empty":            {";; nothing\n", errImportFormat},
	} {
		if _, err := parseSTM(bad.in); !errors.Is(err, bad.want) {
			t.Errorf("%s: want %v, got %v", name, bad.want, err)
		}
	}
}

func TestParseRTTM(t *testing.T) {
	text := ";; c\nSPEAKER clip1 1 0.00 2.50 <NA> <NA> alice <NA> <NA>\nSPEAKER clip1 1 2.5 1.5 <NA> <NA> bob <NA> <NA>\n"
	got, err := parseRTTM(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].speaker != "alice" || got[0].end != 2.5 || got[1].start != 2.5 || got[1].end != 4 {
		t.Fatalf("the duration is added to the start: %+v", got)
	}
	for name, bad := range map[string]struct {
		in   string
		want error
	}{
		"not a speaker record": {"LEXEME clip1 1 0 1 <NA> <NA> alice <NA> <NA>", errImportFormat},
		"missing fields":       {"SPEAKER clip1 1 0 1 <NA> alice", errImportFormat},
		"zero duration":        {"SPEAKER clip1 1 0 0 <NA> <NA> alice <NA> <NA>", errImportTimes},
		"negative start":       {"SPEAKER clip1 1 -2 1 <NA> <NA> alice <NA> <NA>", errImportTimes},
		"no speaker":           {"SPEAKER clip1 1 0 1 <NA> <NA> <NA> <NA> <NA>", errImportFormat},
		"empty":                {"", errImportFormat},
	} {
		if _, err := parseRTTM(bad.in); !errors.Is(err, bad.want) {
			t.Errorf("%s: want %v, got %v", name, bad.want, err)
		}
	}
}

func TestImportedSpeakersOverlap(t *testing.T) {
	stm := []stmSegment{{speaker: "alice"}, {speaker: "bob"}}
	if !referenceSpeakersOverlap(stm, []rttmTurn{{speaker: "bob"}}) {
		t.Fatal("one shared name is enough")
	}
	if referenceSpeakersOverlap(stm, []rttmTurn{{speaker: "carol"}}) {
		t.Fatal("no shared name must be refused")
	}
}

func TestImportRefusesControlCharactersAndAbsurdLines(t *testing.T) {
	// A NUL byte is refused by the database: it must be a format error, not a failure after the transaction began.
	for _, text := range []string{"f 1 alice 0.0 1.0 bon\x00jour\n", "f 1 alice\x01 0.0 1.0 bonjour\n", "f 1 alice 0.0 1.0 " + strings.Repeat("a", benchMaxImportLine) + "\n"} {
		if _, err := parseSTM(text); err == nil {
			t.Fatalf("a control character or an absurd line is refused: %q", text[:min(len(text), 30)])
		}
	}
	if _, err := parseRTTM("SPEAKER f 1 0.0 1.0 <NA> <NA> al\x00ice <NA> <NA>\n"); err == nil {
		t.Fatal("a NUL in an RTTM is refused too")
	}
	if _, err := parseSTM("f 1 alice 0.0 1.0 bonjour\ttout\n"); err != nil {
		t.Fatalf("a tab separates fields in the wild: %v", err)
	}
}
