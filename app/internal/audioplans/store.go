package audioplans

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"snitch/internal/httpapi/contract"
)

// DB accepts a pool or transaction. Pass a transaction when creating and binding
// jobs so no worker can claim an unbound child. The caller owns commit/rollback.
type DB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Store struct{ db DB }

func New(db DB) *Store { return &Store{db} }

var ErrConflict = errors.New("audio identity conflicts with persisted value")

func (s *Store) CreatePlan(ctx context.Context, runID int64, input Input) (Plan, error) {
	c, err := CanonicalInput(input)
	if err != nil {
		return Plan{}, err
	}
	// One statement makes the plan and its selected tracks atomic even on a pool.
	var p Plan
	var raw []byte
	err = s.db.QueryRow(ctx, `WITH inserted AS (
 INSERT INTO audio_plans(run_id,clip_id,input_version,input)
 SELECT id,clip_id,$2,$3::jsonb FROM clip_analysis_runs WHERE id=$1
 ON CONFLICT(run_id,input_version) DO NOTHING RETURNING *), tracks AS (
 INSERT INTO audio_plan_tracks(plan_id,track_index) SELECT id,value::int FROM inserted CROSS JOIN LATERAL jsonb_array_elements_text(input->'track_indices') ON CONFLICT DO NOTHING)
 SELECT id,run_id,clip_id,input_version,created_at,input FROM inserted`, runID, c.Version, c.JSON).Scan(&p.Id, &p.RunId, &p.ClipId, &p.InputVersion, &p.CreatedAt, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.db.QueryRow(ctx, `SELECT id,run_id,clip_id,input_version,created_at,input FROM audio_plans WHERE run_id=$1 AND input_version=$2`, runID, c.Version).Scan(&p.Id, &p.RunId, &p.ClipId, &p.InputVersion, &p.CreatedAt, &raw)
	}
	if err == nil {
		err = json.Unmarshal(raw, &p.Input)
	}
	return p, err
}
func (s *Store) GetPlan(ctx context.Context, id int64) (State, error) {
	out := State{Children: []Child{}, Diagnostics: []Diagnostic{}}
	var raw []byte
	p := &out.Plan
	err := s.db.QueryRow(ctx, `SELECT id,run_id,clip_id,input_version,created_at,input FROM audio_plans WHERE id=$1`, id).Scan(&p.Id, &p.RunId, &p.ClipId, &p.InputVersion, &p.CreatedAt, &raw)
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(raw, &p.Input); err != nil {
		return out, err
	}
	rows, err := s.db.Query(ctx, `SELECT DISTINCT ON(track_index) payload FROM audio_track_diagnostics WHERE plan_id=$1 ORDER BY track_index,attempt DESC`, id)
	if err != nil {
		return out, err
	}
	diagnostics := map[int]Diagnostic{}
	for rows.Next() {
		var b []byte
		var d Diagnostic
		if err = rows.Scan(&b); err == nil {
			err = json.Unmarshal(b, &d)
		}
		if err != nil {
			rows.Close()
			return out, err
		}
		diagnostics[d.TrackIndex] = d
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	for _, track := range p.Input.TrackIndices {
		d, ok := diagnostics[track]
		if !ok {
			d = Diagnostic{PlanId: id, TrackIndex: track, InputVersion: p.InputVersion, Status: "unexamined", Cause: "not_started", DetectorVersion: p.Input.DetectorVersion, DetectorConfigVersion: p.Input.DetectorConfigVersion, Intervals: []contract.AudioSpeechInterval{}}
		}
		out.Diagnostics = append(out.Diagnostics, d)
	}
	rows, err = s.db.Query(ctx, `SELECT c.track_index,c.stage,c.job_id,c.input_version,CASE j.status WHEN 'pending' THEN 'queued' WHEN 'leased' THEN 'running' WHEN 'retry_wait' THEN 'retry' WHEN 'failed' THEN 'dead' ELSE j.status END,j.attempts FROM audio_plan_children c JOIN jobs j ON j.id=c.job_id WHERE c.plan_id=$1 ORDER BY c.track_index,c.stage`, id)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		c := Child{PlanId: id}
		if err = rows.Scan(&c.TrackIndex, &c.Stage, &c.JobId, &c.InputVersion, &c.Status, &c.Attempt); err != nil {
			return out, err
		}
		out.Children = append(out.Children, c)
	}
	return out, rows.Err()
}

