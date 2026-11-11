package jobs

import "testing"

func TestVoiceInputVersion(t *testing.T) {
	for _, c := range []struct {
		version     string
		pipeline    string
		diarization int64
		ok          bool
	}{
		{"d5-a2-pvoice-0123abcd", "voice-0123abcd", 0, true},
		{"f5-a2-pvoice-0123abcd-d12", "voice-0123abcd", 12, true},
		// A recalculation on the net speech (#395) marks its version before the pipeline.
		{"f5-a2-net-pvoice-0123abcd-d12", "voice-0123abcd", 12, true},
		{"f5-a2-pvoice-0123abcd-dx", "", 0, false},
		{"other", "", 0, false},
	} {
		pipeline, diarization, ok := VoiceInputVersion(c.version)
		if pipeline != c.pipeline || diarization != c.diarization || ok != c.ok {
			t.Errorf("%s: got %q %d %v", c.version, pipeline, diarization, ok)
		}
	}
}
