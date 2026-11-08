package httpapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The release tool (scripts/engine_release.py) signs the exact bytes of a manifest. This envelope was made by
// that tool with a throwaway key whose private half was never stored; the Go registry must accept it unchanged.
func TestEngineReleaseEnvelopeFromThePublisherTool(t *testing.T) {
	read := func(name string) []byte {
		raw, err := os.ReadFile("../../../worker/tests/fixtures/engine-release/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	var envelope struct {
		KeyID     string `json:"key_id"`
		Payload   string `json:"payload"`
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(read("envelope.json"), &envelope); err != nil {
		t.Fatal(err)
	}
	keys := map[string]string{envelope.KeyID: strings.TrimSpace(string(read("public-key.txt")))}
	manifest, _, digest, ok := verifyEngineRelease(envelope.KeyID, envelope.Payload, envelope.Signature, keys)
	if !ok || len(digest) != 64 {
		t.Fatal("the registry refused an envelope made by the publisher tool")
	}
	if manifest.Engine != "nemo-titanet" || manifest.Architecture != "titanet" || manifest.Kind != "fingerprint" || len(manifest.Wheels) != 2 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	if manifest.InputContract != "audio-mono-16khz" || manifest.OutputContract != "speaker-embedding-unit-v1" || manifest.RAMBytes != 1<<30 {
		t.Fatalf("contract fields lost: %+v", manifest)
	}
	if _, _, _, ok := verifyEngineRelease(envelope.KeyID, envelope.Payload, envelope.Signature, map[string]string{envelope.KeyID: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}); ok {
		t.Fatal("accepted with another public key")
	}
}

func TestCommittedECAPAReleaseIsAcceptedByRegistry(t *testing.T) {
	raw, err := os.ReadFile("../../../worker/engine-releases/speechbrain-ecapa-1.0.3-r1/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encode := base64.StdEncoding.EncodeToString
	m, accepted, _, ok := verifyEngineRelease("test", encode(raw), encode(ed25519.Sign(private, raw)), map[string]string{"test": encode(public)})
	if !ok || string(accepted) != string(raw) {
		t.Fatal("registry refused the exact committed ECAPA release")
	}
	if m.Engine != "speechbrain-ecapa" || m.Version != "1.0.3-cu128-r1" || m.OutputContract != "fingerprint-192-unit-v1" || len(m.Wheels) != 50 {
		t.Fatalf("unexpected release identity or dependency closure: %+v", m)
	}
}
