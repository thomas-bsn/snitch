package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func metaOf(t *testing.T, raw string) hfMetadata {
	t.Helper()
	var m hfMetadata
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

const sha = `"sha":"cd03eee90fbec18297ac31b8c21546e596b7f71c"`

func TestAdapterMatrix(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		kind   string // proposed kind, "" when not reliable
		engine string // "" when no approved adapter
		kinds  []string
		reason string // substring of the unsupported reason
	}{
		{"TitaNet fingerprint precedes generic NeMo", `{"library_name":"nemo","tags":["nemo","titanet","speaker-embedding"]}`,
			"fingerprint", "nemo-titanet", []string{"fingerprint"}, ""},
		{"ECAPA fingerprint", `{"library_name":"speechbrain","tags":["ecapa-tdnn","speaker-verification"]}`,
			"fingerprint", "speechbrain-ecapa", []string{"fingerprint"}, ""},
		// speechbrain/spkrec-ecapa-voxceleb: the words are two capitalised tags, and no type tag is declared.
		{"ECAPA spelled as two capitalised tags", `{"library_name":"speechbrain","tags":["speechbrain","embeddings","Speaker","Verification","Identification","pytorch","ECAPA","TDNN","en"]}`,
			"", "speechbrain-ecapa", []string{"fingerprint"}, ""},
		{"ECAPA tag in capitals", `{"library_name":"speechbrain","tags":["ECAPA-TDNN","speaker-verification"]}`,
			"fingerprint", "speechbrain-ecapa", []string{"fingerprint"}, ""},
		{"ECAPA alone is not enough", `{"library_name":"speechbrain","tags":["speechbrain","ECAPA"]}`,
			"", "", nil, "no approved adapter"},
		{"WeSpeaker fingerprint", `{"library_name":"wespeaker","tags":["wespeaker","speaker-embedding"]}`,
			"fingerprint", "wespeaker", []string{"fingerprint"}, ""},
		{"pyannote wrapper of WeSpeaker stays pyannote", `{"library_name":"pyannote-audio","tags":["pyannote-audio","wespeaker","speaker-embedding"]}`,
			"fingerprint", "pyannote", []string{"diarization", "fingerprint"}, ""},
		{"unknown SpeechBrain architecture", `{"library_name":"speechbrain","tags":["speaker-embedding"]}`,
			"fingerprint", "", nil, "no approved adapter"},
		{"TitaNet remote code refused", `{"library_name":"nemo","tags":["titanet","speaker-embedding","custom_code"]}`,
			"fingerprint", "", nil, "remote code"},
		{"known Sortformer", `{"id":"nvidia/diar_streaming_sortformer_4spk-v2.1",` + sha + `,"tags":["nemo","speaker-diarization"],"library_name":"nemo","siblings":[{"rfilename":"model.nemo"}]}`,
			"diarization", "nemo", []string{"diarization"}, ""},
		{"pyannote pipeline", `{"id":"acme/p",` + sha + `,"library_name":"pyannote-audio","tags":["pyannote","pyannote-audio","pyannote-audio-pipeline","speaker-diarization"],"siblings":[{"rfilename":"config.yaml"}]}`,
			"diarization", "pyannote", []string{"diarization", "fingerprint"}, ""},
		{"pyannote embedding", `{"id":"acme/e",` + sha + `,"library_name":"pyannote-audio","tags":["pyannote-audio","speaker-embedding"]}`,
			"fingerprint", "pyannote", []string{"diarization", "fingerprint"}, ""},
		{"compatible but type not reliable (no type tag)", `{"id":"acme/speaker-diarization-final",` + sha + `,"library_name":"pyannote-audio","tags":["pyannote-audio"]}`,
			"", "pyannote", []string{"diarization", "fingerprint"}, ""},
		{"misleading name and pipeline tag", `{"id":"acme/diarization",` + sha + `,"pipeline_tag":"automatic-speech-recognition","library_name":"nemo","tags":["nemo"],"siblings":[{"rfilename":"x.nemo"}]}`,
			"", "nemo", []string{"diarization"}, ""},
		{"conflicting type tags", `{"id":"acme/c",` + sha + `,"library_name":"pyannote-audio","tags":["speaker-diarization","speaker-embedding"]}`,
			"", "pyannote", []string{"diarization", "fingerprint"}, ""},
		{"missing metadata", `{"id":"acme/empty",` + sha + `}`, "", "", nil, "no approved adapter"},
		{"transformers only", `{"id":"acme/t",` + sha + `,"library_name":"transformers","tags":["transformers","speaker-diarization"]}`,
			"diarization", "", nil, "no approved adapter"},
		{"remote code", `{"id":"acme/rc",` + sha + `,"library_name":"pyannote-audio","tags":["pyannote-audio","custom_code","speaker-diarization"]}`,
			"diarization", "", nil, "remote code"},
		{"auto_map remote code", `{"id":"acme/am",` + sha + `,"library_name":"nemo","tags":["nemo"],"config":{"auto_map":{"AutoModel":"x.Y"}}}`,
			"", "", nil, "remote code"},
		{"an adapter of another type does not separate", `{"id":"acme/sep",` + sha + `,"library_name":"pyannote-audio","tags":["pyannote-audio","audio-source-separation"]}`,
			"separation", "pyannote", nil, "separation"},
		// The tags of the model card of speechbrain/sepformer-whamr16k, one of the two documented candidates.
		{"documented SepFormer", `{"id":"speechbrain/sepformer-whamr16k",` + sha + `,"library_name":"speechbrain","tags":["speechbrain","audio-to-audio","audio-source-separation","Source Separation","Speech Separation","WHAM!","SepFormer","Transformer","pytorch","en"]}`,
			"separation", "speechbrain-sepformer", []string{"separation"}, ""},
		{"SepFormer whose limits are not documented", `{"id":"acme/sepformer-other",` + sha + `,"library_name":"speechbrain","tags":["speechbrain","SepFormer","audio-source-separation"]}`,
			"separation", "speechbrain-sepformer", nil, "limits"},
		{"SepFormer with remote code", `{"id":"speechbrain/sepformer-whamr16k",` + sha + `,"library_name":"speechbrain","tags":["speechbrain","SepFormer","audio-source-separation","custom_code"]}`,
			"separation", "", nil, "remote code"},
		{"another SpeechBrain separation architecture", `{"id":"acme/convtasnet",` + sha + `,"library_name":"speechbrain","tags":["speechbrain","audio-source-separation"]}`,
			"separation", "", nil, "no approved adapter"},
		// Both approved adapters match: the order of the matrix decides (nemo, then pyannote), never the user.
		{"several adapters", `{"id":"acme/both",` + sha + `,"library_name":"nemo","tags":["nemo","pyannote-audio","speaker-diarization"],"siblings":[{"rfilename":"m.nemo"}]}`,
			"diarization", "nemo", []string{"diarization"}, ""},
	}
	for _, c := range cases {
		a := assessHF(metaOf(t, c.raw))
		if a.ProposedKind != c.kind || a.Engine != c.engine || strings.Join(a.CompatibleKinds, ",") != strings.Join(c.kinds, ",") {
			t.Errorf("%s: kind %q engine %q kinds %v, want %q %q %v", c.name, a.ProposedKind, a.Engine, a.CompatibleKinds, c.kind, c.engine, c.kinds)
		}
		if (c.reason == "") != (a.UnsupportedReason == "") || !strings.Contains(a.UnsupportedReason, c.reason) {
			t.Errorf("%s: reason %q, want %q", c.name, a.UnsupportedReason, c.reason)
		}
	}
}

