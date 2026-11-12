package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"snitch/internal/segmentparts"
)

// ErrStaleResult marks a voice result computed against a diarization that has
// since been replaced.
var ErrStaleResult = errors.New("stale voice result")

// VoiceInputVersion returns the pipeline version embedded in a voice step's
// input version ("d5-a2-pvoice-xxxx", "f5-a2-pvoice-xxxx-d12", or "f5-a2-net-pvoice-xxxx-d12" for a recalculation on
// the net speech, #395: its marker comes before the pipeline, never after the diarization).
func VoiceInputVersion(inputVersion string) (pipeline string, diarizationJobID int64, ok bool) {
	marker := strings.Index(inputVersion, "-p")
	if marker < 0 {
		return "", 0, false
	}
	pipeline = inputVersion[marker+2:]
	if cut := strings.LastIndex(pipeline, "-d"); cut >= 0 && strings.HasPrefix(inputVersion, "f5-") {
		id, err := strconv.ParseInt(pipeline[cut+2:], 10, 64)
		if err != nil {
			return "", 0, false
		}
		return pipeline[:cut], id, true
	}
	return pipeline, 0, true
}

type voiceInputPayload struct {
	PipelineVersion  string          `json:"pipeline_version"`
	Config           json.RawMessage `json:"config"`
	Diarization      json.RawMessage `json:"diarization,omitempty"`
	FingerprintModel json.RawMessage `json:"fingerprint_model,omitempty"`
}

// voiceInput gives a leased voice step its pinned configuration and, for
// fingerprints, the speakers and turns it must embed.
func (s *Service) voiceInput(ctx context.Context, input *Input) error {
	pipeline, diarizationID, ok := VoiceInputVersion(input.Job.InputVersion)
	if !ok {
		return fmt.Errorf("voice input has no pipeline version")
	}
	payload := voiceInputPayload{PipelineVersion: pipeline}
	if err := s.pool.QueryRow(ctx, `SELECT config FROM voice_pipeline_versions WHERE version=$1`, pipeline).Scan(&payload.Config); err != nil {
		return fmt.Errorf("read voice pipeline snapshot: %w", err)
	}
	if diarizationID > 0 {
		// The embedding model is independent of the diarization engine. Send the
		// exact catalog reference, including for candidate benchmark jobs.
		var config map[string]json.RawMessage
		if err := json.Unmarshal(payload.Config, &config); err != nil {
			return fmt.Errorf("decode voice pipeline snapshot: %w", err)
		}
		if string(config["provider"]) != `"fake"` {
			var reference []byte
			err := s.pool.QueryRow(ctx, `SELECT jsonb_build_object('repo',c.repo,'revision',c.revision,'engine',c.engine)
FROM voice_model_catalog c WHERE c.id=COALESCE((SELECT requested_model_id FROM jobs WHERE id=$1),(SELECT model_id FROM voice_active_model WHERE kind='fingerprint'))`, input.Job.ID).Scan(&reference)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("read fingerprint catalog reference: %w", err)
			}
			if err == nil {
				payload.FingerprintModel = reference
			}
		}
		var raw []byte
		err := s.pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('label',ls.label,'turns',COALESCE((SELECT jsonb_agg(jsonb_build_object('index',st.turn_index,'start',st.start_seconds,'end',st.end_seconds) ORDER BY st.turn_index) FROM speaker_turns st WHERE st.local_speaker_id=ls.id),'[]'::jsonb)) ORDER BY ls.ordinal),'[]'::jsonb) FROM local_speakers ls WHERE ls.diarization_job_id=$1`, diarizationID).Scan(&raw)
		if err != nil {
			return fmt.Errorf("read diarization for fingerprints: %w", err)
		}
		if offset, offsetErr := FingerprintTrackOffset(ctx, s.pool, input.Job.ID, diarizationID); offsetErr != nil {
			return fmt.Errorf("read plan track offset: %w", offsetErr)
		} else if offset != 0 {
			// Stored turns are in clip time; the worker reads the stream from its own origin.
			var speakers []map[string]any
			if err = json.Unmarshal(raw, &speakers); err != nil {
				return fmt.Errorf("decode diarization for fingerprints: %w", err)
			}
			for _, speaker := range speakers {
				turns, _ := speaker["turns"].([]any)
				for _, turn := range turns {
					if m, ok := turn.(map[string]any); ok {
						for _, key := range []string{"start", "end"} {
							if v, isNumber := m[key].(float64); isNumber {
								m[key] = v - offset
							}
						}
					}
				}
			}
			if raw, err = json.Marshal(speakers); err != nil {
				return err
			}
		}
		diarization, _ := json.Marshal(map[string]any{"job_id": diarizationID, "speakers": json.RawMessage(raw)})
		payload.Diarization = diarization
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	input.Voice = encoded
	return nil
}

// PlanTrackOffset is the start offset, in seconds, of the audio stream a plan child works on, as
// the plan inventory recorded it. A worker measures a stream from its own first sample while the
// transcript manifest of a plan is in clip time: voice times are converted exactly once, on
// storage (add) and on the way back to a worker (subtract). A job that is not a bound child, or
// a stream whose offset is unknown, has none.
func PlanTrackOffset(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, jobID int64) (float64, error) {
	var offset float64
	err := q.QueryRow(ctx, `SELECT COALESCE((SELECT (t->>'start_seconds')::float8 FROM jsonb_array_elements(p.input->'inventory') t WHERE (t->>'index')::int=c.track_index),0)
