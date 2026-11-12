package httpapi

import (
	"encoding/json"
	"testing"
)

// The WeSpeaker family (#375): fingerprint only, from the package that PyPI actually publishes for WeSpeaker
// inference. `wespeaker` itself is not on PyPI; `wespeakerruntime` 1.0.1 is its latest release (ONNX inference).
func TestWeSpeakerFamily(t *testing.T) {
	f, ok := engineFamilies["wespeaker"]
	if !ok {
		t.Fatal("the wespeaker family is missing")
	}
	if f.Engine != "wespeaker" || f.Distribution != "wespeakerruntime" || f.DefaultVer != "1.0.1" {
		t.Fatalf("family identity: %+v", f)
	}
	if len(f.Kinds) != 1 || f.Kinds[0] != "fingerprint" {
		t.Fatalf("kinds: %v", f.Kinds)
	}
	if !engineVersion.MatchString(f.DefaultVer) {
		t.Fatal("the default version must be a valid engine version")
	}
	// No GPU is required: the ONNX graph runs on the CPU, so a CPU-only Worker hosts the family.
	m := engineManifest{Python: f.Python, Platform: f.Platform, CUDA: f.CUDA, DiskBytes: f.DiskBytes, RAMBytes: f.RAMBytes, VRAMBytes: f.VRAMBytes}
	cpu := &workerHardware{Python: "3.12", Platform: "linux-x86_64", RAMBytes: 16 << 30, DiskFreeBytes: 100 << 30}
	if v := checkEngineResources(m, cpu, 0, false); v.Cause != "" {
		t.Fatalf("a CPU Worker must host the family: %+v", v)
	}
	// The manifest delivered to a Worker names the family and its kind, and its identity differs from NeMo's.
	raw, digest, err := onDemandManifest(f, f.DefaultVer, "fingerprint")
	if err != nil {
		t.Fatal(err)
	}
	var delivered map[string]any
	if err := json.Unmarshal(raw, &delivered); err != nil {
		t.Fatal(err)
	}
	if delivered["family"] != "wespeaker" || delivered["kind"] != "fingerprint" || delivered["distribution"] != "wespeakerruntime" {
		t.Fatalf("manifest: %v", delivered)
	}
	if digest == onDemandDigest("nemo", f.DefaultVer, "fingerprint") || digest != onDemandDigest("wespeaker", "1.0.1", "fingerprint") {
		t.Fatal("identity")
	}
}

func TestOnDemandEngineIdentity(t *testing.T) {
	a := onDemandDigest("nemo", "3.0.0", "diarization")
	if len(a) != 64 || !engineDigest.MatchString(a) {
		t.Fatalf("digest shape: %q", a)
	}
	// Known-good literal: the identity must stay stable across releases of Snitch.
	if want := "7a7708ae8e9732ba400fe397da7b3c6052d5a432810b8e386cdb183ae271259a"; a != want {
		t.Fatalf("identity changed: %s", a)
	}
	for name, other := range map[string]string{
		"version": onDemandDigest("nemo", "3.0.1", "diarization"),
		"kind":    onDemandDigest("nemo", "3.0.0", "fingerprint"),
		"family":  onDemandDigest("nemo2", "3.0.0", "diarization"),
	} {
		if other == a {
			t.Fatalf("%s does not change the identity", name)
		}
	}
	if onDemandDigest("nemo", "3.0.0", "diarization") != a {
		t.Fatal("identity is not deterministic")
	}
}