// The type chosen for an add is checked against the engine: a type the adapter cannot run is unsupported.
func TestAssessmentForKind(t *testing.T) {
	a := assessHF(metaOf(t, `{"id":"nvidia/x",`+sha+`,"library_name":"nemo","tags":["nemo"]}`))
	if ok, reason := a.supports("diarization"); !ok || reason != "" {
		t.Fatalf("%v %q", ok, reason)
	}
	for _, kind := range []string{"fingerprint", "separation"} {
		if ok, reason := a.supports(kind); ok || reason == "" {
			t.Errorf("%s must be unsupported with a reason: %v %q", kind, ok, reason)
		}
	}
	// A documented separation model is a separation model and nothing else.
	sep := assessHF(metaOf(t, `{"id":"speechbrain/sepformer-wsj03mix",`+sha+`,"library_name":"speechbrain","tags":["speechbrain","SepFormer"]}`))
	if ok, reason := sep.supports("separation"); !ok || reason != "" {
		t.Fatalf("%v %q", ok, reason)
	}
	if ok, reason := sep.supports("fingerprint"); ok || reason == "" {
		t.Fatalf("a separation model is not a fingerprint model: %v %q", ok, reason)
	}
}

// The limits shown in the catalogue are the ones of the documentation (ADR 0066), and every supported
// separation reference has both.
func TestSeparationModelsDeclareTheirLimits(t *testing.T) {
	want := map[string]separationLimits{
		"speechbrain/sepformer-whamr16k": {Voices: 2, SampleRateHz: 16000},
		"speechbrain/sepformer-wsj03mix": {Voices: 3, SampleRateHz: 8000},
	}
	if len(separationModels) != len(want) {
		t.Fatalf("supported separation models: %v", separationModels)
	}
	for repo, limits := range want {
		if separationModels[repo] != limits {
			t.Errorf("%s: %+v, want %+v", repo, separationModels[repo], limits)
		}
	}
	if catalogFamily(sepformerEngine, "separation") != "speechbrain" || !engineFamilies["speechbrain"].runs("separation") {
		t.Fatal("the SepFormer adapter runs on the SpeechBrain family")
	}
}
