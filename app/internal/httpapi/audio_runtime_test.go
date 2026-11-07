package httpapi

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"snitch/internal/audioplans"
)

// The plan names the detector the server expects; the Worker reports the one it carries. If they drift, every
// diagnostic is refused as a foreign detector, so both constants are kept in step by this test.
func TestAudioDetectorVersionsMatchTheWorker(t *testing.T) {
	raw, err := os.ReadFile("../../../worker/snitch_worker/audio_diagnostic.py")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	revision := regexp.MustCompile(`(?m)^REVISION = '([0-9a-f]{40})'`).FindStringSubmatch(text)
	config := regexp.MustCompile(`(?m)^CONFIG_VERSION = '([^']+)'`).FindStringSubmatch(text)
	if revision == nil || config == nil {
		t.Fatal("the Worker detector constants moved")
	}
	if got := "silero-v6.2.1-" + revision[1]; got != audioDetectorVersion {
		t.Fatalf("detector version: worker %q, server %q", got, audioDetectorVersion)
	}
	if config[1] != audioDetectorConfigVersion {
		t.Fatalf("detector config: worker %q, server %q", config[1], audioDetectorConfigVersion)
	}
}

// The merge version of a plan is derived from the strategies the Worker adapter enables.
func TestAudioMergeVersionMatchesTheAdapter(t *testing.T) {
	raw, err := os.ReadFile("../../../worker/snitch_worker/acoustic_dedup.py")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`ENABLED_STRATEGIES: dict\[str, str\] = \{([^}]*)\}`).FindStringSubmatch(string(raw))
	if m == nil {
		t.Fatal("the adapter strategies moved")
	}
	for _, pair := range strings.Split(m[1], ",") {
		kv := strings.Split(strings.ReplaceAll(strings.TrimSpace(pair), `"`, ""), ":")
		if len(kv) == 2 && !strings.Contains(audioplans.MergeVersion(), "+"+strings.TrimSpace(kv[0])+"@"+strings.TrimSpace(kv[1])) {
			t.Fatalf("merge version %q does not carry %s", audioplans.MergeVersion(), pair)
		}
	}
}
