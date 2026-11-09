package httpapi

import (
	"encoding/json"
	"testing"
)

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
	// Separation is installed for but never activated: no adapter runs it.
	if !f.runs("fingerprint") || f.runs("separation") || f.runs("diarization") {
		t.Fatalf("only fingerprint may run: %v", f.ActiveKinds)
	}
	if n := engineFamilies["nemo"]; !n.runs("diarization") || !n.runs("fingerprint") {
		t.Fatalf("every kind of the NeMo family still runs: %v", n.ActiveKinds)
	}
	raw, digest, err := onDemandManifest(f, f.DefaultVer)
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
