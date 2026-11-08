package httpapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func testEngineManifest() engineManifest {
	return engineManifest{Schema: 1, Engine: "nemo-titanet", Version: "1.0.0", AdapterVersion: 1, Python: "3.12", Platform: "linux-x86_64", CUDA: "cpu", License: "Apache-2.0", Kind: "fingerprint", Architecture: "titanet", DiskBytes: 1024, Wheels: []engineWheel{{Name: "adapter-1-py3-none-any.whl", URL: "https://files.pythonhosted.org/packages/adapter.whl", SHA256: strings.Repeat("a", 64), Size: 100}}}
}
func TestEngineReleaseTrust(t *testing.T) {
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(testEngineManifest())
	keys := map[string]string{"publisher": base64.StdEncoding.EncodeToString(pub)}
	encode := base64.StdEncoding.EncodeToString
	sig := encode(ed25519.Sign(private, raw))
	if _, _, digest, ok := verifyEngineRelease("publisher", encode(raw), sig, keys); !ok || len(digest) != 64 {
		t.Fatal("valid signature rejected")
	}
	for _, test := range []struct{ name, key, payload, signature string }{
		{"unknown publisher", "other", encode(raw), sig},
		{"changed payload", "publisher", encode(append(raw, ' ')), sig},
		{"malformed signature", "publisher", encode(raw), "bogus"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, ok := verifyEngineRelease(test.key, test.payload, test.signature, keys); ok {
				t.Fatal("untrusted release accepted")
			}
		})
	}
	for _, bad := range []string{strings.Replace(string(raw), `"vram_bytes":0`, `"vram_bytes":null`, 1), strings.Replace(string(raw), `"vram_bytes":0,`, "", 1)} {
		if _, _, _, ok := verifyEngineRelease("publisher", encode([]byte(bad)), encode(ed25519.Sign(private, []byte(bad))), keys); ok {
			t.Fatal("missing or null required field accepted")
		}
	}
	// Even a publisher cannot extend the executable protocol through unknown fields.
	extended := append([]byte(`{"import_path":"evil",`), raw[1:]...)
	if _, _, _, ok := verifyEngineRelease("publisher", encode(extended), encode(ed25519.Sign(private, extended)), keys); ok {
		t.Fatal("unknown protocol field accepted")
	}
}
func TestEngineManifestBoundaries(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*engineManifest)
	}{
		{"unknown architecture", func(m *engineManifest) { m.Architecture = "custom" }},
		{"separation disabled", func(m *engineManifest) { m.Kind = "separation" }},
		{"arbitrary registry", func(m *engineManifest) { m.Wheels[0].URL = "https://evil.example/a.whl" }},
		{"URL credentials", func(m *engineManifest) { m.Wheels[0].URL = "https://user@files.pythonhosted.org/a.whl" }},
		{"path escape", func(m *engineManifest) { m.Wheels[0].Name = "../a.whl" }},
		{"source build", func(m *engineManifest) { m.Wheels[0].Name = "a.tar.gz" }},
		{"unpinned hash", func(m *engineManifest) { m.Wheels[0].SHA256 = "" }},
		{"duplicate wheel", func(m *engineManifest) { m.Wheels = append(m.Wheels, m.Wheels[0]) }},
		{"disk budget", func(m *engineManifest) { m.DiskBytes = 1 }},
		{"unsupported Python", func(m *engineManifest) { m.Python = "3.11" }},
		{"invalid CUDA", func(m *engineManifest) { m.CUDA = "latest" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := testEngineManifest()
			test.mutate(&m)
			if validEngineManifest(m) {
				t.Fatal("invalid release accepted")
			}
		})
	}
}
