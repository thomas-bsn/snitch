package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// registerVoiceTraceRoutes registers the administrator's trace of the alignment between
// the diarization and the transcription (#273).
func (s *Server) registerVoiceTraceRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/clips/{id}/voice-trace", s.protected(true, http.HandlerFunc(s.getVoiceTrace)))
}

type traceTurn struct {
	LocalSpeakerID int64   `json:"local_speaker_id"`
	Label          string  `json:"label"`
	Start          float64 `json:"start"`
	End            float64 `json:"end"`
}

type traceCandidate struct {
	LocalSpeakerID int64   `json:"local_speaker_id"`
	OverlapSeconds float64 `json:"overlap_seconds"`
}

type traceSegment struct {
	Index      int              `json:"index"`
	Start      float64          `json:"start"`
	End        float64          `json:"end"`
	Candidates []traceCandidate `json:"candidates"`
	// DeducedSpeakerID is the alignment alone; LocalSpeakerID is what the transcript shows
	// (a manual correction wins). Both are null when no turn overlaps the segment.
	DeducedSpeakerID *int64 `json:"deduced_local_speaker_id"`
	LocalSpeakerID   *int64 `json:"local_speaker_id"`
	Corrected        bool   `json:"corrected"`
}

// getVoiceTrace answers, for the visible transcription of the clip, the raw steps around the
// alignment: the models, the raw turns of the diarization, each segment's bounds with the
// voices that overlap it and the one retained, and the coverage per voice. It carries no
// transcript text: it can be attached to a report without exposing the clip.
func (s *Server) getVoiceTrace(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	visible, states, found, err := s.readVisibleTranscript(ctx, clip.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	var transcript struct {
		Provider         string `json:"provider"`
		Model            string `json:"model"`
		Version          string `json:"version"`
		Language         string `json:"language"`
		AudioStreamIndex *int   `json:"audio_stream_index"`
		Segments         []struct {
			Start *float64 `json:"start"`
			End   *float64 `json:"end"`
		} `json:"segments"`
	}
	if !found || json.Unmarshal(visible.Result, &transcript) != nil || transcript.AudioStreamIndex == nil {
		fail(w, http.StatusNotFound, "not_found")
		return
	}
	stream := *transcript.AudioStreamIndex
	type diarizationInfo struct {
		JobID        int64  `json:"job_id"`
		Provider     string `json:"provider"`
		Model        string `json:"model"`
		ModelVersion string `json:"model_version"`
	}
	var diarization *diarizationInfo
	var info diarizationInfo
	err = s.Pool.QueryRow(ctx, `SELECT id,COALESCE(result->>'provider',''),COALESCE(result->>'model',''),COALESCE(result->>'model_version','') FROM jobs
 WHERE clip_id=$1 AND step=$2 AND status='done' ORDER BY id DESC LIMIT 1`, clip.ID, "voice.diarization."+strconv.Itoa(stream)).Scan(&info.JobID, &info.Provider, &info.Model, &info.ModelVersion)
	if err == nil {
		diarization = &info
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(ctx, `SELECT ls.id,ls.label,st.start_seconds,st.end_seconds FROM speaker_turns st JOIN local_speakers ls ON ls.id=st.local_speaker_id
 WHERE ls.clip_id=$1 AND ls.audio_stream_index=$2 AND ls.kind<>'archived' AND `+notIgnoredVoice+` ORDER BY ls.ordinal,st.turn_index`, clip.ID, stream)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	turns := []traceTurn{}
	coverageTurns := []coverageTurn{}
	for rows.Next() {
		var t traceTurn
		if err = rows.Scan(&t.LocalSpeakerID, &t.Label, &t.Start, &t.End); err != nil {
			s.internal(w, err)
			return
		}
		turns = append(turns, t)
		coverageTurns = append(coverageTurns, coverageTurn{SpeakerID: t.LocalSpeakerID, Start: t.Start, End: t.End})
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	segments := []traceSegment{}
	coverageSegments := []coverageSegment{}
	for position, raw := range transcript.Segments {
		if raw.Start == nil || raw.End == nil {
			continue
		}
		order, overlap := segmentOverlaps(coverageTurns, *raw.Start, *raw.End)
		seg := traceSegment{Index: position, Start: *raw.Start, End: *raw.End, Candidates: []traceCandidate{}}
		for _, id := range order {
			seg.Candidates = append(seg.Candidates, traceCandidate{LocalSpeakerID: id, OverlapSeconds: overlap[id]})
		}
		cs := coverageSegment{Start: *raw.Start, End: *raw.End}
		if winner, any := dominantSpeaker(order, overlap); any {
			seg.DeducedSpeakerID, seg.LocalSpeakerID = &winner, &winner
		}
		if st, has := states[position]; has && st.StaleAt == nil && st.SpeakerOverrideID != nil {
			seg.LocalSpeakerID, seg.Corrected = st.SpeakerOverrideID, true
			cs.OverrideSpeakerID = st.SpeakerOverrideID
		}
		segments = append(segments, seg)
		coverageSegments = append(coverageSegments, cs)
	}
	reply(w, http.StatusOK, map[string]any{
		"clip_id": clip.ID, "audio_stream_index": stream,
		"transcription": map[string]any{"job_id": visible.JobID, "provider": transcript.Provider, "model": transcript.Model, "version": transcript.Version, "language": transcript.Language, "segment_count": len(transcript.Segments)},
		"diarization":   diarization,
		"turns":         turns,
		"segments":      segments,
		"coverage":      speakerCoverage(coverageTurns, coverageSegments),
	})
}
