package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"snitch/internal/jobs"
)

// benchWork is one task of a run, before the model is chosen: a clip, a step and the input version that makes it
// one work (the same for every model, so models are measured on the same inputs).
type benchTask struct {
	model string
	work  benchWork
}

type benchWork struct {
	clip    int64
	step    string
	version string
}

// benchmarkWorks lists the tasks a stage needs for a corpus, and the parameters to record with the run. It has
// answered the request itself when it returns false.
func (s *Server) benchmarkWorks(w http.ResponseWriter, r *http.Request, stage string, datasetID int64) ([]benchWork, []byte, bool) {
	var works []benchWork
	var params map[string]any
	switch stage {
	case "diarization":
		// One task per certified clip and audio track, on the same audio as the production diarization.
		rows, err := s.Pool.Query(r.Context(), `SELECT DISTINCT clip_id,audio_stream_index FROM benchmark_reference_turns WHERE dataset_id=$1 ORDER BY 1,2`, datasetID)
		if err != nil {
			s.internal(w, err)
			return nil, nil, false
		}
		defer rows.Close()
		for rows.Next() {
			var clip int64
			var track int
			if err = rows.Scan(&clip, &track); err != nil {
				s.internal(w, err)
				return nil, nil, false
			}
			works = append(works, benchWork{clip, fmt.Sprintf("%s%d", jobs.BenchmarkDiarizationPrefix, track), fmt.Sprintf("bd-d%d-a%d", datasetID, track)})
		}
		if err = rows.Err(); err != nil {
			s.internal(w, err)
			return nil, nil, false
		}
		params = map[string]any{"reference": "frozen validated turns", "collar_seconds": derCollar, "resolution_seconds": derResolution, "profile": "production", "clips": len(works)}
	case "transcription":
		// One task per certified clip and audio track, on the production profile and without vocabulary.
		rows, err := s.Pool.Query(r.Context(), `SELECT DISTINCT clip_id,audio_stream_index FROM benchmark_reference_segments WHERE dataset_id=$1 ORDER BY 1,2`, datasetID)
		if err != nil {
			s.internal(w, err)
			return nil, nil, false
		}
		defer rows.Close()
		for rows.Next() {
			var clip int64
			var track int
			if err = rows.Scan(&clip, &track); err != nil {
				s.internal(w, err)
				return nil, nil, false
			}
			works = append(works, benchWork{clip, fmt.Sprintf("%s%d", jobs.BenchmarkTranscriptionPrefix, track), fmt.Sprintf("bt-d%d-a%d", datasetID, track)})
		}
		if err = rows.Err(); err != nil {
			s.internal(w, err)
			return nil, nil, false
		}
		params = map[string]any{"reference": "frozen corrected segments", "normalisation": "lowercase, punctuation removed", "profile": "production", "vocabulary": "none", "clips": len(works)}
	default:
		// The distinct fingerprint works of the dataset: one per clip, audio track and diarization.
		rows, err := s.Pool.Query(r.Context(), `SELECT DISTINCT ls.clip_id,ls.audio_stream_index,ls.diarization_job_id,dj.input_version
			FROM benchmark_dataset_items i JOIN local_speakers ls ON ls.id=i.local_speaker_id JOIN jobs dj ON dj.id=ls.diarization_job_id
			WHERE i.dataset_id=$1 ORDER BY 1,2,3`, datasetID)
		if err != nil {
			s.internal(w, err)
			return nil, nil, false
		}
		defer rows.Close()
		for rows.Next() {
			var clip, diarization int64
			var track int
			var diarizationVersion string
			if err = rows.Scan(&clip, &track, &diarization, &diarizationVersion); err != nil {
				s.internal(w, err)
				return nil, nil, false
			}
			pipeline, _, ok := jobs.VoiceInputVersion(diarizationVersion)
			if !ok {
				fail(w, http.StatusUnprocessableEntity, "benchmark_dataset_stale")
				return nil, nil, false
			}
			works = append(works, benchWork{clip, fmt.Sprintf("voice.fingerprint.%d", track), fmt.Sprintf("f5-a%d-p%s-d%d", track, pipeline, diarization)})
		}
		if err = rows.Err(); err != nil {
			s.internal(w, err)
			return nil, nil, false
		}
		params = map[string]any{"split": "reference/test", "scope": "speaker", "preprocessing": "worker pipeline", "clips": len(works)}
	}
	if len(works) == 0 {
		fail(w, http.StatusUnprocessableEntity, "benchmark_dataset_stale")
		return nil, nil, false
	}
	raw, _ := json.Marshal(params)
	return works, raw, true
}
