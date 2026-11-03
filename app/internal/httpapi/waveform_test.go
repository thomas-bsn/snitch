package httpapi

import (
	"encoding/json"
	"testing"
)

func TestWorkerPeaksAreOptionalAndBounded(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		ok   bool
	}{
		{"absent", `{"segments":[]}`, false},
		{"null", `{"peaks":null}`, false},
		{"empty", `{"peaks":[]}`, false},
		{"valid", `{"peaks":[0,128,255]}`, true},
		{"above full scale", `{"peaks":[256]}`, false},
		{"negative", `{"peaks":[-1]}`, false},
		{"fractional", `{"peaks":[0.5]}`, false},
		{"not an array", `{"peaks":"0,1"}`, false},
		{"too many", `{"peaks":` + manyPeaks(maxWaveformPeaks+1) + `}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := validWaveformPeaks(json.RawMessage(tc.raw))
			if ok != tc.ok {
				t.Fatalf("validWaveformPeaks(%s) ok=%v, want %v", tc.raw, ok, tc.ok)
			}
		})
	}
}

func TestWithoutPeaksKeepsTheRestOfTheResult(t *testing.T) {
	stripped := withoutPeaks(json.RawMessage(`{"provider":"fixture","audio_stream_index":2,"peaks":[1,2],"segments":[]}`))
	var object map[string]json.RawMessage
	if err := json.Unmarshal(stripped, &object); err != nil {
		t.Fatal(err)
	}
	if _, has := object["peaks"]; has || string(object["audio_stream_index"]) != "2" || string(object["segments"]) != "[]" {
		t.Fatalf("unexpected stored result: %s", stripped)
	}
}

func TestWaveformProjectionOnlyForTrackSteps(t *testing.T) {
	raw := json.RawMessage(`{"peaks":[1]}`)
	if waveformProjection("transcription.audio.2", raw) == nil || waveformProjection("voice.diarization.2", raw) == nil {
		t.Fatal("track steps must store their peaks")
	}
	for _, step := range []string{"voice.fingerprint.2", "transcription.vocabulary", "fixture.x"} {
		if waveformProjection(step, raw) != nil {
			t.Fatalf("%s must not store peaks", step)
		}
	}
	if waveformProjection("transcription.audio.2", json.RawMessage(`{}`)) != nil {
		t.Fatal("a result without peaks must store nothing")
	}
}

func manyPeaks(count int) string {
	buffer := make([]byte, 0, count*2+2)
	buffer = append(buffer, '[')
	for i := 0; i < count; i++ {
		if i > 0 {
			buffer = append(buffer, ',')
		}
		buffer = append(buffer, '0')
	}
	return string(append(buffer, ']'))
}
