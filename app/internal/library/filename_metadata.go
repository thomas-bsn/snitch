package library

import (
	"context"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5/pgxpool"
)

var dvrFilename = regexp.MustCompile(`^(.+?) (\d{4})\.(\d{2})\.(\d{2}) - (\d{2})\.(\d{2})\.(\d{2})\.\d{2}\.DVR$`)

// FilenameMetadata recognises the DVR naming convention without guessing at
// ambiguous local times. A nil location leaves the recording time unknown.
func FilenameMetadata(name string, location *time.Location) (string, *time.Time) {
	base := filepath.Base(name)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	parts := dvrFilename.FindStringSubmatch(base)
	if parts == nil {
		return "", nil
	}
	game := strings.TrimSpace(parts[1])
	if game == "" {
		return game, nil
	}
	var n [6]int
	for i := range n {
		n[i], _ = strconv.Atoi(parts[i+2])
	}
	return game, localCaptureTime(n[0], time.Month(n[1]), n[2], n[3], n[4], n[5], location)
}

func localCaptureTime(y int, m time.Month, d, h, min, sec int, location *time.Location) *time.Time {
	if location == nil {
		return nil
	}
	local := time.Date(y, m, d, h, min, sec, 0, location)
	matches := func(v time.Time) bool {
		return v.Year() == y && v.Month() == m && v.Day() == d && v.Hour() == h && v.Minute() == min && v.Second() == sec
	}
	if !matches(local) {
		return nil
	}
	_, offset := local.Zone()
	for _, delta := range []time.Duration{-24 * time.Hour, 24 * time.Hour} {
		_, other := local.Add(delta).Zone()
		if other != offset && matches(local.Add(time.Duration(offset-other)*time.Second).In(location)) {
			return nil
		}
	}
	utc := local.UTC()
	return &utc
}

// ExtractCaptureMetadata prefers explicitly named container tags to the DVR
// filename. The returned origins describe each independently extracted field.
func ExtractCaptureMetadata(name string, tags map[string]string, location *time.Location) (game, gameOrigin string, recorded *time.Time, dateOrigin string) {
	filenameGame, filenameDate := FilenameMetadata(name, location)
	game = strings.TrimSpace(tag(tags, "game"))
	if game != "" {
		gameOrigin = "container"
	} else if filenameGame != "" {
		game, gameOrigin = filenameGame, "filename"
	}
	if value := tag(tags, "creation_time"); value != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
			utc := parsed.UTC()
			recorded, dateOrigin = &utc, "container"
		} else if parsed, err := time.Parse("2006-01-02T15:04:05", value); err == nil {
			recorded = localCaptureTime(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), location)
			if recorded != nil {
				dateOrigin = "container"
			}
		}
	}
	if recorded == nil && filenameDate != nil {
		recorded, dateOrigin = filenameDate, "filename"
	}
	return
}

// CaptureMetadata reads the configured timezone for a newly discovered clip.
func CaptureMetadata(ctx context.Context, pool *pgxpool.Pool, name string, tags map[string]string) (string, string, *time.Time, string, error) {
	var zone string
	if err := pool.QueryRow(ctx, `SELECT timezone FROM clip_metadata_settings WHERE id=true`).Scan(&zone); err != nil {
		return "", "", nil, "", err
	}
	var location *time.Location
	if zone != "" {
		var err error
		location, err = time.LoadLocation(zone)
		if err != nil {
			return "", "", nil, "", err
		}
	}
	game, gameOrigin, recorded, dateOrigin := ExtractCaptureMetadata(name, tags, location)
	return game, gameOrigin, recorded, dateOrigin, nil
}
