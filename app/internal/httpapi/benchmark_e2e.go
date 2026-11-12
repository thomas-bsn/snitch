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
	"snitch/internal/segmentparts"
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
	cpwerMetrics // segment rule
	// WordRule is the same measure with the word rule (#400); absent from the runs measured before.
	WordRule cpwerMetrics `json:"word_rule"`
	DER      *float64     `json:"der"`
	// Separation is present for a chain with a separation model only.
	Separation *separationMetrics `json:"separation,omitempty"`
}

// computeE2EResult measures a chain on the clips and tracks where both of its models finished: the transcript is
// attributed to the speakers by production's segment rule, then by its word rule (ADR 0065), and the
// speaker-attributed word error rate of each compares it with the frozen reference, whose segments carry their speaker. The DER of the same turns is reported too.
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
	var separated separationMetrics
	var withoutClips, withClips []attributedClip
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
	var clips, wordClips []attributedClip
	var turnClips []clipTurns
	version := ""
	for _, key := range order {
		rawT, okT := transcripts[key]
		rawD, okD := diarizations[key]
		if !okT || !okD {
			continue
		}
		var tr struct {
			Version  string                  `json:"version"`
			Segments []jobs.BenchmarkSegment `json:"segments"`
		}
		var di struct {
			Speakers []jobs.BenchmarkSpeaker `json:"speakers"`
		}
		if json.Unmarshal(rawT, &tr) != nil || json.Unmarshal(rawD, &di) != nil {
			continue // a result that cannot be read is a task that did not measure
		}
		if version == "" {
			version = tr.Version
		}
		segments := make([]timedText, 0, len(tr.Segments))
		for _, s := range tr.Segments {
			seg := timedText{start: s.Start, end: s.End, text: s.Text}
			for _, w := range s.Words {
				seg.words = append(seg.words, timedWord{start: w.Start, end: w.End, text: w.Word})
			}
			segments = append(segments, seg)
		}
		var hypTurns []speakerTurn
		for _, sp := range di.Speakers {
			for _, t := range sp.Turns {
				hypTurns = append(hypTurns, speakerTurn{t.Start, t.End, sp.Label})
			}
		}
		hypothesis := attributeSegments(segmentparts.RuleSegment, segments, hypTurns)
		raw, ok := separations[jobs.SeparationInputVersion(jobIDs[jobs.BenchmarkDiarizationPrefix][key], jobIDs[jobs.BenchmarkTranscriptionPrefix][key])]
		if ok && hasOverlap(refTurns[key]) {
			separateTrack(&separated, &withoutClips, &withClips, raw, di.Speakers, tr.Segments, refSegments[key])
		}
		clips = append(clips, attributedClip{reference: refSegments[key], hypothesis: hypothesis})
		wordClips = append(wordClips, attributedClip{reference: refSegments[key], hypothesis: attributeSegments(segmentparts.RuleWord, segments, hypTurns)})
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
	metrics := e2eMetrics{cpwerMetrics: evaluateCPWER(clips), WordRule: evaluateCPWER(wordClips), DER: evaluateDiarization(turnClips).DER}
	if separations != nil {
		separated.MinClips = separationMinClips
		separated.WithoutSeparation, separated.WithSeparation = evaluateCPWER(withoutClips), evaluateCPWER(withClips)
		metrics.Separation = &separated
	}
	if res.metrics, err = json.Marshal(metrics); err != nil {
		return res, err
	}
	return res, nil
}

// hasOverlap tells whether two speakers of the reference speak at once on the track: a Chevauchement.
func hasOverlap(turns []speakerTurn) bool {
	for i, a := range turns {
		for _, b := range turns[i+1:] {
			if a.speaker != b.speaker && min(a.end, b.end) > max(a.start, b.start) {
				return true
			}
		}
	}
	return false
}

// separateTrack measures the separation result of one track with a Chevauchement. Both hypotheses are read word by
// word, each word given its voice by the word rule of production (segmentparts, ADR 0065), the rule that also lists the
// speakers without words, so the words added fall in their place in time; the one with separation adds them under the
// speaker they were given. The speakers without words are counted the same way on both: the voices of the turns that
// hold no word of the transcript (before, what segmentparts.Silent gives the task; after, on the merged transcript).
func separateTrack(m *separationMetrics, without, with *[]attributedClip, raw []byte, speakers []jobs.BenchmarkSpeaker, segments []jobs.BenchmarkSegment, reference []speakerText) {
	added, ok := m.add(raw, reference)
	if !ok {
		return
	}
	turns, parts := jobs.BenchmarkTrack(speakers, segments)
	var words []speakerText
	for i, s := range parts {
		given := segmentparts.Attribute(segmentparts.RuleWord, s.Span, s.Words, turns)
		if len(s.Words) == 0 {
			words = append(words, speakerText{s.Span.Start, s.Span.End, given[0].Speaker, segments[i].Text})
		}
		for _, p := range given {
			for _, w := range segments[i].Words[p.First : p.First+p.Count] {
				words = append(words, speakerText{w.Start, w.End, p.Speaker, w.Word})
			}
		}
	}
	merged := slices.Clone(words)
	for _, a := range added {
		merged = append(merged, speakerText{a.Start, a.End, a.Speaker, a.Word})
	}
	m.SpeakersWithoutWordsBefore += withoutWords(turns, words)
	m.SpeakersWithoutWordsAfter += withoutWords(turns, merged)
	*without = append(*without, attributedClip{reference, words})
	*with = append(*with, attributedClip{reference, merged})
}

