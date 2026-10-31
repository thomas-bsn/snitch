package library

import (
	"testing"
	"time"
)

func TestFilenameMetadata(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name       string
		location   *time.Location
		game, date string
	}{
		{"Call of Duty Black Ops 3 2025.04.09 - 21.04.47.04.DVR.mp4", paris, "Call of Duty Black Ops 3", "2025-04-09T19:04:47Z"},
		{"Game 2025.04.09 - 21.04.47.04.DVR.mp4", nil, "Game", ""},
		{"Game 2025.10.26 - 02.30.00.00.DVR.mp4", paris, "Game", ""},
		{"Game 2025.03.30 - 02.30.00.00.DVR.mp4", paris, "Game", ""},
		{"Game 2025.02.30 - 02.30.00.00.DVR.mp4", paris, "Game", ""},
		{"unknown.mp4", paris, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			game, date := FilenameMetadata(tc.name, tc.location)
			if game != tc.game {
				t.Fatalf("game = %q, want %q", game, tc.game)
			}
			if tc.date == "" {
				if date != nil {
					t.Fatalf("unexpected date %v", date)
				}
				return
			}
			if date == nil || date.Format(time.RFC3339) != tc.date {
				t.Fatalf("date = %v, want %s", date, tc.date)
			}
		})
	}
}

func TestExtractCaptureMetadataPrefersContainer(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}
	game, gameOrigin, recorded, dateOrigin := ExtractCaptureMetadata(
		"Other 2025.04.09 - 21.04.47.04.DVR.mp4",
		map[string]string{"game": "Container game", "creation_time": "2025-01-01T10:00:00+02:00"}, paris,
	)
	if game != "Container game" || gameOrigin != "container" || dateOrigin != "container" || recorded == nil || recorded.Format(time.RFC3339) != "2025-01-01T08:00:00Z" {
		t.Fatalf("unexpected metadata: %q %q %v %q", game, gameOrigin, recorded, dateOrigin)
	}
}

func TestContainerDateWithoutZoneUsesConfiguredLocation(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}
	_, _, date, origin := ExtractCaptureMetadata("unknown.mp4", map[string]string{"creation_time": "2025-04-09T21:04:47"}, paris)
	if date == nil || date.Format(time.RFC3339) != "2025-04-09T19:04:47Z" || origin != "container" {
		t.Fatalf("date = %v (%s)", date, origin)
	}
	_, _, date, _ = ExtractCaptureMetadata("unknown.mp4", map[string]string{"creation_time": "2025-10-26T02:30:00"}, paris)
	if date != nil {
		t.Fatalf("ambiguous date was guessed: %v", date)
	}
}
