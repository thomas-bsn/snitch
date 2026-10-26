package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// This file is the single extension point for the "effective text of a segment"
// (phase 6). The worker transcription in jobs.result is never modified; a segment is
// identified by (transcription job id, index in result.segments) and its correction
// lives in segment_corrections. Every reader of a transcript text (clip reading,
// search, embeddings) goes through one of:
//
//   - visibleTranscriptsCTE + effectiveSegmentsCTE: SQL fragments for set-based reads
//     (search, #52; reindexing, #53);
//   - Server.visibleTranscription + overlaySegments: Go reading of one clip (getClip);
//   - Server.authorizedSegment: resolution of one segment for the correction routes.
//
// See docs/segment-corrections.md.

// queryer is satisfied by both the pool and a transaction.
type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// visibleTranscriptsCTE is the CTE `transcript(clip_id, job_id, result)`: the visible
// transcription of each clip is its newest successful transcription job. A queued,
// running or failed retranscription never replaces it; a successful one does. The
// optional clipFilter is a SQL condition on the alias j (for example "j.clip_id=$1").
func visibleTranscriptsCTE(clipFilter string) string {
	if clipFilter != "" {
		clipFilter = " AND " + clipFilter
	}
	return `transcript AS (SELECT DISTINCT ON (j.clip_id) j.clip_id,j.id job_id,j.result FROM jobs j WHERE j.status='done' AND j.step LIKE 'transcription.audio.%'` + clipFilter + ` ORDER BY j.clip_id,j.updated_at DESC,j.id DESC)`
}

// effectiveSegmentsCTE is the CTE `eff_segments`, one row per segment of each visible
// transcription (it needs the `transcript` CTE). Columns: clip_id, job_id, stream
// (audio stream index, -1 if absent), idx (0-based rank in result.segments), seg (the
// original segment JSON, words included), seg_start, seg_end, original_text, text
// (effective text: the correction when there is one and it is not obsolete), corrected,
// version (0 without correction), author_id, author_username, corrected_at,
// speaker_override_id (reserved for the speaker step; NULL = deduced speaker).
// Word timestamps stay those of the original words: the corrected text is not realigned.
func effectiveSegmentsCTE() string {
	return `eff_segments AS (SELECT t.clip_id,t.job_id,COALESCE((t.result->>'audio_stream_index')::int,-1) stream,(s.ord-1)::int idx,s.seg,
 COALESCE((s.seg->>'start')::double precision,0) seg_start,COALESCE((s.seg->>'end')::double precision,(s.seg->>'start')::double precision,0) seg_end,
 COALESCE(s.seg->>'text','') original_text,COALESCE(sc.text,s.seg->>'text','') text,(sc.text IS NOT NULL) corrected,COALESCE(sc.version,0) version,
 sc.text_author_id author_id,sc.text_author_username author_username,sc.text_corrected_at corrected_at,sc.speaker_override_id
 FROM transcript t CROSS JOIN LATERAL jsonb_array_elements(COALESCE(t.result->'segments','[]'::jsonb)) WITH ORDINALITY s(seg,ord)
 LEFT JOIN segment_corrections sc ON sc.transcription_job_id=t.job_id AND sc.segment_index=s.ord-1 AND sc.stale_at IS NULL)`
}

// segmentState is the current correction state of one segment.
type segmentState struct {
	Index             int
	Version           int
	Text              *string
	AuthorID          *int64
	AuthorUsername    string
	CorrectedAt       *time.Time
	SpeakerOverrideID *int64
	StaleAt           *time.Time
}

const segmentStateColumns = `segment_index,version,text,text_author_id,text_author_username,text_corrected_at,speaker_override_id,stale_at`

func scanSegmentState(row pgx.Row) (segmentState, error) {
	var st segmentState
	err := row.Scan(&st.Index, &st.Version, &st.Text, &st.AuthorID, &st.AuthorUsername, &st.CorrectedAt, &st.SpeakerOverrideID, &st.StaleAt)
	return st, err
}