FROM audio_plan_children c JOIN audio_plans p ON p.id=c.plan_id WHERE c.job_id=$1`, jobID).Scan(&offset)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return offset, err
}

// FingerprintTrackOffset is the stream offset of a fingerprint job: its own as a bound plan child, otherwise the one of
// the diarization it embeds. A fingerprint job outside the plan (a recalculation on the net speech, #395) reads the
// same stream as that diarization, whose stored turns are in clip time.
func FingerprintTrackOffset(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, jobID, diarizationJobID int64) (float64, error) {
	var offset float64
	err := q.QueryRow(ctx, `SELECT COALESCE((SELECT (t->>'start_seconds')::float8 FROM jsonb_array_elements(p.input->'inventory') t WHERE (t->>'index')::int=c.track_index),0)
FROM audio_plan_children c JOIN audio_plans p ON p.id=c.plan_id WHERE c.job_id IN ($1,$2) ORDER BY c.job_id=$1 DESC LIMIT 1`, jobID, diarizationJobID).Scan(&offset)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return offset, err
}

// SeparationInputVersion is the input version of a chain's separation task: it names the diarization and the
// transcription tasks it reads (#402). The scheduling query of httpapi builds the same text in SQL.
func SeparationInputVersion(diarizationID, transcriptionID int64) string {
	return fmt.Sprintf("bs-d%d-t%d", diarizationID, transcriptionID)
}

// ParseSeparationInputVersion reads the two tasks a SeparationInputVersion names.
func ParseSeparationInputVersion(version string) (diarizationID, transcriptionID int64, err error) {
	_, err = fmt.Sscanf(version, "bs-d%d-t%d", &diarizationID, &transcriptionID)
	return diarizationID, transcriptionID, err
}

// BenchmarkSpeaker is a speaker of the result of a benchmark diarization task, its turns in the time of the track.
type BenchmarkSpeaker struct {
	Label string `json:"label"`
	Turns []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"turns"`
}

