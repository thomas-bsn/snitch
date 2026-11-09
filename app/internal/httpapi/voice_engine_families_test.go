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
	raw, digest, err := onDemandManifest(f, f.DefaultVer)
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