// BindChild must run in the same transaction as job creation, before dispatch.
// Each stage/track owns one durable job, whose attempts implement retry. A done
// legacy job cannot be adopted (that would hide an already published transcript).
func (s *Store) BindChild(ctx context.Context, c Child) error {
	if c.PlanId < 1 || c.TrackIndex < 0 || c.JobId < 1 || c.InputVersion == "" {
		return ErrConflict
	}
	var existing int64
	err := s.db.QueryRow(ctx, `SELECT job_id FROM audio_plan_children WHERE plan_id=$1 AND track_index=$2 AND stage=$3 AND input_version=$4`, c.PlanId, c.TrackIndex, c.Stage, c.InputVersion).Scan(&existing)
	if err == nil {
		if existing == c.JobId {
			return nil
		}
		return ErrConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	tag, err := s.db.Exec(ctx, `INSERT INTO audio_plan_children(plan_id,track_index,stage,job_id,input_version)
 SELECT p.id,$2,$3,j.id,$5 FROM audio_plans p JOIN audio_plan_tracks t ON t.plan_id=p.id AND t.track_index=$2 JOIN jobs j ON j.clip_id=p.clip_id
 WHERE p.id=$1 AND j.id=$4 AND j.input_version=$5 AND j.status='pending'
 AND NOT EXISTS(SELECT 1 FROM transcript_revisions WHERE legacy_job_id=j.id)
 AND (($3='transcription' AND j.step='transcription.audio.'||$2::text)
 OR ($3='diagnostic' AND j.step='audio.diagnostic.'||$2::text)
 OR ($3='diarization' AND j.step='voice.diarization.'||$2::text)
 OR ($3='fingerprint' AND j.step='voice.fingerprint.'||$2::text))
 ON CONFLICT DO NOTHING`, c.PlanId, c.TrackIndex, c.Stage, c.JobId, c.InputVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	// Concurrent identical binding is also a successful retry.
	err = s.db.QueryRow(ctx, `SELECT job_id FROM audio_plan_children WHERE plan_id=$1 AND track_index=$2 AND stage=$3 AND input_version=$4`, c.PlanId, c.TrackIndex, c.Stage, c.InputVersion).Scan(&existing)
	if err == nil && existing == c.JobId {
		return nil
	}
	return ErrConflict
}

// RecordDiagnostic appends a terminal observation for one job attempt. Failed or
// ambiguous observations are never silently converted into no_speech. Retrying a
// job records a higher attempt; old observations remain immutable.
func (s *Store) RecordDiagnostic(ctx context.Context, d Diagnostic) error {
	state, err := s.GetPlan(ctx, d.PlanId)
	if err != nil {
		return err
	}
	if d.InputVersion != state.Plan.InputVersion || d.DetectorVersion != state.Plan.Input.DetectorVersion || d.DetectorConfigVersion != state.Plan.Input.DetectorConfigVersion || d.JobId == nil || d.Attempt < 1 || d.Cause == "" {
		return ErrConflict
	}
	switch d.Status {
	case "speech":
		if len(d.Intervals) == 0 {
			return ErrConflict
		}
	case "no_speech":
		if len(d.Intervals) != 0 {
			return ErrConflict
		}
	case "ambiguous", "failed":
	default:
		return ErrConflict
	}
	last := float64(-1)
	for _, v := range d.Intervals {
		if !finite(v.Start) || !finite(v.End) || v.Start < 0 || v.End <= v.Start || v.Start < last {
			return ErrConflict
		}
		last = v.End
	}
	if d.Intervals == nil {
		d.Intervals = []contract.AudioSpeechInterval{}
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `INSERT INTO audio_track_diagnostics(plan_id,track_index,attempt,payload)
 SELECT c.plan_id,c.track_index,$3,$4::jsonb FROM audio_plan_children c JOIN jobs j ON j.id=c.job_id
 WHERE c.plan_id=$1 AND c.track_index=$2 AND c.stage='diagnostic' AND c.job_id=$5 AND j.attempts=$3
 AND j.status IN ('leased','done','failed','dead','retry_wait') ON CONFLICT DO NOTHING`, d.PlanId, d.TrackIndex, d.Attempt, raw, *d.JobId)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var same bool
	err = s.db.QueryRow(ctx, `SELECT payload=$4::jsonb FROM audio_track_diagnostics WHERE plan_id=$1 AND track_index=$2 AND attempt=$3`, d.PlanId, d.TrackIndex, d.Attempt, raw).Scan(&same)
	if err == nil && same {
		return nil
	}
	return ErrConflict
}
func (s *Store) GetRevision(ctx context.Context, id string) (Revision, error) {
	var raw []byte
	var r Revision
	err := s.db.QueryRow(ctx, `SELECT manifest FROM transcript_revisions WHERE id=$1`, id).Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &r)
	}
	return r, err
}
