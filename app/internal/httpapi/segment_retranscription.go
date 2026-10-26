package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// Retranscription of a corrected clip (phase 6). A retranscription creates a
// new transcription job whose result replaces the visible text once it succeeds; the
// corrections of the replaced job are then marked obsolete (segment_corrections.stale_at,
// set at completion by jobs.CompleteWith) and are never reapplied. Every path that can
// schedule a retranscription goes through correctionsGuard first:
//
//   - POST /api/admin/clips/{id}/transcription, PUT/DELETE /api/admin/clips/{id}/audio-track,
//     POST /api/admin/jobs (a transcription step) and POST /api/admin/jobs/{id}/retry
//     answer 409 corrections_would_be_replaced unless the query parameter
//     confirm_replace_corrections=true is present;
//   - the batch excludes corrected clips unless include_corrected is explicit;
//   - a source audio rule change (reprocessSource) skips corrected clips and counts them.
//
// See docs/segment-corrections.md.

// registerSegmentRetranscriptionRoutes: this step adds no route; it protects the
// existing retranscription operations.
func (s *Server) registerSegmentRetranscriptionRoutes(mux *http.ServeMux) {}

const transcriptionStepPrefix = "transcription.audio."

func isTranscriptionStep(step string) bool { return strings.HasPrefix(step, transcriptionStepPrefix) }

// correctionsAtStake describes the active corrections of the visible transcription of a
// clip: what a retranscription would replace.
type correctionsAtStake struct {
	TranscriptionJobID int64      `json:"transcription_job_id"`
	TextSegments       int        `json:"text_segments"`
	SpeakerSegments    int        `json:"speaker_segments"`
	Authors            []string   `json:"authors"`
	LastCorrectedAt    *time.Time `json:"last_corrected_at"`
}

// activeCorrectionRows is the set of non-obsolete corrections (text or speaker) that
// apply to the visible transcription of each clip, without any clip filter.
const activeCorrectionRows = `SELECT sc.clip_id,sc.transcription_job_id,sc.segment_index,sc.text,sc.speaker_override_id,
 sc.text_author_username,sc.speaker_author_username,GREATEST(sc.text_corrected_at,sc.speaker_corrected_at) corrected_at
 FROM segment_corrections sc JOIN transcript t ON t.job_id=sc.transcription_job_id
 WHERE sc.stale_at IS NULL AND (sc.text IS NOT NULL OR sc.speaker_override_id IS NOT NULL)`

// clipCorrections returns the active corrections of the clip, ok=false when there are none.
func (s *Server) clipCorrections(ctx context.Context, clipID int64) (correctionsAtStake, bool, error) {
	rows, err := s.Pool.Query(ctx, `WITH `+visibleTranscriptsCTE("j.clip_id=$1")+` `+activeCorrectionRows, clipID)
	if err != nil {
		return correctionsAtStake{}, false, err
	}
	defer rows.Close()
	out := correctionsAtStake{Authors: []string{}}
	seen := map[string]bool{}
	for rows.Next() {
		var clip, job int64
		var index int
		var text *string
		var speaker *int64
		var textAuthor, speakerAuthor string
		var at *time.Time
		if err = rows.Scan(&clip, &job, &index, &text, &speaker, &textAuthor, &speakerAuthor, &at); err != nil {
			return correctionsAtStake{}, false, err
		}
		out.TranscriptionJobID = job
		if text != nil {
			out.TextSegments++
		}
		if speaker != nil {
			out.SpeakerSegments++
		}
		for _, author := range []string{textAuthor, speakerAuthor} {
			if author != "" && !seen[author] {
				seen[author] = true
				out.Authors = append(out.Authors, author)
			}
		}
		if at != nil && (out.LastCorrectedAt == nil || at.After(*out.LastCorrectedAt)) {
			out.LastCorrectedAt = at
		}
	}
	if err = rows.Err(); err != nil {
		return correctionsAtStake{}, false, err
	}
	return out, out.TextSegments+out.SpeakerSegments > 0, nil
}

// correctedClipSegments counts, per clip, the active corrections of its visible
// transcription (clips without corrections are absent).
func (s *Server) correctedClipSegments(ctx context.Context) (map[int64]int, error) {
	rows, err := s.Pool.Query(ctx, `WITH `+visibleTranscriptsCTE("")+` SELECT clip_id,count(*) FROM (`+activeCorrectionRows+`) a GROUP BY clip_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var clip int64
		var n int
		if err = rows.Scan(&clip, &n); err != nil {
			return nil, err
		}
		out[clip] = n
	}
	return out, rows.Err()
}

// correctionsGuard stops a retranscription that would replace human corrections unless
// the administrator confirmed with confirm_replace_corrections=true. It answers 409
// corrections_would_be_replaced with the description of what is replaced and returns false;
// it returns true when the caller may go on.
func (s *Server) correctionsGuard(w http.ResponseWriter, r *http.Request, clipID int64) bool {
	if r.URL.Query().Get("confirm_replace_corrections") == "true" {
		return true
	}
	at, found, err := s.clipCorrections(r.Context(), clipID)
	if err != nil {
		s.internal(w, err)
		return false
	}
	if !found {
		return true
	}
	reply(w, http.StatusConflict, map[string]any{"error": "corrections_would_be_replaced", "corrections": at})
	return false
}