// withoutWords counts the voices of turns that hold no word of texts. An added word keeps the voice the separation gave
// it: read by the word rule, a word of a Chevauchement would go to whichever voice comes first in a tie.
func withoutWords(turns []segmentparts.Turn[string], texts []speakerText) int {
	silent := map[string]bool{}
	for _, t := range turns {
		silent[t.Speaker] = true
	}
	for _, t := range texts {
		delete(silent, t.speaker)
	}
	return len(silent)
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

// separationMinClips is the number of clips with a Chevauchement under which the report does not conclude (#403).
const separationMinClips = 10

// separationMetrics measure a chain's separation (#402, #403) on the certified tracks whose reference contains a
// Chevauchement and whose separation task is done: the targeted zones, the Local speakers without any word on their
// track before and after the merge (production word rule), the separated sources, those left without a speaker, the
// words added, those absent from the reference and those repeated between two sources, and the cpWER and WER of
// the same tracks without and with the words added.
type separationMetrics struct {
	Tracks                        int          `json:"tracks"`
	Zones                         int          `json:"zones"`
	SpeakersWithoutWordsBefore    int          `json:"speakers_without_words_before"`
	SpeakersWithoutWordsAfter     int          `json:"speakers_without_words_after"`
	Sources                       int          `json:"sources"`
	SourcesWithoutSpeaker         int          `json:"sources_without_speaker"`
	WordsAdded                    int          `json:"words_added"`
	WordsAddedAbsentFromReference int          `json:"words_added_absent_from_reference"`
	WordsDuplicatedBetweenSources int          `json:"words_duplicated_between_sources"`
	MinClips                      int          `json:"min_clips"`
	WithoutSeparation             cpwerMetrics `json:"without_separation"`
	WithSeparation                cpwerMetrics `json:"with_separation"`
}

// separationResult is what a separation task stores: the Local speakers without words on the track, its zones,
// every source with the speaker it was given, and the words added to the original transcript, which is never changed.
type separationResult struct {
	AudioStreamIndex *int     `json:"audio_stream_index"`
	SilentSpeakers   []string `json:"silent_speakers"`
	Zones            []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"zones"`
	Sources []separatedSource `json:"sources"`
	Added   []addedWord       `json:"added"`
}

type separatedSource struct {
	Zone    int         `json:"zone"`
	Speaker *string     `json:"speaker"`
	Words   []addedWord `json:"words"` // only their times and text
}

type addedWord struct {
	Zone    int     `json:"zone"`
	Speaker string  `json:"speaker"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Word    string  `json:"word"`
}

// add counts the separation result of one track, given its reference segments, and returns the words it adds; false
// when the result cannot be read.
func (c *separationMetrics) add(raw []byte, reference []speakerText) ([]addedWord, bool) {
	var r separationResult
	if json.Unmarshal(raw, &r) != nil {
		return nil, false
	}
	c.Tracks++
	c.Zones += len(r.Zones)
	c.Sources += len(r.Sources)
	c.WordsAdded += len(r.Added)
	same := func(a, b addedWord) bool {
		return min(a.End, b.End) > max(a.Start, b.Start) && slices.Equal(normalizeWords(a.Word), normalizeWords(b.Word))
	}
	for i, s := range r.Sources {
		if s.Speaker == nil {
			c.SourcesWithoutSpeaker++
		}
		// A word an earlier source of the zone also says at the same time: one voice leaking into two sources.
		for _, w := range s.Words {
			if slices.ContainsFunc(r.Sources[:i], func(o separatedSource) bool {
				return o.Zone == s.Zone && slices.ContainsFunc(o.Words, func(x addedWord) bool { return same(w, x) })
			}) {
				c.WordsDuplicatedBetweenSources++
			}
		}
	}
	for _, a := range r.Added {
		// Absent: not among the words of the reference segments over its zone.
		var said []string
		if a.Zone >= 0 && a.Zone < len(r.Zones) {
			z := r.Zones[a.Zone]
			for _, s := range reference {
				if min(s.end, z.End) > max(s.start, z.Start) {
					said = append(said, normalizeWords(s.text)...)
				}
			}
		}
		for _, w := range normalizeWords(a.Word) {
			if !slices.Contains(said, w) {
				c.WordsAddedAbsentFromReference++
				break
			}
		}
	}
	return r.Added, true
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