// benchmarkSeparationInput gives a separation task of an end-to-end chain (#402) what it reads, in the time of the
// track: the turns of the chain's diarization task as the speakers to fingerprint on their net speech (production
// pipeline and active fingerprint model), the segments of its transcription task, the transcription model to
// transcribe the sources with, and the requested separation model. Nothing of the library is read.
func (s *Service) benchmarkSeparationInput(ctx context.Context, input *Input) error {
	diarizationID, transcriptionID, err := ParseSeparationInputVersion(input.Job.InputVersion)
	if err != nil {
		return fmt.Errorf("separation input version: %w", err)
	}
	var repo, revision string
	var speakersRaw, segments []byte
	var margin float64
	err = s.pool.QueryRow(ctx, `SELECT c.repo,c.revision,t.requested_model_id,d.result->'speakers',t.result->'segments',(SELECT ambiguity_margin FROM voice_matching_settings WHERE id=true)
FROM jobs j JOIN voice_model_catalog c ON c.id=j.requested_model_id, jobs d, jobs t WHERE j.id=$1 AND d.id=$2 AND t.id=$3 AND d.status='done' AND t.status='done'`,
		input.Job.ID, diarizationID, transcriptionID).Scan(&repo, &revision, &input.TranscriptionModel, &speakersRaw, &segments, &margin)
	if err != nil {
		return fmt.Errorf("read separation task of a chain: %w", err)
	}
	var speakers []BenchmarkSpeaker
	if err = json.Unmarshal(speakersRaw, &speakers); err != nil {
		return fmt.Errorf("decode chain diarization: %w", err)
	}
	// The fingerprint code reads the turns with their index, as stored for a production diarization.
	indexed := make([]map[string]any, 0, len(speakers))
	for _, sp := range speakers {
		turns := make([]map[string]any, 0, len(sp.Turns))
		for i, t := range sp.Turns {
			turns = append(turns, map[string]any{"index": i, "start": t.Start, "end": t.End})
		}
		indexed = append(indexed, map[string]any{"label": sp.Label, "turns": turns})
	}
	payload := voiceInputPayload{}
	if err = s.pool.QueryRow(ctx, `SELECT a.version,v.config FROM voice_pipeline_active a JOIN voice_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&payload.PipelineVersion, &payload.Config); err != nil {
		return fmt.Errorf("read active voice pipeline for a separation benchmark: %w", err)
	}
	var reference []byte
	err = s.pool.QueryRow(ctx, `SELECT jsonb_build_object('repo',c.repo,'revision',c.revision,'engine',c.engine) FROM voice_model_catalog c WHERE c.id=(SELECT model_id FROM voice_active_model WHERE kind='fingerprint')`).Scan(&reference)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read active fingerprint model: %w", err)
	}
	payload.FingerprintModel = reference
	if payload.Diarization, err = json.Marshal(map[string]any{"job_id": diarizationID, "speakers": indexed}); err != nil {
		return err
	}
	if input.Voice, err = json.Marshal(payload); err != nil {
		return err
	}
	// The speakers without words, and the times of each one's words, come from the production attribution rule, so the
	// Worker has no rule of its own (#403).
	var decoded []BenchmarkSegment
	if err = json.Unmarshal(segments, &decoded); err != nil {
		return fmt.Errorf("decode chain transcription: %w", err)
	}
	turns, parts := BenchmarkTrack(speakers, decoded)
	byVoice := segmentparts.Spans(segmentparts.RuleWord, parts, turns)
	spans := []map[string]any{}
	for _, sp := range speakers {
		for _, w := range byVoice[sp.Label] {
			spans = append(spans, map[string]any{"speaker": sp.Label, "start": w.Start, "end": w.End})
		}
	}
	input.Separation, err = json.Marshal(map[string]any{"model": repo, "revision": revision, "ambiguity_margin": margin, "segments": json.RawMessage(segments),
		"silent_speakers": segmentparts.Silent(segmentparts.RuleWord, parts, turns), "word_spans": spans})
	return err
}

// BenchmarkSegment is a segment of the result of a benchmark transcription task, with its timed words.
type BenchmarkSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
	Words []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Word  string  `json:"word"`
	} `json:"words"`
}

// BenchmarkTrack reads a chain's diarization speakers and transcription segments of one track as the turns and
// segments of the production attribution rule (segmentparts, ADR 0065).
func BenchmarkTrack(speakers []BenchmarkSpeaker, segments []BenchmarkSegment) ([]segmentparts.Turn[string], []segmentparts.Segment) {
	var turns []segmentparts.Turn[string]
	for _, sp := range speakers {
		for _, t := range sp.Turns {
			turns = append(turns, segmentparts.Turn[string]{Speaker: sp.Label, Start: t.Start, End: t.End})
		}
	}
	parts := make([]segmentparts.Segment, 0, len(segments))
	for _, s := range segments {
		words := make([]segmentparts.Word, 0, len(s.Words))
		for _, w := range s.Words {
			words = append(words, segmentparts.Word{Start: w.Start, End: w.End})
		}
		parts = append(parts, segmentparts.Segment{Span: segmentparts.Span{Start: s.Start, End: s.End}, Words: words})
	}
	return turns, parts
}

// benchmarkDiarizationInput gives a diarization benchmark task the production pipeline configuration with the
// requested catalogue model in place of the active one (ADR 0059). The provider is the model's engine, so a
// Sortformer task runs on the NeMo adapter whatever the production model is. A fake provider (tests) stays fake.
func (s *Service) benchmarkDiarizationInput(ctx context.Context, input *Input) error {
	var version string
	var raw []byte
	if err := s.pool.QueryRow(ctx, `SELECT a.version,v.config FROM voice_pipeline_active a JOIN voice_pipeline_versions v ON v.version=a.version WHERE a.id=true`).Scan(&version, &raw); err != nil {
		return fmt.Errorf("read active voice pipeline for a diarization benchmark: %w", err)
	}
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("decode voice pipeline: %w", err)
	}
	if config["provider"] != "fake" {
		var engine, repo, revision string
		err := s.pool.QueryRow(ctx, `SELECT c.engine,c.repo,c.revision FROM voice_model_catalog c WHERE c.id=(SELECT requested_model_id FROM jobs WHERE id=$1)`, input.Job.ID).Scan(&engine, &repo, &revision)
		if err != nil {
			return fmt.Errorf("read requested diarization model: %w", err)
		}
		config["provider"], config["model"], config["model_version"] = engine, repo, revision
	}
	encoded, err := json.Marshal(map[string]any{"pipeline_version": version, "config": config})
	if err != nil {
		return err
	}
	input.Voice = encoded
	return nil
}
