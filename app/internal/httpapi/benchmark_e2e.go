package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

// benchmarkChain is a diarization model and a transcription model measured together (ADR 0060). Its tasks are those
// of the two stages: chains that share a model share its tasks, so a model is never run twice on the same clip.
type benchmarkChain struct {
	ID                 string `json:"id"`
	DiarizationModelID string `json:"diarization_model_id"`
	TranscriptionModel string `json:"transcription_model"`
}

type e2eMetrics struct {
	cpwerMetrics
	DER *float64 `json:"der"`
}

// computeE2EResult measures a chain on the clips and tracks where both of its models finished: the transcript
// segments take the speaker whose turns overlap them the most, and the speaker-attributed word error rate compares
// them with the frozen reference, whose segments carry their speaker. The DER of the same turns is reported too.
func computeE2EResult(ctx context.Context, q pgQuerier, runID, datasetID int64, chain benchmarkChain) (benchModelResult, error) {
	res := benchModelResult{model: chain.TranscriptionModel}
	var repo string
	if err := q.QueryRow(ctx, `SELECT repo FROM voice_model_catalog WHERE id=$1`, chain.DiarizationModelID).Scan(&repo); err != nil {
		repo = chain.DiarizationModelID
	}
	res.model = fmt.Sprintf("%s + %s", repo, chain.TranscriptionModel)

	refSegments := map[transcriptKey][]speakerText{}
	rows, err := q.Query(ctx, `SELECT clip_id,audio_stream_index,start_seconds,end_seconds,text,COALESCE(speaker,'') FROM benchmark_reference_segments WHERE dataset_id=$1 ORDER BY clip_id,audio_stream_index,ordinal`, datasetID)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	var order []transcriptKey
	for rows.Next() {
		var key transcriptKey
		var t speakerText
		if err = rows.Scan(&key.clip, &key.track, &t.start, &t.end, &t.text, &t.speaker); err != nil {
			return res, err
		}
		if _, seen := refSegments[key]; !seen {
			order = append(order, key)
		}
		refSegments[key] = append(refSegments[key], t)
	}
	if err = rows.Err(); err != nil {
		return res, err
	}
	rows.Close()
	refTurns := map[transcriptKey][]speakerTurn{}
	turnRows, err := q.Query(ctx, `SELECT clip_id,audio_stream_index,speaker,start_seconds,end_seconds FROM benchmark_reference_turns WHERE dataset_id=$1`, datasetID)
	if err != nil {
		return res, err
	}
	defer turnRows.Close()
	for turnRows.Next() {
		var key transcriptKey
		var t speakerTurn
		if err = turnRows.Scan(&key.clip, &key.track, &t.speaker, &t.start, &t.end); err != nil {
			return res, err
		}
		refTurns[key] = append(refTurns[key], t)
	}
	if err = turnRows.Err(); err != nil {
		return res, err
	}
	turnRows.Close()
	res.total = len(order)

	taskResults := func(model, prefix string) (map[transcriptKey][]byte, error) {
		out := map[transcriptKey][]byte{}
		jr, err := q.Query(ctx, `SELECT j.clip_id,substring(j.step FROM '[0-9]+$')::int,j.result FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id
			WHERE rj.run_id=$1 AND rj.model_id=$2 AND j.step LIKE $3 AND j.status='done' AND j.result IS NOT NULL`, runID, model, prefix+"%")
		if err != nil {
			return nil, err
		}
		defer jr.Close()
		for jr.Next() {
			var key transcriptKey
			var raw []byte
			if err = jr.Scan(&key.clip, &key.track, &raw); err != nil {
				return nil, err
			}
			out[key] = raw
		}
		return out, jr.Err()
	}
	transcripts, err := taskResults(chain.TranscriptionModel, "benchmark.transcription.")
	if err != nil {
		return res, err
	}
	diarizations, err := taskResults(chain.DiarizationModelID, "benchmark.diarization.")
	if err != nil {
		return res, err
	}

	var clips []attributedClip
	var turnClips []clipTurns
	version := ""
	for _, key := range order {
		rawT, okT := transcripts[key]
		rawD, okD := diarizations[key]
		if !okT || !okD {
			continue
		}
		var tr struct {
			Version  string `json:"version"`
			Segments []struct {
				Start float64 `json:"start"`
				End   float64 `json:"end"`
				Text  string  `json:"text"`
			} `json:"segments"`
		}
		var di struct {
			Speakers []struct {
				Label string `json:"label"`
				Turns []struct {
					Start float64 `json:"start"`
					End   float64 `json:"end"`
				} `json:"turns"`
			} `json:"speakers"`
		}
		if json.Unmarshal(rawT, &tr) != nil || json.Unmarshal(rawD, &di) != nil {
			continue // a result that cannot be read is a task that did not measure
		}
		if version == "" {
			version = tr.Version
		}
		segments := make([]timedText, 0, len(tr.Segments))
		for _, s := range tr.Segments {
			segments = append(segments, timedText{s.Start, s.End, s.Text})
		}
		var hypTurns []speakerTurn
		for _, sp := range di.Speakers {
			for _, t := range sp.Turns {
				hypTurns = append(hypTurns, speakerTurn{t.Start, t.End, sp.Label})
			}
		}
		clips = append(clips, attributedClip{reference: refSegments[key], hypothesis: attributeSegments(segments, hypTurns)})
		turnClips = append(turnClips, clipTurns{refTurns[key], hypTurns})
	}
	res.version = version
	res.scored = len(clips)
	jobModels := []string{chain.DiarizationModelID, chain.TranscriptionModel}
	sort.Strings(jobModels)
	if err = q.QueryRow(ctx, `SELECT extract(epoch FROM max(j.updated_at)-min(j.created_at)) FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 AND rj.model_id=ANY($2) AND j.status='done'`, runID, jobModels).Scan(&res.elapsed); err != nil {
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
	metrics := e2eMetrics{cpwerMetrics: evaluateCPWER(clips), DER: evaluateDiarization(turnClips).DER}
	if res.metrics, err = json.Marshal(metrics); err != nil {
		return res, err
	}
	return res, nil
}
