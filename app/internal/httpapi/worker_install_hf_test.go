package httpapi

import (
	"os"
	"strings"
	"testing"
)

// Hugging Face causes of the voice-model download at installation (#203).

func TestInstallerPastedTokenBlanksAreNotPartOfTheToken(t *testing.T) {
	r := newInstallerRun(t)
	out, err := r.runTTY(t, "rw_synthetic-fixture-token", []string{"y", "accept", "  " + hfFixtureToken + "  "}, "FAKE_PROVISION_RC=0")
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, out)
	}
	envFile, _ := os.ReadFile(r.log + ".provision-env")
	if !strings.Contains(string(envFile), "HF_TOKEN="+hfFixtureToken+"\n") {
		t.Fatalf("pasted blanks reached the token:\n%q", envFile)
	}
	if !strings.Contains(r.events(), `"phase":"voice-models","status":"succeeded"`) {
		t.Fatalf("a valid pasted token must install the models:\n%s", r.events())
	}
}

func TestInstallerMalformedTokenIsRefusedBeforeAnyHuggingFaceCall(t *testing.T) {
	r := newInstallerRun(t)
	out, err := r.runTTY(t, "rw_synthetic-fixture-token", []string{"y", "accept", "not-a-token"})
	if err != nil {
		t.Fatalf("voice failure must not fail the installation: %v\n%s", err, out)
	}
	if strings.Contains(r.dockerLog(), "bootstrap provision") {
		t.Fatal("a malformed token must not start a provisioning container")
	}
	if !strings.Contains(out, "[hf_token_invalid]") || strings.Contains(out, "not-a-token") {
		t.Fatalf("expected an actionable token message without echoing the input:\n%s", out)
	}
	if !strings.Contains(r.events(), `"phase":"voice-models","status":"failed","occurred_at":`) || !strings.Contains(r.events(), `"cause_code":"hf_access_denied"`) {
		t.Fatalf("malformed token not reported as an access cause:\n%s", r.events())
	}
	if !strings.Contains(r.events(), `"phase":"service","status":"succeeded"`) {
		t.Fatalf("transcription must stay available:\n%s", r.events())
	}
}

func TestInstallerMapsHuggingFaceCausesOntoTheInstallVocabulary(t *testing.T) {
	cases := []struct{ provision, event, message string }{
		{"hf_token_invalid", "hf_access_denied", "[hf_token_invalid]"},
		{"hf_gated_denied", "hf_access_denied", "[hf_gated_denied]"},
		{"hf_repo_invalid", "unknown", "[hf_repo_invalid]"},
		{"network_unreachable", "network_unreachable", "[network_unreachable]"},
		{"hf_access_denied", "hf_access_denied", "[hf_access_denied]"},
		{"not_a_cause", "unknown", "[unknown]"},
	}
	for _, c := range cases {
		t.Run(c.provision, func(t *testing.T) {
			r := newInstallerRun(t)
			out, err := r.runTTY(t, "rw_synthetic-fixture-token", []string{"y", "accept", hfFixtureToken}, "FAKE_PROVISION_RC=1", "FAKE_PROVISION_OUT=cause="+c.provision+"\n")
			if err != nil {
				t.Fatalf("voice failure must not fail the installation: %v\n%s", err, out)
			}
			if !strings.Contains(r.events(), `"cause_code":"`+c.event+`"`) {
				t.Errorf("event cause want %s:\n%s", c.event, r.events())
			}
			if !strings.Contains(out, c.message) || strings.Contains(out+r.events(), hfFixtureToken) {
				t.Errorf("message %s missing or secret leaked:\n%s", c.message, out)
			}
		})
	}
}