// loadSegmentStates reads the correction rows of one transcription job by segment index.
func loadSegmentStates(ctx context.Context, q queryer, jobID int64) (map[int]segmentState, error) {
	rows, err := q.Query(ctx, `SELECT `+segmentStateColumns+` FROM segment_corrections WHERE transcription_job_id=$1`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]segmentState{}
	for rows.Next() {
		st, err := scanSegmentState(rows)
		if err != nil {
			return nil, err
		}
		out[st.Index] = st
	}
	return out, rows.Err()
}

// visibleTranscript is the visible transcription of a clip: job id and raw result.
type visibleTranscript struct {
	JobID  int64
	Result json.RawMessage
}

// visibleTranscription returns the visible transcription of the clip (same rule as
// visibleTranscriptsCTE), or ok=false when the clip has none.
func (s *Server) visibleTranscription(ctx context.Context, q queryer, clipID int64) (visibleTranscript, bool, error) {
	var v visibleTranscript
	err := q.QueryRow(ctx, `SELECT id,result FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' AND status='done' ORDER BY updated_at DESC,id DESC LIMIT 1`, clipID).Scan(&v.JobID, &v.Result)
	if errors.Is(err, pgx.ErrNoRows) {
		return visibleTranscript{}, false, nil
	}
	return v, err == nil, err
}

// overlaySegments decorates the transcription object of a clip reading in place: the
// job id, and for each segment its index, original_text, the effective text, and the
// correction fields. Starts, ends and words are never touched. Obsolete corrections
// are ignored.
func overlaySegments(transcription any, jobID int64, states map[int]segmentState) {
	object, ok := transcription.(map[string]any)
	if !ok {
		return
	}
	object["transcription_job_id"] = jobID
	segments, _ := object["segments"].([]any)
	for i, raw := range segments {
		segment, isMap := raw.(map[string]any)
		if !isMap {
			continue
		}
		original, _ := segment["text"].(string)
		segment["index"], segment["original_text"], segment["corrected"], segment["correction_version"] = i, original, false, 0
		st, found := states[i]
		if !found || st.StaleAt != nil {
			continue
		}
		segment["correction_version"] = st.Version
		if st.Text != nil {
			segment["text"], segment["corrected"] = *st.Text, true
			segment["corrected_by"], segment["corrected_at"] = st.AuthorUsername, st.CorrectedAt
		}
	}
}

// segmentRef is one segment of the visible transcription of a visible clip.
type segmentRef struct {
	Clip         clipItem
	JobID        int64
	Index        int
	Count        int
	OriginalText string
	Stream       int
}

// authorizedSegment resolves {id} and {index} of the path to a segment of the visible
// transcription. It answers 400 invalid_clip_id / invalid_segment_index, and 404
// clip_not_found (also for an invisible clip) / segment_not_found (no transcription or
// index out of range).
func (s *Server) authorizedSegment(w http.ResponseWriter, r *http.Request) (segmentRef, bool) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return segmentRef{}, false
	}
	index, ok := parseSegmentIndex(w, r)
	if !ok {
		return segmentRef{}, false
	}
	ref, found, err := s.resolveSegment(r.Context(), s.Pool, clip, index)
	if err != nil {
		s.internal(w, err)
		return segmentRef{}, false
	}
	if !found {
		fail(w, http.StatusNotFound, "segment_not_found")
		return segmentRef{}, false
	}
	return ref, true
}

func segmentIndexOf(raw string) (int, bool) {
	index, err := strconv.Atoi(raw)
	return index, err == nil && index >= 0
}

// resolveSegment looks the segment up in the visible transcription of the clip.
func (s *Server) resolveSegment(ctx context.Context, q queryer, clip clipItem, index int) (segmentRef, bool, error) {
	visible, ok, err := s.visibleTranscription(ctx, q, clip.ID)
	if err != nil || !ok {
		return segmentRef{}, false, err
	}
	var result struct {
		Stream   *int `json:"audio_stream_index"`
		Segments []struct {
			Text string `json:"text"`
		} `json:"segments"`
	}
	if err = json.Unmarshal(visible.Result, &result); err != nil || index >= len(result.Segments) {
		return segmentRef{}, false, nil
	}
	ref := segmentRef{Clip: clip, JobID: visible.JobID, Index: index, Count: len(result.Segments), OriginalText: result.Segments[index].Text, Stream: -1}
	if result.Stream != nil {
		ref.Stream = *result.Stream
	}
	return ref, true, nil
}
