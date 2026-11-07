//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
)

// The durable engine is a contract fixture here: routes, auth, inventory, receipt,
// immutable plans and transaction rollback all use the real HTTP/PostgreSQL stack.
type selectionEngineFixture struct {
	h           *voiceHarness
	diagnostics []contract.AudioSelectionDiagnostic
	fail        bool
}

func (f *selectionEngineFixture) Context(context.Context, pgx.Tx, int64) (contract.AudioSelectionContext, error) {
	var input audioplans.Input
	_ = json.Unmarshal(audioplans.ContractFixture(), &input)
	return contract.AudioSelectionContext{Input: &input, StateVersion: "synthetic-engine-v1", Diagnostics: f.diagnostics}, nil
}
func (f *selectionEngineFixture) Preview(ctx context.Context, tx pgx.Tx, id int64, input audioplans.Input, _ contract.AudioAnalysisChoice) (contract.AudioSelectionEffects, error) {
	stake, _, err := f.h.api.clipCorrections(ctx, tx, id)
	if err != nil {
		return contract.AudioSelectionEffects{}, err
	}
	raw, _ := json.Marshal(stake)
	steps := []string{}
	for _, i := range input.TrackIndices {
		steps = append(steps, fmt.Sprintf("audio.diagnostic.%d", i), fmt.Sprintf("transcription.audio.%d", i))
	}
	return contract.AudioSelectionEffects{AddedSteps: steps, InvalidatedSteps: []string{"indexing"}, TrackIndices: input.TrackIndices, StakeVersion: string(raw), Losses: contract.AudioLossPreview{ExpectedRevisionId: fmt.Sprintf("legacy:%d", stake.TranscriptionJobID), CorrectionCount: stake.TextSegments + stake.SpeakerSegments, InvalidatedSteps: []string{"indexing"}, ReplacedTrackIndices: input.TrackIndices}}, nil
}
func (f *selectionEngineFixture) Start(ctx context.Context, tx pgx.Tx, p audioplans.Plan, _ contract.AudioAnalysisChoice, _ contract.AudioSelectionEffects) error {
	for _, i := range p.Input.TrackIndices {
		var id int64
		if err := tx.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version) VALUES($1,$2,$3) RETURNING id`, p.ClipId, fmt.Sprintf("audio.diagnostic.%d", i), p.InputVersion).Scan(&id); err != nil {
			return err
		}
		if err := audioplans.New(tx).BindChild(ctx, audioplans.Child{PlanId: p.Id, TrackIndex: i, Stage: "diagnostic", JobId: id, InputVersion: p.InputVersion}); err != nil {
			return err
		}
	}
	if f.fail {
		return errors.New("synthetic engine unavailable")
	}
	return nil
}
func testAudioSelection(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	path := fmt.Sprintf("/api/admin/clips/%d/audio-selection", h.clipID)
	var current contract.AudioSelectionContext
	load := func() {
		t.Helper()
		raw := h.adminCall("GET", path, nil, 200)
		assertAudioContract(t, "AudioSelectionContext", raw)
		if err := json.Unmarshal(raw, &current); err != nil {
			t.Fatal(err)
		}
	}
	load()
	if current.Enabled {
		t.Fatal("production gate enabled")
	}
	h.adminCall("POST", path+"/preview", map[string]any{}, 503)
	h.transcribe()
	h.adminCall("PUT", segmentTextPath(h.clipID, 0), map[string]any{"text": "human edit", "expected_version": 0}, 200)
	// Noncontiguous real indices, with video interleaved and no language/order heuristic.
	_, err := pool.Exec(context.Background(), `UPDATE clips SET tracks=$2 WHERE id=$1`, h.clipID, `[{"index":0,"type":"video"},{"index":2,"type":"audio","title":"Desktop","codec":"aac","channels":2,"language":"fr"},{"index":7,"type":"audio","title":"Microphone","start_seconds":1.25,"channels":1}]`)
	if err != nil {
		t.Fatal(err)
	}
	f := &selectionEngineFixture{h: h}
	h.api.AudioSelection = f
	load()
	if !current.Enabled || len(current.Inventory) != 2 || current.Inventory[1].Index != 7 || current.Inventory[0].Codec != "aac" {
		t.Fatal(current)
	}
	choice := contract.AudioSelectionRequest{Version: current.Version, Mode: "manual", TrackIndices: []int{2}, Decisions: []contract.AudioSelectionDecision{}, Analysis: contract.AudioAnalysisChoice{Scope: "full", Mode: "missing", Steps: []string{}}}
	preview := func(c contract.AudioSelectionRequest) contract.AudioSelectionPreview {
		t.Helper()
		raw := h.adminCall("POST", path+"/preview", c, 200)
		assertAudioContract(t, "AudioSelectionPreview", raw)
		var p contract.AudioSelectionPreview
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatal(err)
		}
		return p
	}
	start := func(c contract.AudioSelectionRequest, p contract.AudioSelectionPreview, confirm bool, status int) []byte {
		t.Helper()
		return h.adminCall("POST", path, contract.AudioSelectionStart{Selection: c, PreviewToken: p.PreviewToken, ConfirmLosses: confirm}, status)
	}
	t.Run("invalid choices create no work", func(t *testing.T) {
		for _, indices := range [][]int{{}, {0}, {1}, {999}} {
			bad := choice
			bad.TrackIndices = indices
			h.adminCall("POST", path+"/preview", bad, 400)
		}
		bad := choice
		bad.Version = "stale"
		h.adminCall("POST", path+"/preview", bad, 409)
		if h.count(`SELECT count(*) FROM audio_plans WHERE clip_id=$1`, h.clipID) != 0 {
			t.Fatal("invalid choice persisted")
		}
	})
	p := preview(choice)
	if len(p.Input.TrackIndices) != 1 || !p.Effects.Losses.ConfirmationRequired || p.Effects.Losses.CorrectionCount != 1 {
		t.Fatal(p)
	}
	start(choice, p, false, 409)
	t.Run("edit and forgery invalidate confirmation", func(t *testing.T) {
		bad := choice
		bad.TrackIndices = []int{7}
		start(bad, p, true, 409)
		tampered := p
		tampered.PreviewToken += "x"
		start(choice, tampered, true, 409)
	})
	t.Run("human edits conflict even with same counts", func(t *testing.T) {
		h.adminCall("PUT", segmentTextPath(h.clipID, 0), map[string]any{"text": "new human edit", "expected_version": 1}, 200)
		start(choice, p, true, 409)
	})
	t.Run("automatic inventory and multiple tracks", func(t *testing.T) {
		c := choice
		c.Mode = "auto"
		c.TrackIndices = []int{}
		auto := preview(c)
		if len(auto.Input.TrackIndices) != 2 || auto.Input.TrackIndices[1] != 7 {
			t.Fatal(auto)
		}
		c.Mode = "manual"
		c.TrackIndices = []int{7, 2}
		multi := preview(c)
		if len(multi.Input.TrackIndices) != 2 {
			t.Fatal(multi)
		}
	})
	t.Run("ambiguous and forced intent", func(t *testing.T) {
		f.diagnostics = []contract.AudioSelectionDiagnostic{{TrackIndex: 7, Status: "ambiguous", Freshness: "current", Cause: "insufficient_evidence"}, {TrackIndex: 2, Status: "no_speech", Freshness: "current", Cause: "silence"}}
		load()
		choice.Version = current.Version
		h.adminCall("POST", path+"/preview", choice, 400)
		choice.Decisions = []contract.AudioSelectionDecision{{TrackIndex: 7, Action: "exclude", Reason: "reviewed sample"}}
		if body := h.adminCall("POST", path+"/preview", choice, 400); !strings.Contains(string(body), "audio_force_required") {
			t.Fatal(string(body))
		}
		choice.Decisions = []contract.AudioSelectionDecision{{TrackIndex: 7, Action: "exclude", Reason: "reviewed sample"}, {TrackIndex: 2, Action: "include", Reason: "audible synthetic speech", Forced: true}}
		p = preview(choice)
		if !p.Input.Decisions[0].Forced || p.Input.Decisions[0].ActorId <= 0 || p.Input.Decisions[0].DecidedAt.IsZero() {
			t.Fatal(p.Input.Decisions)
		}
	})
	t.Run("inventory change invalidates old preview", func(t *testing.T) {
		_, err := pool.Exec(context.Background(), `UPDATE clips SET tracks=jsonb_set(tracks,'{2,title}','"Renamed"') WHERE id=$1`, h.clipID)
		if err != nil {
			t.Fatal(err)
		}
		start(choice, p, true, 409)
		load()
		choice.Version = current.Version
	})
	p = preview(choice)
	t.Run("engine error rolls back plan and jobs", func(t *testing.T) {
		f.fail = true
		start(choice, p, true, 503)
		if h.count(`SELECT count(*) FROM audio_plans WHERE clip_id=$1`, h.clipID) != 0 || h.count(`SELECT count(*) FROM jobs WHERE clip_id=$1 AND step LIKE 'audio.diagnostic.%'`, h.clipID) != 0 {
			t.Fatal("failed start leaked work")
		}
		f.fail = false
	})
	raw := start(choice, p, true, 200)
	assertAudioContract(t, "AudioSelectionStarted", raw)
	var started contract.AudioSelectionStarted
	_ = json.Unmarshal(raw, &started)
	stateRaw := h.adminCall("GET", fmt.Sprintf("/api/admin/clips/%d/audio-plans/%d", h.clipID, started.State.Plan.Id), nil, 200)
	assertAudioContract(t, "AudioPlanState", stateRaw)
	if len(started.State.Children) != 1 || started.State.Plan.Input.TrackIndices[0] != 2 || started.WorkerCompatible {
		t.Fatal(started)
	}
	if h.clipTranscript(h.adminUser()).Transcription.Segments[0].Text != "new human edit" {
		t.Fatal("start destroyed human work")
	}
	start(choice, p, true, 409) // Receipt cannot be reused, including after the run completes.
}
