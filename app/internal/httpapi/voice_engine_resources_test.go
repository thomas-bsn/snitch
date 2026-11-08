package httpapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

const gib = int64(1) << 30

func reportedHardware() *workerHardware {
	return &workerHardware{Python: "3.12", Platform: "linux-x86_64", CUDA: "12.4", RAMBytes: 16 * gib, VRAMBytes: 8 * gib, DiskFreeBytes: 100 * gib}
}

func cudaManifest() engineManifest {
	m := testEngineManifest()
	m.CUDA, m.VRAMBytes = "12.4", 4*gib
	return m
}

func TestLegacyManifestStaysValidAndOptionalFieldsAreChecked(t *testing.T) {
	if !validEngineManifest(testEngineManifest()) {
		t.Fatal("a manifest without optional fields must stay valid")
	}
	raw, _ := json.Marshal(testEngineManifest())
	var fields map[string]any
	_ = json.Unmarshal(raw, &fields)
	if len(fields) != len(engineRequiredFields) {
		t.Fatalf("a legacy manifest must serialize to exactly its %d required fields, got %d", len(engineRequiredFields), len(fields))
	}
	valid := testEngineManifest()
	valid.RAMBytes, valid.Capabilities, valid.InputContract, valid.OutputContract, valid.CPUFallback = 8*gib, []string{"fingerprint"}, "audio-16k-mono", "embedding-v1", true
	if !validEngineManifest(valid) {
		t.Fatal("a manifest with constraints and contracts must be valid")
	}
	for name, mutate := range map[string]func(*engineManifest){
		"negative RAM":                 func(m *engineManifest) { m.RAMBytes = -1 },
		"separation capability":        func(m *engineManifest) { m.Capabilities = []string{"separation"} },
		"capability of the wrong kind": func(m *engineManifest) { m.Capabilities = []string{"diarization"} },
		"duplicate capability":         func(m *engineManifest) { m.Capabilities = []string{"fingerprint", "fingerprint"} },
		"malformed contract":           func(m *engineManifest) { m.InputContract = "Not A Contract" },
	} {
		m := testEngineManifest()
		mutate(&m)
		if validEngineManifest(m) {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestSignedManifestWithOptionalFieldsIsTrusted(t *testing.T) {
	pub, private, _ := ed25519.GenerateKey(rand.Reader)
	encode := base64.StdEncoding.EncodeToString
	m := testEngineManifest()
	m.RAMBytes, m.InputContract = 2*gib, "audio-16k-mono"
	raw, _ := json.Marshal(m)
	if _, _, _, ok := verifyEngineRelease("p", encode(raw), encode(ed25519.Sign(private, raw)), map[string]string{"p": encode(pub)}); !ok {
		t.Fatal("signed manifest with optional constraints rejected")
	}
	bad := strings.Replace(string(raw), `"ram_bytes":2147483648`, `"ram_bytes":null`, 1)
	if _, _, _, ok := verifyEngineRelease("p", encode([]byte(bad)), encode(ed25519.Sign(private, []byte(bad))), map[string]string{"p": encode(pub)}); ok {
		t.Fatal("null optional field accepted")
	}
}

func TestEngineResourceVerdicts(t *testing.T) {
	cases := []struct {
		name     string
		manifest func() engineManifest
		hw       func(*workerHardware)
		reserved int64
		skipDisk bool
		cause    string
	}{
		{"fits", cudaManifest, func(*workerHardware) {}, 0, false, ""},
		{"unknown hardware never blocks", cudaManifest, nil, 0, false, ""},
		{"other python", cudaManifest, func(h *workerHardware) { h.Python = "3.11" }, 0, false, causePlatform},
		{"other platform", cudaManifest, func(h *workerHardware) { h.Platform = "linux-aarch64" }, 0, false, causePlatform},
		{"other CUDA", cudaManifest, func(h *workerHardware) { h.CUDA = "11.8" }, 0, false, causeCUDA},
		{"no CUDA", cudaManifest, func(h *workerHardware) { h.CUDA, h.VRAMBytes = "", 0 }, 0, false, causeCUDA},
		{"not enough VRAM", cudaManifest, func(h *workerHardware) { h.VRAMBytes = 2 * gib }, 0, false, causeVRAM},
		{"not enough RAM", func() engineManifest { m := cudaManifest(); m.RAMBytes = 32 * gib; return m }, func(*workerHardware) {}, 0, false, causeInsufficient},
		{"not enough disk", cudaManifest, func(h *workerHardware) { h.DiskFreeBytes = 512 }, 0, false, causeDisk},
		{"reservations count against the disk", cudaManifest, func(h *workerHardware) { h.DiskFreeBytes = 1500 }, 1000, false, causeDisk},
		{"a runtime already present needs no disk", cudaManifest, func(h *workerHardware) { h.DiskFreeBytes = 0 }, 0, true, ""},
		{"a CPU release needs no GPU", func() engineManifest { return testEngineManifest() }, func(h *workerHardware) { h.CUDA, h.VRAMBytes = "", 0 }, 0, false, ""},
		{"CPU fallback needs the contract", cudaManifest, func(h *workerHardware) { h.CUDA, h.VRAMBytes, h.CPUFallback = "", 0, true }, 0, false, causeCUDA},
		{"CPU fallback needs the configuration", func() engineManifest { m := cudaManifest(); m.CPUFallback = true; return m }, func(h *workerHardware) { h.CUDA, h.VRAMBytes = "", 0 }, 0, false, causeCUDA},
		{"CPU fallback flags cannot substitute a CPU release", func() engineManifest { m := cudaManifest(); m.CPUFallback = true; return m }, func(h *workerHardware) { h.CUDA, h.VRAMBytes, h.CPUFallback = "", 0, true }, 0, false, causeCUDA},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var hw *workerHardware
			if c.hw != nil {
				hw = reportedHardware()
				c.hw(hw)
			}
			v := checkEngineResources(c.manifest(), hw, c.reserved, c.skipDisk)
			if v.Cause != c.cause {
				t.Fatalf("cause %q, want %q", v.Cause, c.cause)
			}
			if c.cause == causeVRAM && (v.Required != 4*gib || v.Available != 2*gib) {
				t.Fatalf("the refusal must carry the exact numbers: %+v", v)
			}
			if c.cause == causeDisk && c.reserved > 0 && v.Available != 500 {
				t.Fatalf("available disk must exclude reservations: %+v", v)
			}
		})
	}
}
