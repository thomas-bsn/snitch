package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"snitch/internal/jobs"
)

// validBenchmarkDiarization checks the shape of a diarization benchmark result: speakers with finite, ordered turns.
// It is the only thing a benchmark result must satisfy; nothing of it enters the local speakers of a clip.
func validBenchmarkDiarization(raw json.RawMessage) bool {
	var result struct {
		Speakers []struct {
			Label string `json:"label"`
			Turns []struct {
				Start *float64 `json:"start"`
				End   *float64 `json:"end"`
			} `json:"turns"`
		} `json:"speakers"`
	}
	if json.Unmarshal(raw, &result) != nil || result.Speakers == nil {
		return false
	}
	for _, speaker := range result.Speakers {
		if speaker.Label == "" || speaker.Turns == nil {
			return false
		}
		for _, turn := range speaker.Turns {
			if turn.Start == nil || turn.End == nil || *turn.Start < 0 || *turn.End < *turn.Start || math.IsNaN(*turn.Start) || math.IsNaN(*turn.End) || math.IsInf(*turn.End, 0) {
				return false
			}
		}
	}
	return true
}

// snapshotReferenceTurns freezes the speaking turns of the certified clips of a corpus (ADR 0059). A speaker is
// the validated Person, or the local voice itself when nobody validated it. Times are put back on the clock of the
// audio stream, as a Worker measures them.
func (s *Server) snapshotReferenceTurns(ctx context.Context, tx pgQuerier, datasetID int64, clips []int64) error {
	rows, err := tx.Query(ctx, `SELECT ls.diarization_job_id,ls.clip_id,ls.audio_stream_index,COALESCE('p'||a.person_id::text,'v'||ls.id::text),st.start_seconds,st.end_seconds
		FROM local_speakers ls JOIN speaker_turns st ON st.local_speaker_id=ls.id
		LEFT JOIN speaker_attributions a ON a.local_speaker_id=ls.id AND a.validation_state='validated'
		WHERE ls.clip_id=ANY($1) AND ls.kind='diarized' ORDER BY ls.clip_id,ls.audio_stream_index,st.start_seconds`, clips)
	if err != nil {
		return err
	}
	type turn struct {
		clip           int64
		track          int
		speaker        string
		start, end     float64
		diarizationJob int64
	}
	var turns []turn
	for rows.Next() {
		var t turn
		if err = rows.Scan(&t.diarizationJob, &t.clip, &t.track, &t.speaker, &t.start, &t.end); err != nil {
			rows.Close()
			return err
		}
		turns = append(turns, t)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	offsets := map[int64]float64{}
	for _, t := range turns {
		offset, known := offsets[t.diarizationJob]
		if !known {
			if offset, err = jobs.PlanTrackOffset(ctx, tx, t.diarizationJob); err != nil {
				return err
			}
			offsets[t.diarizationJob] = offset
		}
		start, end := math.Max(t.start-offset, 0), t.end-offset
		if end <= start {
			continue
		}
		if _, err = tx.Exec(ctx, `INSERT INTO benchmark_reference_turns(dataset_id,clip_id,audio_stream_index,speaker,start_seconds,end_seconds) VALUES($1,$2,$3,$4,$5,$6)`, datasetID, t.clip, t.track, t.speaker, start, end); err != nil {
			return err
		}
	}
	return nil
}

type diarizationKey = transcriptKey

// computeDiarizationResult measures one diarization model of a run against the frozen reference turns, on the
// clips and tracks its own tasks finished.
func computeDiarizationResult(ctx context.Context, q pgQuerier, runID, datasetID int64, modelID string) (benchModelResult, error) {
	res := benchModelResult{model: modelID}
	var repo, revision string
	err := q.QueryRow(ctx, `SELECT repo,revision FROM voice_model_catalog WHERE id=$1`, modelID).Scan(&repo, &revision)
	if err == nil {
		res.model, res.version = repo, revision
	}
	rows, err := q.Query(ctx, `SELECT clip_id,audio_stream_index,speaker,start_seconds,end_seconds FROM benchmark_reference_turns WHERE dataset_id=$1 ORDER BY clip_id,audio_stream_index,start_seconds`, datasetID)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	references := map[diarizationKey][]speakerTurn{}
	var order []diarizationKey
	for rows.Next() {
		var key diarizationKey
		var t speakerTurn
		if err = rows.Scan(&key.clip, &key.track, &t.speaker, &t.start, &t.end); err != nil {
			return res, err
		}
		if _, seen := references[key]; !seen {
			order = append(order, key)
		}
		references[key] = append(references[key], t)
	}
	if err = rows.Err(); err != nil {
		return res, err
	}
	rows.Close()
	res.total = len(order)

	jobRows, err := q.Query(ctx, `SELECT j.clip_id,substring(j.step FROM '[0-9]+$')::int,j.result FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id
		WHERE rj.run_id=$1 AND rj.model_id=$2 AND j.status='done' AND j.result IS NOT NULL`, runID, modelID)
	if err != nil {
		return res, err
	}
	defer jobRows.Close()
	hypotheses := map[diarizationKey][]speakerTurn{}
	for jobRows.Next() {
		var key diarizationKey
		var raw []byte
		if err = jobRows.Scan(&key.clip, &key.track, &raw); err != nil {
			return res, err
		}
		var result struct {
			Speakers []struct {
				Label string `json:"label"`
				Turns []struct {
					Start float64 `json:"start"`
					End   float64 `json:"end"`
				} `json:"turns"`
			} `json:"speakers"`
		}
		if json.Unmarshal(raw, &result) != nil {
			continue // a result that cannot be read is a task that did not measure
		}
		hyp := []speakerTurn{}
		for _, speaker := range result.Speakers {
			for _, t := range speaker.Turns {
				hyp = append(hyp, speakerTurn{t.Start, t.End, speaker.Label})
			}
		}
		hypotheses[key] = hyp
	}
	if err = jobRows.Err(); err != nil {
		return res, err
	}
	jobRows.Close()

	var clips []clipTurns
	for _, key := range order {
		if hyp, ok := hypotheses[key]; ok {
			clips = append(clips, clipTurns{references[key], hyp})
		}
	}
	res.scored = len(clips)
	if err = q.QueryRow(ctx, `SELECT extract(epoch FROM max(j.updated_at)-min(j.created_at)) FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 AND rj.model_id=$2 AND j.status='done'`, runID, modelID).Scan(&res.elapsed); err != nil {
		return res, err
	}
	switch {
	case res.scored == 0:
		res.state = "failed"
	case res.scored < res.total:
		res.state = "partial"
	default:
		res.state = "complete"
	}
	if res.metrics, err = json.Marshal(evaluateDiarization(clips)); err != nil {
		return res, fmt.Errorf("encode diarization metrics: %w", err)
	}
	return res, nil
}
