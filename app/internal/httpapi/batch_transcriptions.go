package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"snitch/internal/jobs"
)

type batchPreview struct {
	Eligible          int     `json:"eligible"`
	AlreadyProcessed  int     `json:"already_processed"`
	AlreadyQueued     int     `json:"already_queued"`
	Unavailable       int     `json:"unavailable"`
	SelectionRequired int     `json:"selection_required"`
	ClipIDs           []int64 `json:"-"`
}

func (s *Server) batchTranscriptionPreview(w http.ResponseWriter, r *http.Request) {
	preview, err := s.previewTranscriptionBatch(r)
	if errors.Is(err, jobs.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_filter")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	preview.ClipIDs = nil
	reply(w, http.StatusOK, preview)
}

func (s *Server) batchTranscriptionSchedule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Confirm bool `json:"confirm"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !input.Confirm {
		fail(w, http.StatusBadRequest, "batch_confirmation_required")
		return
	}
	preview, err := s.previewTranscriptionBatch(r)
	if errors.Is(err, jobs.ErrInvalidInput) {
		fail(w, http.StatusBadRequest, "invalid_filter")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	scheduled := 0
	for _, id := range preview.ClipIDs {
		_, created, scheduleErr := s.scheduleResolvedTranscription(r.Context(), id, nil, true)
		if scheduleErr != nil {
			s.internal(w, scheduleErr)
			return
		}
		if created {
			scheduled++
		}
	}
	preview.ClipIDs = nil
	reply(w, http.StatusOK, map[string]any{"preview": preview, "scheduled": scheduled})
}

// previewTranscriptionBatch evaluates the same library filters and current track
// selection used by clip details. Scheduling is a separate, explicit admin POST.
func (s *Server) previewTranscriptionBatch(r *http.Request) (batchPreview, error) {
	q := r.URL.Query()
	visibility, status := q.Get("visibility"), q.Get("status")
	if (visibility != "" && visibility != "private" && visibility != "shared") || (status != "" && status != "ready" && status != "waiting_stability" && status != "unreadable" && status != "missing") {
		return batchPreview{}, errBatchFilter
	}
	ownerID, sourceID := int64(0), int64(0)
	var err error
	for key, dest := range map[string]*int64{"owner_id": &ownerID, "source_id": &sourceID} {
		if raw := q.Get(key); raw != "" {
			*dest, err = strconv.ParseInt(raw, 10, 64)
			if err != nil || *dest < 1 {
				return batchPreview{}, errBatchFilter
			}
		}
	}
	from, to := q.Get("from"), q.Get("to")
	for _, date := range []string{from, to} {
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return batchPreview{}, errBatchFilter
			}
		}
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT c.id,c.tracks,
	CASE WHEN EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready') THEN true ELSE false END
	FROM clips c WHERE ($1='' OR c.visibility=$1) AND ($2=0 OR c.owner_id=$2) AND ($3=0 OR EXISTS(SELECT 1 FROM source_files sf JOIN clip_sources cs ON cs.id=sf.source_id WHERE sf.clip_id=c.id AND cs.id=$3)) AND ($4='' OR c.recorded_at::date >= $4::date) AND ($5='' OR c.recorded_at::date <= $5::date) AND ($6='' OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status=$6) OR ($6='ready' AND EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id)) OR ($6='missing' AND NOT EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=c.id) AND NOT EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=c.id AND sf.status='ready'))) ORDER BY c.id`, visibility, ownerID, sourceID, from, to, status)
	if err != nil {
		return batchPreview{}, err
	}
	defer rows.Close()
	result := batchPreview{ClipIDs: []int64{}}
	for rows.Next() {
		var id int64
		var raw []byte
		var ready bool
		if err = rows.Scan(&id, &raw, &ready); err != nil {
			return batchPreview{}, err
		}
		if !ready {
			result.Unavailable++
			continue
		}
		if _, mediaErr := s.resolveClipMedia(r.Context(), id); mediaErr != nil {
			if errors.Is(mediaErr, os.ErrNotExist) {
				result.Unavailable++
				continue
			}
			return batchPreview{}, mediaErr
		}
		selection, selErr := s.audioSelection(r.Context(), id, tracksFromClip(json.RawMessage(raw)))
		if selErr != nil {
			return batchPreview{}, selErr
		}
		if selection.AudioStreamIndex == nil {
			result.SelectionRequired++
			continue
		}
		step := "transcription.audio." + strconv.Itoa(*selection.AudioStreamIndex)
		vocabulary, vocabErr := s.activeVocabulary(r.Context())
		if vocabErr != nil {
			return batchPreview{}, vocabErr
		}
		pipelineVersion, pipelineErr := s.activeTranscriptionPipeline(r.Context())
		if pipelineErr != nil {
			return batchPreview{}, pipelineErr
		}
		version := fmt.Sprintf("t4-a%d-r%d-v%s-p%s", *selection.AudioStreamIndex, selection.RuleVersion, vocabularyVersion(vocabulary), pipelineVersion)
		var doneCurrent, queuedCurrent bool
		if err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM jobs WHERE clip_id=$1 AND step=$2 AND input_version=$3 AND status='done'), EXISTS(SELECT 1 FROM jobs WHERE clip_id=$1 AND step=$2 AND input_version=$3 AND status IN ('pending','leased','retry_wait'))`, id, step, version).Scan(&doneCurrent, &queuedCurrent); err != nil {
			return batchPreview{}, err
		}
		if doneCurrent {
			result.AlreadyProcessed++
			continue
		}
		if queuedCurrent {
			result.AlreadyQueued++
			continue
		}
		result.Eligible++
		result.ClipIDs = append(result.ClipIDs, id)
	}
	if err = rows.Err(); err != nil {
		return batchPreview{}, err
	}
	return result, nil
}

var errBatchFilter = jobs.ErrInvalidInput
