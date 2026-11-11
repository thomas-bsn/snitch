//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/jackc/pgx/v5/pgxpool"
)

type measuresProbe struct {
	Version          int      `json:"version"`
	NetSpeechSeconds float64  `json:"net_speech_seconds"`
	OverlapRatio     float64  `json:"overlap_ratio"`
	Overlapped       bool     `json:"overlapped"`
	ClippedRatio     *float64 `json:"clipped_ratio"`
	MeanLevelDBFS    *float64 `json:"mean_level_dbfs"`
	Coherence        *float64 `json:"coherence"`
}

type fingerprintProbe struct {
	LocalSpeakerID int64          `json:"local_speaker_id"`
	Label          string         `json:"label"`
	Scope          string         `json:"scope"`
	TurnIndex      *int           `json:"turn_index"`
	Duration       float64        `json:"duration_seconds"`
	Usable         bool           `json:"usable"`
	Measures       *measuresProbe `json:"measures"`
}

// validateTraceContract checks a voice trace answer against contracts/openapi.yaml: measures present, absent (null)
// and partially null are all described by the contract.
func validateTraceContract(t *testing.T, url string, body []byte) {
	t.Helper()
	spec, err := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec.Servers = nil
	router, err := legacy.NewRouter(spec)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	route, params, err := router.FindRoute(req)
	if err != nil {
		t.Fatal(err)
	}
	input := &openapi3filter.RequestValidationInput{Request: req, Route: route, PathParams: params, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
	header := http.Header{"Content-Type": []string{"application/json"}}
	if err = openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: http.StatusOK, Header: header, Body: io.NopCloser(bytes.NewReader(body))}); err != nil {
		t.Fatalf("voice trace contract: %v", err)
	}
}