func TestSpeechBrainFamily(t *testing.T) {
	f, ok := engineFamilies["speechbrain"]
	if !ok {
		t.Fatal("the SpeechBrain family is missing")
	}
	if f.Engine != "speechbrain" || f.Distribution != "speechbrain" || f.DefaultVer == "" || !engineVersion.MatchString(f.DefaultVer) {
		t.Fatalf("family identity: %+v", f)
	}
	// The engine name must never be the one of the signed ECAPA release: two identities are never confused.
	if f.Engine == "speechbrain-ecapa" {
		t.Fatal("the family engine collides with the signed ECAPA engine")
	}
	if len(f.Kinds) != 2 || f.Kinds[0] != "fingerprint" || f.Kinds[1] != "separation" {
		t.Fatalf("kinds: %v", f.Kinds)
	}
	// Both kinds of the family have an adapter (ECAPA-TDNN fingerprints, SepFormer separation since #401).
	if !f.runs("fingerprint") || !f.runs("separation") || f.runs("diarization") {
		t.Fatalf("fingerprint and separation run, nothing else: %v", f.ActiveKinds)
	}
	if n := engineFamilies["nemo"]; !n.runs("diarization") || !n.runs("fingerprint") {
		t.Fatalf("every kind of the NeMo family still runs: %v", n.ActiveKinds)
	}
	raw, digest, err := onDemandManifest(f, f.DefaultVer, "fingerprint")
	if err != nil || len(digest) != 64 {
		t.Fatalf("manifest: %v %q", err, digest)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["source"] != "on_demand" || m["family"] != "speechbrain" || m["engine"] != "speechbrain" || m["distribution"] != "speechbrain" || m["kind"] != "fingerprint" || m["cuda"] != "" {
		t.Fatalf("manifest: %v", m)
	}
	if kinds, _ := m["kinds"].([]any); len(kinds) != 2 || kinds[0] != "fingerprint" || kinds[1] != "separation" {
		t.Fatalf("manifest kinds: %v", m["kinds"])
	}
	if digest == onDemandDigest("nemo", f.DefaultVer, "fingerprint") {
		t.Fatal("families share an identity")
	}
	cpu := &workerHardware{Python: "3.12", Platform: "linux-x86_64", RAMBytes: 64 << 30, DiskFreeBytes: 100 << 30}
	if v := checkEngineResources(engineManifest{Python: f.Python, Platform: f.Platform, CUDA: f.CUDA, DiskBytes: f.DiskBytes, RAMBytes: f.RAMBytes}, cpu, 0, false); v.Cause != "" {
		t.Fatalf("a CPU Worker must host the family: %+v", v)
	}
}

func TestOnDemandFamilyConstraints(t *testing.T) {
	f := engineFamilies["nemo"]
	m := engineManifest{Python: f.Python, Platform: f.Platform, CUDA: f.CUDA, DiskBytes: f.DiskBytes, RAMBytes: f.RAMBytes}
	cpu := &workerHardware{Python: "3.12", Platform: "linux-x86_64", RAMBytes: 64 << 30, DiskFreeBytes: 100 << 30}
	if v := checkEngineResources(m, cpu, 0, false); v.Cause != "" {
		t.Fatalf("a CPU Worker must host the family: %+v", v)
	}
	gpu := *cpu
	gpu.CUDA, gpu.VRAMBytes = "12.8", 24<<30
	if v := checkEngineResources(m, &gpu, 0, false); v.Cause != "" {
		t.Fatalf("a GPU Worker must host the family: %+v", v)
	}
	if v := checkEngineResources(m, cpu, 95<<30, false); v.Cause != causeDisk {
		t.Fatalf("reserved disk must count: %+v", v)
	}
}

func TestOnDemandManifestCarriesTheKind(t *testing.T) {
	f := engineFamilies["nemo"]
	for _, kind := range f.Kinds {
		raw, digest, err := onDemandManifest(f, "3.0.0", kind)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		if m["kind"] != kind || digest != onDemandDigest("nemo", "3.0.0", kind) {
			t.Fatalf("%s: manifest kind %v digest %s", kind, m["kind"], digest)
		}
	}
	if f.covers("separation") || !f.covers("fingerprint") || !f.covers("diarization") {
		t.Fatal("a family covers exactly its kinds")
	}
}

func TestCatalogEntryEngineNeed(t *testing.T) {
	cases := []struct {
		name      string
		entry     voiceCatalogEntry
		needed    bool
		supported bool
	}{
		{"embedded Sortformer keeps its engine", voiceCatalogEntry{Engine: "nemo", Kind: "diarization"}, false, true},
		{"signed fingerprint adapter", voiceCatalogEntry{Engine: "nemo-titanet", Kind: "fingerprint"}, false, true},
		{"a newer NeMo model asks for the family", voiceCatalogEntry{Engine: "nemo", Kind: "diarization", Family: "nemo"}, true, true},
		{"NeMo fingerprint", voiceCatalogEntry{Engine: "nemo", Kind: "fingerprint", Family: "nemo"}, true, true},
		{"a family the table does not know", voiceCatalogEntry{Engine: "speechbrain-ecapa", Kind: "fingerprint", Family: "mystery-lib"}, true, false},
		{"a kind the family does not cover", voiceCatalogEntry{Engine: "nemo", Kind: "separation", Family: "nemo"}, true, false},
	}
	for _, c := range cases {
		family, needed, supported := c.entry.engineNeed()
		if needed != c.needed || supported != c.supported {
			t.Errorf("%s: needed=%v supported=%v", c.name, needed, supported)
		}
		if needed && supported && family.Key != c.entry.Family {
			t.Errorf("%s: family %q", c.name, family.Key)
		}
	}
}

func TestCatalogFamilyOfADiscoveredModel(t *testing.T) {
	cases := []struct{ engine, kind, want string }{
		{"nemo", "diarization", "nemo"}, // a discovered NeMo model is tried on the on-demand family
		{"pyannote", "diarization", ""}, // embedded
		{"pyannote", "fingerprint", ""}, // embedded
		{"nemo-titanet", "fingerprint", "nemo"},
		{"speechbrain-ecapa", "fingerprint", "speechbrain"},
		{"speechbrain-sepformer", "separation", "speechbrain"},
		{"speechbrain-sepformer", "fingerprint", "speechbrain"}, // the family covers it; the matrix never proposes it
		{"speechbrain-ecapa", "diarization", ""},                // a kind the family does not cover
		{"wespeaker", "fingerprint", "wespeaker"},
		{"wespeaker", "diarization", ""},
		{"silero-jit", "vad", ""},
		{"", "separation", ""},
	}
	for _, c := range cases {
		if got := catalogFamily(c.engine, c.kind); got != c.want {
			t.Errorf("%s/%s: family %q, want %q", c.engine, c.kind, got, c.want)
		}
	}
}

func TestFamilyActiveKinds(t *testing.T) {
	for family, kinds := range map[string]map[string]bool{
		"nemo":        {"diarization": true, "fingerprint": true},
		"speechbrain": {"fingerprint": true, "separation": true, "diarization": false},
		"wespeaker":   {"fingerprint": true},
	} {
		for kind, active := range kinds {
			if got := engineFamilies[family].runs(kind); got != active {
				t.Errorf("%s/%s runs=%v, want %v", family, kind, got, active)
			}
		}
	}
}
