package httpapi

import (
	"context"
	"encoding/json"
	"strings"
)

type transcriptKey struct {
	clip  int64
	track int
}

// computeTranscriptionResult measures one transcription model of a run: the transcripts its own tasks produced,
// against the reference text frozen in the corpus. A task that did not finish leaves its clip and track out, and
// the result says how many were measured.
func computeTranscriptionResult(ctx context.Context, q pgQuerier, runID, datasetID int64, model string) (benchModelResult, error) {
	res := benchModelResult{model: model}
	rows, err := q.Query(ctx, `SELECT clip_id,audio_stream_index,text FROM benchmark_reference_segments WHERE dataset_id=$1 ORDER BY clip_id,audio_stream_index,ordinal`, datasetID)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	references := map[transcriptKey][]string{}
	var order []transcriptKey
	for rows.Next() {
		var key transcriptKey
		var text string
		if err = rows.Scan(&key.clip, &key.track, &text); err != nil {
			return res, err
		}
		if _, seen := references[key]; !seen {
			order = append(order, key)
		}
		references[key] = append(references[key], text)
	}
	if err = rows.Err(); err != nil {
		return res, err
	}
	rows.Close()
	res.total = len(order)

	jobRows, err := q.Query(ctx, `SELECT j.clip_id,substring(j.step FROM '[0-9]+$')::int,j.result FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id
		WHERE rj.run_id=$1 AND rj.model_id=$2 AND j.status='done' AND j.result IS NOT NULL`, runID, model)
	if err != nil {
		return res, err
	}
	defer jobRows.Close()
	hypotheses := map[transcriptKey]string{}
	for jobRows.Next() {
		var key transcriptKey
		var raw []byte
		if err = jobRows.Scan(&key.clip, &key.track, &raw); err != nil {
			return res, err
		}
		var result struct {
			Model    string `json:"model"`
			Version  string `json:"version"`
			Segments []struct {
				Text string `json:"text"`
			} `json:"segments"`
		}
		if json.Unmarshal(raw, &result) != nil {
			continue // a result that cannot be read is a task that did not measure
		}
		texts := make([]string, 0, len(result.Segments))
		for _, segment := range result.Segments {
			texts = append(texts, segment.Text)
		}
		hypotheses[key] = strings.Join(texts, " ")
		if res.version == "" {
			res.version = result.Version
		}
	}
	if err = jobRows.Err(); err != nil {
		return res, err
	}
	jobRows.Close()

	var pairs []transcriptPair
	for _, key := range order {
		if hypothesis, ok := hypotheses[key]; ok {
			pairs = append(pairs, transcriptPair{strings.Join(references[key], " "), hypothesis})
		}
	}
	res.scored = len(pairs)
	if err = q.QueryRow(ctx, `SELECT extract(epoch FROM max(j.updated_at)-min(j.created_at)) FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id WHERE rj.run_id=$1 AND rj.model_id=$2 AND j.status='done'`, runID, model).Scan(&res.elapsed); err != nil {
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
	if res.metrics, err = json.Marshal(evaluateTranscripts(pairs)); err != nil {
		return res, err
	}
	return res, nil
}
