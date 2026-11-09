package httpapi

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
)

// External references of a benchmark corpus (#287, ADR 0061): a hand-written STM (text) and RTTM (speaking turns)
// replace the production reference of a test clip. The parsers are strict: a doubtful line refuses the whole file,
// since a silently dropped line would shift every score.

const (
	// benchMaxImportBytes bounds all the imported texts of one corpus.
	benchMaxImportBytes = 2 << 20
	// benchDatasetBodyLimit is the request body of a corpus creation: the imported texts and room for their JSON
	// escaping (the texts themselves are bounded by benchMaxImportBytes).
	benchDatasetBodyLimit = 4*benchMaxImportBytes + 64<<10
	// benchMaxImportLines bounds the lines of one file.
	benchMaxImportLines = 100000
)

var (
	errImportFormat = errors.New("benchmark import: invalid format")
	errImportTimes  = errors.New("benchmark import: inconsistent times")
)

type stmSegment struct {
	speaker    string
	start, end float64
	text       string
}

type rttmTurn struct {
	speaker    string
	start, end float64
}

// importSeconds reads a time: a finite number. Negative values are the caller's to judge (a time error).
func importSeconds(raw string) (float64, bool) {
	v, err := strconv.ParseFloat(raw, 64)
	return v, err == nil && !math.IsNaN(v) && !math.IsInf(v, 0)
}

// importLines yields the meaningful lines of a file: blank lines and ";;" comments are dropped.
func importLines(text string) ([]string, error) {
	raw := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(raw) > benchMaxImportLines {
		return nil, errImportFormat
	}
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";;") {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

// parseSTM reads `<file> <channel> <speaker> <start> <end> [<labels>] text`. A line without words is a silence and
// is skipped; a file without any word is refused.
func parseSTM(text string) ([]stmSegment, error) {
	lines, err := importLines(text)
	if err != nil {
		return nil, err
	}
	var out []stmSegment
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			return nil, errImportFormat
		}
		start, okStart := importSeconds(fields[3])
		end, okEnd := importSeconds(fields[4])
		if !okStart || !okEnd {
			return nil, errImportFormat
		}
		if start < 0 || end <= start {
			return nil, errImportTimes
		}
		words := fields[5:]
		if strings.HasPrefix(words[0], "<") && strings.HasSuffix(words[0], ">") {
			words = words[1:]
		}
		if len(words) == 0 {
			continue
		}
		out = append(out, stmSegment{speaker: fields[2], start: start, end: end, text: strings.Join(words, " ")})
	}
	if len(out) == 0 {
		return nil, errImportFormat
	}
	return out, nil
}

// parseRTTM reads `SPEAKER <file> <channel> <start> <duration> <NA> <NA> <speaker> <NA> <NA>`.
func parseRTTM(text string) ([]rttmTurn, error) {
	lines, err := importLines(text)
	if err != nil {
		return nil, err
	}
	var out []rttmTurn
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 10 || fields[0] != "SPEAKER" || fields[7] == "<NA>" {
			return nil, errImportFormat
		}
		start, okStart := importSeconds(fields[3])
		duration, okDuration := importSeconds(fields[4])
		if !okStart || !okDuration {
			return nil, errImportFormat
		}
		if start < 0 || duration <= 0 {
			return nil, errImportTimes
		}
		out = append(out, rttmTurn{speaker: fields[7], start: start, end: start + duration})
	}
	if len(out) == 0 {
		return nil, errImportFormat
	}
	return out, nil
}

// referenceSpeakersOverlap tells whether the text and the turns of a clip name at least one speaker in common: two
// files with no name in common describe different recordings or a mislabelled one.
func referenceSpeakersOverlap(segments []stmSegment, turns []rttmTurn) bool {
	named := map[string]bool{}
	for _, s := range segments {
		named[s.speaker] = true
	}
	for _, t := range turns {
		if named[t.speaker] {
			return true
		}
	}
	return false
}

type benchmarkImportInput struct {
	ClipID int64  `json:"clip_id"`
	STM    string `json:"stm"`
	RTTM   string `json:"rttm"`
}

// referenceImport is the checked import of one test clip.
type referenceImport struct {
	clip     int64
	track    int
	segments []stmSegment
	turns    []rttmTurn
}

// prepareReferenceImports checks the imports of a corpus against its roles and parses them. It returns the stable
// error code of the first refusal, "" when all is well. The track is resolved later, against the database.
func prepareReferenceImports(inputs []benchmarkImportInput, role map[int64]string) ([]referenceImport, string) {
	total := 0
	for _, in := range inputs {
		total += len(in.STM) + len(in.RTTM)
	}
	if total > benchMaxImportBytes {
		return nil, "benchmark_import_size"
	}
	seen := map[int64]bool{}
	var out []referenceImport
	for _, in := range inputs {
		// Only a test clip carries a reference, and once.
		if role[in.ClipID] != "test" || seen[in.ClipID] {
			return nil, "benchmark_import_clip"
		}
		seen[in.ClipID] = true
		item := referenceImport{clip: in.ClipID}
		var err error
		if strings.TrimSpace(in.STM) == "" && strings.TrimSpace(in.RTTM) == "" {
			return nil, "benchmark_import_format"
		}
		if strings.TrimSpace(in.STM) != "" {
			if item.segments, err = parseSTM(in.STM); err != nil {
				return nil, importErrorCode(err)
			}
		}
		if strings.TrimSpace(in.RTTM) != "" {
			if item.turns, err = parseRTTM(in.RTTM); err != nil {
				return nil, importErrorCode(err)
			}
		}
		if item.segments != nil && item.turns != nil && !referenceSpeakersOverlap(item.segments, item.turns) {
			return nil, "benchmark_import_speakers"
		}
		out = append(out, item)
	}
	return out, ""
}

func importErrorCode(err error) string {
	if errors.Is(err, errImportTimes) {
		return "benchmark_import_times"
	}
	return "benchmark_import_format"
}

// importTrack is the audio track an imported reference lives on: the one of the production transcript of the clip,
// or of its diarized voices. The two files carry no track of their own; a clip with several tracks is ambiguous.
func importTrack(ctx context.Context, q pgQuerier, clip int64) (int, bool, error) {
	rows, err := q.Query(ctx, `SELECT substring(step FROM '[0-9]+$')::int FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' AND status='done' AND requested_model_id IS NULL
		UNION SELECT audio_stream_index FROM local_speakers WHERE clip_id=$1 AND kind='diarized'`, clip)
	if err != nil {
		return 0, false, err
	}
	defer rows.Close()
	var tracks []int
	for rows.Next() {
		var track int
		if err = rows.Scan(&track); err != nil {
			return 0, false, err
		}
		tracks = append(tracks, track)
	}
	if err = rows.Err(); err != nil {
		return 0, false, err
	}
	if len(tracks) != 1 {
		return 0, false, nil
	}
	return tracks[0], true, nil
}
