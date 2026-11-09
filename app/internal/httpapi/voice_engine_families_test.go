package httpapi

import "testing"

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
