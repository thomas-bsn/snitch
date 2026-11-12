package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"snitch/internal/jobs"
)

// benchmarkChain is a diarization model and a transcription model measured together (ADR 0060). Its tasks are those
// of the two stages: chains that share a model share its tasks, so a model is never run twice on the same clip.
type benchmarkChain struct {
	ID                 string `json:"id"`
	DiarizationModelID string `json:"diarization_model_id"`
	TranscriptionModel string `json:"transcription_model"`
	// SeparationModelID, when set, adds a separation task per clip and track once the two others are done (#402).
	SeparationModelID string `json:"separation_model_id,omitempty"`
}

type e2eMetrics struct {
	cpwerMetrics
	DER *float64 `json:"der"`
	// Separation is present for a chain with a separation model only.
	Separation *separationCounts `json:"separation,omitempty"`
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

	jobIDs := map[string]map[transcriptKey]int64{} // by step prefix: the done task a result comes from
	taskResults := func(model, prefix string) (map[transcriptKey][]byte, error) {
		out := map[transcriptKey][]byte{}
		jobIDs[prefix] = map[transcriptKey]int64{}
		jr, err := q.Query(ctx, `SELECT j.id,j.clip_id,substring(j.step FROM '[0-9]+$')::int,j.result FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id
			WHERE rj.run_id=$1 AND rj.model_id=$2 AND j.step LIKE $3 AND j.status='done' AND j.result IS NOT NULL`, runID, model, prefix+"%")
		if err != nil {
			return nil, err
		}
		defer jr.Close()
		for jr.Next() {
			var id int64
			var key transcriptKey
			var raw []byte
			if err = jr.Scan(&id, &key.clip, &key.track, &raw); err != nil {
				return nil, err
			}
			out[key] = raw
			jobIDs[prefix][key] = id
		}
		return out, jr.Err()
	}
	transcripts, err := taskResults(chain.TranscriptionModel, jobs.BenchmarkTranscriptionPrefix)
	if err != nil {
		return res, err
	}
	diarizations, err := taskResults(chain.DiarizationModelID, jobs.BenchmarkDiarizationPrefix)
	if err != nil {
		return res, err
	}

	// The separation results of the chain, by the input version naming its two tasks of a clip and track.
	var separations map[string][]byte
	var separated separationCounts
	if chain.SeparationModelID != "" {
		separations = map[string][]byte{}
		sr, err := q.Query(ctx, `SELECT j.input_version,j.result FROM benchmark_run_jobs rj JOIN jobs j ON j.id=rj.job_id
			WHERE rj.run_id=$1 AND rj.model_id=$2 AND j.step LIKE 'benchmark.separation.%' AND j.status='done' AND j.result IS NOT NULL`, runID, chain.SeparationModelID)
		if err != nil {
			return res, err
		}
		for sr.Next() {
			var version string
			var raw []byte
			if err = sr.Scan(&version, &raw); err != nil {
				sr.Close()
				return res, err
			}
			separations[version] = raw
		}
		sr.Close()
		if err = sr.Err(); err != nil {
			return res, err
		}
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
		if separations != nil {
			if raw, ok := separations[jobs.SeparationInputVersion(jobIDs[jobs.BenchmarkDiarizationPrefix][key], jobIDs[jobs.BenchmarkTranscriptionPrefix][key])]; ok {
				separated.add(raw)
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
	if separations != nil {
		metrics.Separation = &separated
	}
	if res.metrics, err = json.Marshal(metrics); err != nil {
		return res, err
	}
	return res, nil
}

// scheduleSeparationTasks schedules, for each chain with a separation model, one separation task per clip and track
// whose diarization and transcription tasks of the chain are done and that has none yet in this run. Its input version
// names those two tasks; it asks for the separation model, so only a Worker holding it ready takes it. It tells
// whether it scheduled any.
func (s *Server) scheduleSeparationTasks(ctx context.Context, tx pgx.Tx, runID int64, params []byte) (bool, error) {
	var p struct {
		Chains []benchmarkChain `json:"chains"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return false, err
	}
	type pending struct {
		clip                       int64
		track                      string
		diarization, transcription int64
	}
	scheduled := false
	for _, c := range p.Chains {
		if c.SeparationModelID == "" {
			continue
		}
		// The input version of the NOT EXISTS is jobs.SeparationInputVersion, built in SQL.
		rows, err := tx.Query(ctx, `SELECT d.clip_id,substring(d.step FROM '[0-9]+$'),d.id,t.id
			FROM benchmark_run_jobs rd JOIN jobs d ON d.id=rd.job_id
			JOIN benchmark_run_jobs rt ON rt.run_id=rd.run_id AND rt.model_id=$3
			JOIN jobs t ON t.id=rt.job_id AND t.clip_id=d.clip_id AND t.step='benchmark.transcription.'||substring(d.step FROM '[0-9]+$')
			WHERE rd.run_id=$1 AND rd.model_id=$2 AND d.step LIKE 'benchmark.diarization.%' AND d.status='done' AND t.status='done'
			  AND NOT EXISTS (SELECT 1 FROM benchmark_run_jobs rs JOIN jobs sj ON sj.id=rs.job_id WHERE rs.run_id=$1 AND rs.model_id=$4 AND sj.input_version='bs-d'||d.id||'-t'||t.id)
			ORDER BY 1,2`, runID, c.DiarizationModelID, c.TranscriptionModel, c.SeparationModelID)
		if err != nil {
			return false, err
		}
		var works []pending
		for rows.Next() {
			var w pending
			if err = rows.Scan(&w.clip, &w.track, &w.diarization, &w.transcription); err != nil {
				rows.Close()
				return false, err
			}
			works = append(works, w)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return false, err
		}
		for _, w := range works {
			job, _, err := s.Jobs.Schedule(ctx, jobs.ScheduleInput{ClipID: w.clip, Step: jobs.BenchmarkSeparationPrefix + w.track, InputVersion: jobs.SeparationInputVersion(w.diarization, w.transcription), RequestedModelID: c.SeparationModelID})
			if err != nil {
				return false, err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO benchmark_run_jobs(run_id,job_id,model_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, runID, job.ID, c.SeparationModelID); err != nil {
				return false, err
			}
			scheduled = true
		}
	}
	return scheduled, nil
}

// separationCounts are the raw counts of a chain's separation tasks (#402), summed over its measured tracks: the
// targeted zones, the Local speakers without words there before and after the merge, the separated sources, those
// left without a speaker, and the words added.
type separationCounts struct {
	Tracks                     int `json:"tracks"`
	Zones                      int `json:"zones"`
	SpeakersWithoutWordsBefore int `json:"speakers_without_words_before"`
	SpeakersWithoutWordsAfter  int `json:"speakers_without_words_after"`
	Sources                    int `json:"sources"`
	SourcesWithoutSpeaker      int `json:"sources_without_speaker"`
	WordsAdded                 int `json:"words_added"`
}

// separationResult is what a separation task stores: its zones, every source with the speaker it was given, and the
// words added to the original transcript, which is never changed.
type separationResult struct {
	AudioStreamIndex *int `json:"audio_stream_index"`
	Zones            []struct {
		Start          float64  `json:"start"`
		End            float64  `json:"end"`
		SilentSpeakers []string `json:"silent_speakers"`
	} `json:"zones"`
	Sources []struct {
		Zone    int     `json:"zone"`
		Speaker *string `json:"speaker"`
	} `json:"sources"`
	Added []addedWord `json:"added"`
}

type addedWord struct {
	Zone    int     `json:"zone"`
	Speaker string  `json:"speaker"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Word    string  `json:"word"`
}

func (c *separationCounts) add(raw []byte) {
	var r separationResult
	if json.Unmarshal(raw, &r) != nil {
		return
	}
	c.Tracks++
	c.Zones += len(r.Zones)
	c.Sources += len(r.Sources)
	c.WordsAdded += len(r.Added)
	for _, s := range r.Sources {
		if s.Speaker == nil {
			c.SourcesWithoutSpeaker++
		}
	}
	for i, z := range r.Zones {
		c.SpeakersWithoutWordsBefore += len(z.SilentSpeakers)
		for _, speaker := range z.SilentSpeakers {
			if !slices.ContainsFunc(r.Added, func(a addedWord) bool { return a.Zone == i && a.Speaker == speaker }) {
				c.SpeakersWithoutWordsAfter++
			}
		}
	}
}

// validSeparationResult checks a separation result of the track of its step: times, words and speakers well formed.
func validSeparationResult(raw []byte, step string) bool {
	var r separationResult
	track, err := strconv.Atoi(strings.TrimPrefix(step, jobs.BenchmarkSeparationPrefix))
	if err != nil || json.Unmarshal(raw, &r) != nil || r.AudioStreamIndex == nil || *r.AudioStreamIndex != track || r.Zones == nil || r.Sources == nil || r.Added == nil {
		return false
	}
	for _, s := range r.Sources {
		if s.Zone < 0 || s.Zone >= len(r.Zones) {
			return false
		}
	}
	for _, a := range r.Added {
		if a.Zone < 0 || a.Zone >= len(r.Zones) || a.Speaker == "" || strings.TrimSpace(a.Word) == "" || !validTime(a.Start) || !validTime(a.End) || a.End < a.Start {
			return false
		}
	}
	return true
}
