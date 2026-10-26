package workers

import (
	"strings"
	"testing"
)

func TestBundleEnvCarriesTheTokenAndPublicURL(t *testing.T) {
	bundle, err := NewBundle(BundleInput{APIURL: "https://snitch.example.org", Token: "rw_abc-DEF_123", Image: "ghcr.io/thomas-bsn/snitch-worker:latest"})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"SNITCH_API_URL=https://snitch.example.org",
		"SNITCH_WORKER_TOKEN=rw_abc-DEF_123",
		"SNITCH_WORKER_IMAGE=ghcr.io/thomas-bsn/snitch-worker:latest",
		"TRANSCRIPTION_API_URL=A_RENSEIGNER",
		"HF_TOKEN=A_RENSEIGNER",
		"SNITCH_VOICE_DOWNLOAD_ON_START=1",
		"SNITCH_VOICE_CONFIRM_TERMS=1",
	} {
		if !strings.Contains("\n"+bundle.Env, "\n"+line+"\n") {
			t.Errorf("env lacks line %q:\n%s", line, bundle.Env)
		}
	}
	if bundle.InsecureAPIURL {
		t.Error("an https URL must not be flagged insecure")
	}
	// The model name is the library's: the deployment files never require one.
	if strings.Contains("\n"+bundle.Env, "\nTRANSCRIPTION_MODEL=") || strings.Contains(bundle.Compose, "TRANSCRIPTION_MODEL: ${TRANSCRIPTION_MODEL:?") {
		t.Errorf("the deployment files must not require a model name:\n%s", bundle.Env)
	}
	if strings.Contains(bundle.Compose, "rw_abc") {
		t.Error("the compose file must never embed the token")
	}
	for _, want := range []string{"SNITCH_WORKER_TOKEN: ${SNITCH_WORKER_TOKEN:?", "pull_policy: always", "host.docker.internal:host-gateway", "driver: nvidia", "snitch-models:/models"} {
		if !strings.Contains(bundle.Compose, want) {
			t.Errorf("compose lacks %q", want)
		}
	}
}

func TestBundleCarriesTheWhisperAddressGivenAtCreation(t *testing.T) {
	bundle, err := NewBundle(BundleInput{APIURL: "https://snitch.example.org", Token: "rw_x", Image: "img:latest", WhisperURL: "http://host.docker.internal:8000"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains("\n"+bundle.Env, "\nTRANSCRIPTION_API_URL=http://host.docker.internal:8000\n") {
		t.Errorf("env lacks the Whisper address:\n%s", bundle.Env)
	}
}

func TestBundleFlagsPlainHTTP(t *testing.T) {
	bundle, err := NewBundle(BundleInput{APIURL: "http://localhost:8080", Token: "rw_x", Image: "img:latest"})
	if err != nil {
		t.Fatal(err)
	}
	if !bundle.InsecureAPIURL {
		t.Error("an http URL must be flagged insecure")
	}
}

func TestEnvValueQuotesUnsafeValues(t *testing.T) {
	for in, want := range map[string]string{
		"plain-1.2/x:y@z": "plain-1.2/x:y@z",
		"":                `""`,
		"a b":             `"a b"`,
		`$HOME "q" \`:     `"\$HOME \"q\" \\"`,
		"a#b":             `"a#b"`,
		"l1\nl2":          `"l1\nl2"`,
	} {
		if got := envValue(in); got != want {
			t.Errorf("envValue(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestBundleRejectsMissingInputs(t *testing.T) {
	for _, in := range []BundleInput{{Token: "t", Image: "i"}, {APIURL: "https://a", Image: "i"}, {APIURL: "https://a", Token: "t"}} {
		if _, err := NewBundle(in); err == nil {
			t.Errorf("expected an error for %+v", in)
		}
	}
}
