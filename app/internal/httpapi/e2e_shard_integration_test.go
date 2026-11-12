//go:build integration

package httpapi

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// e2eWeights are rough durations (seconds) of the browser suites, measured on CI. They only balance the shards of
// SNITCH_E2E_SHARD: a stale or missing weight makes a shard slower, never leaves a suite unrun (see shardSuites).
var e2eWeights = map[string]int{
	"shell-accessibility.spec.ts": 48, "admin-workers-live.spec.ts": 38, "admin-queue-board.spec.ts": 33,
	"library-explorer.spec.ts": 26, "people.spec.ts": 22, "library.spec.ts": 19, "admin-queue-analysis.spec.ts": 15,
	"people-directory.spec.ts": 15, "search-screen.spec.ts": 15, "admin-workers.spec.ts": 14,
}

const e2eDefaultWeight = 8

// shardSuites returns the suites of shard "i/n" (1-based). Every suite lands in exactly one shard: the heaviest
// suites are placed first, each on the currently lightest shard (greedy), ties broken by shard index and suite
// name so that all the shards, which run in separate jobs, compute the same partition.
func shardSuites(suites []string, spec string) ([]string, error) {
	parts := strings.Split(spec, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("SNITCH_E2E_SHARD %q: want i/n", spec)
	}
	index, err1 := strconv.Atoi(parts[0])
	count, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || count < 1 || index < 1 || index > count {
		return nil, fmt.Errorf("SNITCH_E2E_SHARD %q: want i/n with 1 <= i <= n", spec)
	}
	weight := func(s string) int {
		if w, ok := e2eWeights[s]; ok {
			return w
		}
		return e2eDefaultWeight
	}
	ordered := append([]string(nil), suites...)
	sort.SliceStable(ordered, func(a, b int) bool {
		if weight(ordered[a]) != weight(ordered[b]) {
			return weight(ordered[a]) > weight(ordered[b])
		}
		return ordered[a] < ordered[b]
	})
	load := make([]int, count)
	var mine []string
	for _, s := range ordered {
		lightest := 0
		for k := range load {
			if load[k] < load[lightest] {
				lightest = k
			}
		}
		load[lightest] += weight(s)
		if lightest == index-1 {
			mine = append(mine, s)
		}
	}
	return mine, nil
}

func TestShardSuitesPartition(t *testing.T) {
	suites := []string{"a.spec.ts", "b.spec.ts", "shell-accessibility.spec.ts", "c.spec.ts", "people.spec.ts", "d.spec.ts", "e.spec.ts"}
	for count := 1; count <= 9; count++ {
		seen := map[string]int{}
		for index := 1; index <= count; index++ {
			got, err := shardSuites(suites, fmt.Sprintf("%d/%d", index, count))
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range got {
				seen[s]++
			}
		}
		for _, s := range suites {
			if seen[s] != 1 {
				t.Fatalf("%d shards: suite %s ran %d times, want exactly once", count, s, seen[s])
			}
		}
	}
	for _, bad := range []string{"", "1", "0/3", "4/3", "x/3", "1/0", "1/2/3"} {
		if _, err := shardSuites(suites, bad); err == nil {
			t.Fatalf("%q must be rejected", bad)
		}
	}
}

func foundationBrowserSuites() []string {
	return []string{
		"shell-auth.spec.ts", "shell-preferences.spec.ts", "shell-accessibility.spec.ts", "shell-queue-nav.spec.ts",
		"setup-wizard.spec.ts", "account-close.spec.ts", "admin-members.spec.ts", "account-requests.spec.ts",
		"admin-transcription.spec.ts", "admin-semantic.spec.ts", "admin-benchmark.spec.ts", "admin-voice.spec.ts", "admin-voice-add.spec.ts", "admin-voice-replacement.spec.ts",
		"admin-voice-engines.spec.ts", "admin-voice-engines-recovery.spec.ts",
		"admin-voice-engine-models.spec.ts", "admin-voice-engine-uninstall.spec.ts", "admin-voice-engines-resources.spec.ts",
		"admin-queue-folders.spec.ts", "admin-folders.spec.ts", "account-settings.spec.ts",
		"admin-workers.spec.ts", "admin-workers-live.spec.ts", "admin-worker-logs-control.spec.ts", "import.spec.ts", "clip-corrections.spec.ts",
		"library.spec.ts", "search-screen.spec.ts", "people.spec.ts",
		"people-groups.spec.ts", "people-groups-disposition.spec.ts", "people-directory.spec.ts", "people-tabs.spec.ts", "people-to-process.spec.ts", "people-process.spec.ts", "segment-reindex.spec.ts", "search-corrections.spec.ts",
		"segment-export.spec.ts", "video-storage.spec.ts", "voice-rediarization.spec.ts", "voice-profile.spec.ts", "clip-analysis.spec.ts", "clip-stepper-live.spec.ts", "clip-audio-selection.spec.ts", "clip-audio-activation.spec.ts", "notifications.spec.ts", "admin-queue-analysis.spec.ts", "admin-queue-board.spec.ts", "admin-queue-timeline.spec.ts", "clip-voices.spec.ts",
		"library-explorer.spec.ts",
		"import-server.spec.ts",
	}
}

func TestEngineBrowserSuitesReachCI(t *testing.T) {
	files, err := filepath.Glob("../../../web/e2e/admin-voice-engine*.spec.ts")
	if err != nil || len(files) != 5 {
		t.Fatalf("engine browser suites: files=%v err=%v", files, err)
	}
	seen := map[string]int{}
	for index := 1; index <= 5; index++ {
		suites, err := shardSuites(foundationBrowserSuites(), fmt.Sprintf("%d/5", index))
		if err != nil {
			t.Fatal(err)
		}
		for _, suite := range suites {
			seen[suite]++
		}
	}
	for _, file := range files {
		if seen[filepath.Base(file)] != 1 {
			t.Errorf("%s reaches %d CI shards, want exactly one", file, seen[filepath.Base(file)])
		}
	}
}