// Measures of a fingerprint (#389, ADR 0064): stored with it, read by an administrator in the clip's voice trace;
// a result of an older Worker, without measures, is accepted as before. Synthetic vectors only.
func testVoiceFingerprintMeasures(t *testing.T, pool *pgxpool.Pool) {
	h := newVoiceHarness(t, pool)
	h.putPipeline(fakePipeline(2))
	h.transcribe()
	h.adminCall(http.MethodPost, fmt.Sprintf("/api/admin/clips/%d/voice", h.clipID), map[string]any{"audio_stream_index": 2}, http.StatusCreated)
	diarization, ok := h.claim()
	if !ok || diarization.Step != "voice.diarization.2" {
		t.Fatalf("diarization expected: %+v", diarization)
	}
	h.complete(diarization, diarizationPayload("SPEAKER_00", "SPEAKER_01"), http.StatusOK)
	fp, ok := h.claim()
	if !ok || fp.Step != "voice.fingerprint.2" {
		t.Fatalf("fingerprint step expected: %+v", fp)
	}

	vec := []float64{0.5, 0.5, 0.5, 0.5}
	measures := func(net, overlap float64, overlapped bool, clipped, level, coherence any) map[string]any {
		return map[string]any{"version": 1, "net_speech_seconds": net, "overlap_ratio": overlap, "overlapped": overlapped, "clipped_ratio": clipped, "mean_level_dbfs": level, "coherence": coherence}
	}
	result := func(items ...map[string]any) map[string]any {
		return map[string]any{"provider": "fake", "embedding_model": "fixture-embedding", "embedding_model_version": "emb-v1", "audio_stream_index": 2, "diarization_job_id": diarization.ID, "dimensions": 4, "fingerprints": items}
	}
	speaker := map[string]any{"speaker_label": "SPEAKER_00", "scope": "speaker", "start": 0, "end": 1.0, "quality": 0.9, "embedding": vec, "measures": measures(1.0, 0.3333, false, 0.001, -23.5, 0.82)}
	turn := map[string]any{"speaker_label": "SPEAKER_00", "scope": "turn", "turn_index": 0, "start": 0, "end": 1.5, "quality": 0.9, "embedding": vec, "measures": measures(1.0, 0.3333, false, 0.001, -23.5, nil)}
	// SPEAKER_01 is sent as an older Worker sends it: no measures at all.
	legacy := map[string]any{"speaker_label": "SPEAKER_01", "scope": "speaker", "start": 1, "end": 2.5, "quality": 0.9, "embedding": vec}

	// Measures out of their range are refused with the whole result; the lease stays usable.
	for _, bad := range []map[string]any{
		measures(1.0, 1.5, false, nil, nil, nil),
		measures(-1.0, 0.1, false, nil, nil, nil),
		measures(1.0, 0.1, false, 2.0, nil, nil),
		measures(1.0, 0.1, false, nil, 3.0, nil),
		measures(1.0, 0.1, false, nil, nil, 1.5),
		{"version": 0, "net_speech_seconds": 1.0, "overlap_ratio": 0.1, "overlapped": false},
		{"version": 1, "overlap_ratio": 0.1, "overlapped": false},
	} {
		invalid := map[string]any{"speaker_label": "SPEAKER_00", "scope": "speaker", "start": 0, "end": 1.0, "quality": 0.9, "embedding": vec, "measures": bad}
		h.complete(fp, result(invalid), http.StatusBadRequest)
	}
	h.complete(fp, result(speaker, turn, legacy), http.StatusOK)

	var trace struct {
		Fingerprints []fingerprintProbe `json:"fingerprints"`
	}
	tracePath := fmt.Sprintf("/api/admin/clips/%d/voice-trace", h.clipID)
	body := h.adminCall(http.MethodGet, tracePath, nil, http.StatusOK)
	if err := json.Unmarshal(body, &trace); err != nil {
		t.Fatal(err)
	}
	validateTraceContract(t, h.ts.URL+tracePath, body)
	if len(trace.Fingerprints) != 3 {
		t.Fatalf("the trace lists every fingerprint of the track: %+v", trace.Fingerprints)
	}
	first, second, third := trace.Fingerprints[0], trace.Fingerprints[1], trace.Fingerprints[2]
	if first.Label != "SPEAKER_00" || first.Scope != "speaker" || first.Measures == nil || first.Measures.Version != 1 || first.Measures.NetSpeechSeconds != 1.0 || first.Measures.OverlapRatio != 0.3333 || first.Measures.Overlapped ||
		first.Measures.ClippedRatio == nil || *first.Measures.ClippedRatio != 0.001 || first.Measures.MeanLevelDBFS == nil || *first.Measures.MeanLevelDBFS != -23.5 || first.Measures.Coherence == nil || *first.Measures.Coherence != 0.82 {
		t.Fatalf("speaker measures not stored and exposed as sent: %+v %+v", first, first.Measures)
	}
	if second.Scope != "turn" || second.TurnIndex == nil || *second.TurnIndex != 0 || second.Measures == nil || second.Measures.Coherence != nil {
		t.Fatalf("a turn has no coherence: %+v %+v", second, second.Measures)
	}
	if third.Label != "SPEAKER_01" || third.Measures != nil || !third.Usable {
		t.Fatalf("a fingerprint without measures is stored as before, its measures unknown: %+v", third)
	}
	// Eligibility is not decided by the measures in this ticket: the same rule as before applies.
	if !first.Usable || first.Duration != 1.0 {
		t.Fatalf("usable unchanged: %+v", first)
	}
	// Measures never leave the administrator's trace: the clip's voice view stays as it was.
	var view struct {
		Speakers []struct {
			Fingerprints []map[string]any `json:"fingerprints"`
		} `json:"speakers"`
	}
	if err := json.Unmarshal(h.adminCall(http.MethodGet, fmt.Sprintf("/api/clips/%d/voice", h.clipID), nil, http.StatusOK), &view); err != nil {
		t.Fatal(err)
	}
	if _, leaked := view.Speakers[0].Fingerprints[0]["measures"]; leaked {
		t.Fatal("the clip's voice view must not carry measures")
	}
}
