package httpapi

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// updaterRun runs the installed local updater (worker_updater.sh) against a fake docker and a fake
// Snitch. Nothing touches a daemon or the network. The fake records every docker call and every
// status the updater reports.
type updaterRun struct {
	dir, bin, log, reports string
	env                    []string
}

const (
	updaterWorkerID = "wk_fixture"
	digestFixture   = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	repoFixture     = "ghcr.io/thomas-bsn/snitch-worker"
)

func newUpdaterRun(t *testing.T, current, updateBody, health string) *updaterRun {
	t.Helper()
	root := t.TempDir()
	r := &updaterRun{
		dir:     filepath.Join(root, "home", ".local", "share", "snitch-worker", updaterWorkerID),
		bin:     filepath.Join(root, "bin"),
		log:     filepath.Join(root, "docker.log"),
		reports: filepath.Join(root, "reports.log"),
	}
	for _, d := range []string{r.dir, r.bin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := "SNITCH_API_URL=https://snitch.example.org\nSNITCH_WORKER_TOKEN=rw_synthetic-fixture-token\n"
	if err := os.WriteFile(filepath.Join(r.dir, "worker.env"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.dir, "updater.sh"), []byte(workerUpdaterScript), 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(r.bin, name), []byte("#!/usr/bin/env bash\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// docker: answers the exact inspect formats the updater uses; everything else succeeds.
	write("docker", `echo "$*" >> "$FAKE_LOG"
case "$*" in
 "inspect -f {{.Config.Image}} "*) [ -n "$FAKE_CURRENT" ] && echo "$FAKE_CURRENT" || exit 1 ;;
 "inspect -f {{.Image}} "*) echo "$FAKE_PREVIOUS_ID" ;;
 "inspect -f {{.State.Running}} {{.RestartCount}} "*) echo "$FAKE_HEALTH" ;;
 "image inspect -f {{index .RepoDigests 0}} "*) echo "$FAKE_REPO_DIGEST" ;;
 "run "*) [ -z "${FAKE_RUN_FAIL:-}" ] || [[ "$*" != *"$FAKE_RUN_FAIL"* ]] || exit 1 ;;
 "pull "*) [ -z "${FAKE_PULL_FAIL:-}" ] || exit 1 ;;
esac
exit 0`)
	// curl: GET /api/worker/update returns the target; the status report body is recorded.
	write("curl", `out=''; data=''; url=''; while [ $# -gt 0 ]; do case "$1" in -o) out=$2; shift;; --data-binary) data=$2; shift;; http*) url=$1;; esac; shift; done
cat >/dev/null
case "$url" in
 *update-status) printf '%s\n' "$data" >> "$FAKE_REPORTS"; printf 204; exit 0;;
 *worker/update) printf '%s' "$FAKE_UPDATE_BODY" > "$out"; printf 200; exit 0;;
esac
printf 404`)
	r.env = []string{
		"HOME=" + filepath.Join(root, "home"),
		"PATH=" + r.bin + ":/usr/bin:/bin",
		"FAKE_LOG=" + r.log,
		"FAKE_REPORTS=" + r.reports,
		"FAKE_CURRENT=" + current,
		"FAKE_PREVIOUS_ID=sha256:previousimageid",
		"FAKE_HEALTH=true 0",
		"FAKE_REPO_DIGEST=" + repoFixture + "@" + digestFixture,
		"FAKE_UPDATE_BODY=" + updateBody,
		"SNITCH_UPDATE_SETTLE=0",
	}
	if health != "" {
		r.env = append(r.env, "FAKE_HEALTH="+health)
	}
	return r
}

func (r *updaterRun) run(t *testing.T, extra ...string) string {
	t.Helper()
	cmd := exec.Command("bash", filepath.Join(r.dir, "updater.sh"))
	cmd.Env = append(append([]string{}, r.env...), extra...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("updater exited with error: %v\n%s", err, out)
	}
	return r.calls()
}

func (r *updaterRun) calls() string {
	b, _ := os.ReadFile(r.log)
	return string(b)
}

func (r *updaterRun) reported() string {
	b, _ := os.ReadFile(r.reports)
	return string(b)
}

const (
	newerTarget = `{"target_version":"v1.2.0","target_image":"ghcr.io/thomas-bsn/snitch-worker:v1.2.0"}`
	noTarget    = `{"target_version":null,"target_image":null}`
)

func TestUpdaterDoesNothingWhenNoTargetOrAlreadyCurrent(t *testing.T) {
	current := repoFixture + ":v1.2.0"
	for name, tc := range map[string]struct{ current, body string }{
		"no target":        {current, noTarget},
		"already current":  {current, newerTarget},
		"no container yet": {"", newerTarget},
	} {
		t.Run(name, func(t *testing.T) {
			r := newUpdaterRun(t, tc.current, tc.body, "")
			log := r.run(t)
			if strings.Contains(log, "pull ") || strings.Contains(log, "run ") {
				t.Fatalf("nothing to do, but docker was asked to change something:\n%s", log)
			}
			if rep := r.reported(); rep != "" {
				t.Fatalf("no update must report nothing, got %q", rep)
			}
		})
	}
}

func TestUpdaterUpdatesToTargetAndReportsDigest(t *testing.T) {
	r := newUpdaterRun(t, repoFixture+":v1.1.0", newerTarget, "")
	log := r.run(t)
	if !strings.Contains(log, "pull "+repoFixture+":v1.2.0") || !strings.Contains(log, "--name snitch-worker-"+updaterWorkerID) || !strings.Contains(log, repoFixture+"@"+digestFixture) {
		t.Fatalf("update not applied:\n%s", log)
	}
	rep := r.reported()
	for _, want := range []string{`"state":"pulling"`, `"state":"restarting"`, `"state":"updated"`, `"version":"v1.2.0"`, `"digest":"` + digestFixture + `"`} {
		if !strings.Contains(rep, want) {
			t.Fatalf("missing %s in reports:\n%s", want, rep)
		}
	}
	if restartedWith(log, "sha256:previousimageid") {
		t.Fatalf("a healthy update must not roll back:\n%s", log)
	}
}

func TestUpdaterRollsBackWhenNewContainerRestarts(t *testing.T) {
	r := newUpdaterRun(t, repoFixture+":v1.1.0", newerTarget, "true 3")
	log := r.run(t)
	if !strings.Contains(log, "run -d --name snitch-worker-"+updaterWorkerID) || !strings.Contains(log, "sha256:previousimageid") {
		t.Fatalf("previous image not restored:\n%s", log)
	}
	if rep := r.reported(); !strings.Contains(rep, `"state":"rolled_back"`) || !strings.Contains(rep, `"cause_code":"unhealthy"`) {
		t.Fatalf("rollback not reported:\n%s", rep)
	}
}

func TestUpdaterReportsFailureWhenPullFails(t *testing.T) {
	r := newUpdaterRun(t, repoFixture+":v1.1.0", newerTarget, "")
	r.env = append(r.env, "FAKE_PULL_FAIL=1")
	log := r.run(t)
	if strings.Contains(log, "--name") {
		t.Fatalf("container must not be replaced when the pull failed:\n%s", log)
	}
	if rep := r.reported(); !strings.Contains(rep, `"state":"failed"`) || !strings.Contains(rep, `"cause_code":"image_pull_failed"`) {
		t.Fatalf("pull failure not reported:\n%s", rep)
	}
}

func TestUpdaterRefusesTargetOfAnotherRepositoryAndUnknownDigest(t *testing.T) {
	foreign := `{"target_version":"v1.2.0","target_image":"evil.example/x:v1.2.0"}`
	r := newUpdaterRun(t, repoFixture+":v1.1.0", foreign, "")
	if log := r.run(t); strings.Contains(log, "pull ") {
		t.Fatalf("foreign repository must never be pulled:\n%s", log)
	}
	if rep := r.reported(); !strings.Contains(rep, `"cause_code":"target_mismatch"`) {
		t.Fatalf("mismatch not reported:\n%s", rep)
	}

	r = newUpdaterRun(t, repoFixture+":v1.1.0", newerTarget, "")
	r.env = append(r.env, "FAKE_REPO_DIGEST=")
	log := r.run(t)
	if strings.Contains(log, "--name") {
		t.Fatalf("an image without a known digest must not run:\n%s", log)
	}
	if rep := r.reported(); !strings.Contains(rep, `"cause_code":"digest_unknown"`) {
		t.Fatalf("missing digest not reported:\n%s", rep)
	}
}

// restartedWith reports whether docker run was called with the given image reference as its last argument.
func restartedWith(log, image string) bool {
	for _, line := range strings.Split(log, "\n") {
		if strings.HasPrefix(line, "run -d --name snitch-worker-"+updaterWorkerID+" ") && strings.HasSuffix(line, " "+image) {
			return true
		}
	}
	return false
}

func TestUpdaterWaitsForBusyWorkerThenGivesUp(t *testing.T) {
	busy := `{"target_version":"v1.2.0","target_image":"ghcr.io/thomas-bsn/snitch-worker:v1.2.0","idle":false}`
	r := newUpdaterRun(t, repoFixture+":v1.1.0", busy, "")
	// First run only notes the wait; a job is running, nothing is touched.
	if log := r.run(t); strings.Contains(log, "pull ") || strings.Contains(log, "rm ") {
		t.Fatalf("a busy Worker must not be restarted:\n%s", log)
	}
	if log := r.run(t); strings.Contains(log, "pull ") {
		t.Fatalf("still inside the waiting window:\n%s", log)
	}
	// After 2 hours of waiting the update goes ahead anyway: a leased job is retried by Snitch.
	old := time.Now().Add(-3 * time.Hour)
	if err := os.Chtimes(filepath.Join(r.dir, "update.deferred"), old, old); err != nil {
		t.Fatal(err)
	}
	if log := r.run(t); !strings.Contains(log, "pull "+repoFixture+":v1.2.0") {
		t.Fatalf("waiting must end after 2 hours:\n%s", log)
	}
}

func TestUpdaterDoesNotRetryAFailedVersionInALoop(t *testing.T) {
	r := newUpdaterRun(t, repoFixture+":v1.1.0", newerTarget, "true 3")
	r.run(t)
	if !strings.Contains(r.reported(), `"state":"rolled_back"`) {
		t.Fatalf("first attempt must roll back:\n%s", r.reported())
	}
	// The next cron tick sees the same target, still on the old image: it must stay silent and untouched.
	before := r.calls()
	if delta := strings.TrimPrefix(r.run(t), before); strings.Contains(delta, "pull ") || strings.Contains(delta, "run ") || strings.Contains(delta, "rm ") {
		t.Fatalf("a failed version was retried:\n%s", delta)
	}
	// A newer version is a new chance, even inside the 24 hours.
	r.env = append(r.env, `FAKE_UPDATE_BODY={"target_version":"v1.3.0","target_image":"ghcr.io/thomas-bsn/snitch-worker:v1.3.0","idle":true}`)
	if log := r.run(t); !strings.Contains(strings.TrimPrefix(log, before), "pull "+repoFixture+":v1.3.0") {
		t.Fatalf("a newer target must be tried:\n%s", log)
	}
	// The same version is retried after 24 hours.
	r.env = append(r.env, "FAKE_UPDATE_BODY="+newerTarget)
	if err := os.WriteFile(filepath.Join(r.dir, "update.failed"), []byte("v1.2.0"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-25 * time.Hour)
	_ = os.Chtimes(filepath.Join(r.dir, "update.failed"), old, old)
	mark := r.calls()
	if log := r.run(t); !strings.Contains(strings.TrimPrefix(log, mark), "pull "+repoFixture+":v1.2.0") {
		t.Fatalf("a version failed more than 24h ago must be retried:\n%s", log)
	}
}
