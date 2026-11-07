package httpapi

import (
	"bytes"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// installerRun renders the launcher and runs it with fake docker, curl, uname and nvidia-smi
// executables first in PATH. Nothing touches a real Docker daemon, network or library.
type installerRun struct {
	home, bin, log string
}

func newInstallerRun(t *testing.T) *installerRun {
	t.Helper()
	root := t.TempDir()
	r := &installerRun{home: filepath.Join(root, "home"), bin: filepath.Join(root, "bin"), log: filepath.Join(root, "docker.log")}
	for _, d := range []string{r.home, r.bin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(r.bin, name), []byte("#!/usr/bin/env bash\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write("docker", `echo "$*" >> "$FAKE_LOG"
if [ -n "${FAKE_DOCKER_FAIL:-}" ] && [[ "$*" =~ $FAKE_DOCKER_FAIL ]]; then exit 1; fi
if [ "$1 $2" = "container inspect" ] && [ ! -e "$FAKE_LOG.container" ]; then exit 1; fi
if [ "$1" = run ] && [[ "$*" == *" -d "* ]]; then : > "$FAKE_LOG.container"; fi
if [ "$1" = rm ]; then rm -f "$FAKE_LOG.container"; fi
if [ "$1" = inspect ]; then echo sha256:previous; fi
exit 0
`)
	// crontab is faked: the installer schedules the updater, and a test must never edit the real crontab.
	write("crontab", `case "${1:-}" in -l) [ -f "$FAKE_CRON" ] && cat "$FAKE_CRON"; exit 0;; *) cat > "$FAKE_CRON";; esac`)
	write("uname", `if [ "${1:-}" = -r ]; then echo "${FAKE_UNAME_R:-6.8.0-generic}"; else echo "${FAKE_UNAME:-Linux}"; fi`)
	write("curl", `out=''; data=''; url=''; while [ $# -gt 0 ]; do case "$1" in -o) out=$2; shift;; --data-binary) data=${2#@}; shift;; http*) url=$1;; esac; shift; done
cat >/dev/null
status=${FAKE_CURL_STATUS:-200}
case "$url" in *install-events)
 if [ "$status" = 000 ]; then printf 000; exit 7; fi
 cat "$data" >> "$FAKE_LOG.events"; printf '\n' >> "$FAKE_LOG.events"
 printf '%s' "${FAKE_EVENTS_STATUS:-204}"; exit 0;;
esac
case "$url" in *worker/update)
 if [ "$status" = 200 ]; then printf '%s' "${FAKE_UPDATE_BODY:-}" > "$out"; fi
 printf '%s' "$status"; exit 0;;
esac
case "$url" in *worker/revocation)
 rstatus=${FAKE_REVOCATION_STATUS:-401}
 if [ "$rstatus" = 200 ]; then printf '%s' "${FAKE_REVOCATION_BODY:-}" > "$out"; fi
 printf '%s' "$rstatus"; exit 0;;
esac
if [ "$status" = 200 ]; then printf '%s' "${FAKE_CURL_BODY:-}" > "$out"; fi
printf '%s' "$status"
[ "$status" = 000 ] && exit 7
exit 0
`)
	return r
}

func (r *installerRun) run(t *testing.T, token string, env ...string) (stdout, stderr string, err error) {
	t.Helper()
	var script bytes.Buffer
	b64 := func(v string) string { return base64.StdEncoding.EncodeToString([]byte(v)) }
	err = workerInstallScript.Execute(&script, map[string]string{
		"WorkerID": "wk_syntheticFixtureWorker123", "APIURL": b64("https://snitch.example.org"), "Token": b64(token),
		"Image": b64("ghcr.io/thomas-bsn/snitch-worker:fixture"), "TranscriptionMode": "unconfigured",
		"TranscriptionEndpoint": b64(""), "TranscriptionModel": b64(""), "TranscriptionLanguage": b64(""), "UpdaterScript": b64(workerUpdaterScript), "SupervisorToken": "", "SupervisorScript": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "-s")
	cmd.Stdin = &script
	cmd.Env = append([]string{"HOME=" + r.home, "PATH=" + r.bin + ":/usr/bin:/bin", "FAKE_LOG=" + r.log, "FAKE_CRON=" + r.log + ".cron"}, env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // no controlling terminal: the voice prompt is skipped
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err = cmd.Run()
	return out.String(), errb.String(), err
}

func (r *installerRun) dockerLog() string {
	data, _ := os.ReadFile(r.log)
	return string(data)
}

const fakeConfig = `{"mode":"enabled","endpoint":"http://gpu.local:9000?a=1\u0026b=2","model":"fixture-model","language":"fr","version":"v1"}`

func TestInstallerFirstRunAndResumeWithFakeDocker(t *testing.T) {
	token := "rw_synthetic-fixture-token"
	r := newInstallerRun(t)
	stdout, stderr, err := r.run(t, token, "FAKE_CURL_BODY="+fakeConfig)
	if err != nil {
		t.Fatalf("first run failed: %v\n%s\n%s", err, stdout, stderr)
	}
	envFile := filepath.Join(r.home, ".local/share/snitch-worker/wk_syntheticFixtureWorker123/worker.env")
	info, err := os.Stat(envFile)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("worker.env must be mode 600: %v %v", info, err)
	}
	content, _ := os.ReadFile(envFile)
	for _, want := range []string{"SNITCH_WORKER_TOKEN=" + token, "TRANSCRIPTION_MODEL=fixture-model", "TRANSCRIPTION_API_URL=http://gpu.local:9000?a=1&b=2", "SNITCH_VOICE_DIARIZATION_MODEL_PATH=/models/diarization", "SNITCH_VOICE_EMBEDDING_MODEL_PATH=/models/embedding"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("worker.env lacks %q", want)
		}
	}
	log := r.dockerLog()
	for _, want := range []string{"pull ghcr.io/thomas-bsn/snitch-worker:fixture", "volume create snitch-models-wk_syntheticFixtureWorker123", "--user 10001:10001", "-d --name snitch-worker-wk_syntheticFixtureWorker123 --restart unless-stopped"} {
		if !strings.Contains(log, want) {
			t.Errorf("docker log lacks %q:\n%s", want, log)
		}
	}
	if strings.Contains(log+stdout+stderr, token) {
		t.Fatal("token leaked in docker arguments or output")
	}

	// Resume: the identity stays the one in worker.env even if the launcher carries another token;
	// configuration follows Snitch; the existing service is replaced, the volume reused.
	changed := strings.Replace(fakeConfig, "fixture-model", "newer-model", 1)
	if stdout, stderr, err = r.run(t, "rw_other-token-never-used", "FAKE_CURL_BODY="+changed); err != nil {
		t.Fatalf("resume failed: %v\n%s\n%s", err, stdout, stderr)
	}
	content, _ = os.ReadFile(envFile)
	if !strings.Contains(string(content), "SNITCH_WORKER_TOKEN="+token) || strings.Contains(string(content), "never-used") || !strings.Contains(string(content), "TRANSCRIPTION_MODEL=newer-model") {
		t.Fatalf("resume did not keep identity and refresh configuration:\n%s", content)
	}
	if !strings.Contains(r.dockerLog(), "rm -f snitch-worker-wk_syntheticFixtureWorker123") {
		t.Fatal("resume did not replace the existing service")
	}
}

// A deleted Worker: Snitch refuses the token, the narrow confirmation says it is revoked, and the
// launcher stops the service without touching the volume, the models or the local files.
func TestInstallerStopsServiceOnRevocationConfirmationWithoutErasingLocalData(t *testing.T) {
	token := "rw_synthetic-fixture-token"
	r := newInstallerRun(t)
	if _, stderr, err := r.run(t, token, "FAKE_CURL_BODY="+fakeConfig); err != nil {
		t.Fatalf("first run: %v %s", err, stderr)
	}
	stdout, stderr, err := r.run(t, token, "FAKE_CURL_STATUS=401", "FAKE_REVOCATION_STATUS=200", `FAKE_REVOCATION_BODY={"revoked":true,"revoked_at":"2026-11-01T10:00:00Z"}`)
	if err != nil {
		t.Fatalf("a confirmed revocation is not a failure: %v\n%s\n%s", err, stdout, stderr)
	}
	log := r.dockerLog()
	if !strings.HasSuffix(strings.TrimSpace(log), "rm -f snitch-worker-wk_syntheticFixtureWorker123") {
		t.Fatalf("service not stopped as the last action:\n%s", log)
	}
	for _, forbidden := range []string{"volume rm", "volume prune", "rmi", "system prune"} {
		if strings.Contains(log, forbidden) {
			t.Fatalf("launcher erased local data (%s):\n%s", forbidden, log)
		}
	}
	for _, want := range []string{"launcher erased nothing", "lanceur n’a rien effacé", "erases its own models", "service was stopped", "arrêté"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("message lacks %q:\n%s", want, stdout)
		}
	}
	if _, err = os.Stat(filepath.Join(r.home, ".local/share/snitch-worker/wk_syntheticFixtureWorker123/worker.env")); err != nil {
		t.Fatal("the local configuration must stay in place")
	}
	if strings.Contains(stdout+stderr+log, token) {
		t.Fatal("token exposed")
	}
	// An unknown credential is not a revocation: the old rejection message stays.
	if _, stderr, err = r.run(t, token, "FAKE_CURL_STATUS=401"); err == nil || !strings.Contains(stderr, "[registration_rejected]") {
		t.Fatalf("unconfirmed 401 must keep the rejection: %v %s", err, stderr)
	}
}

func TestInstallerUnconfiguredTranscriptionStillInstalls(t *testing.T) {
	r := newInstallerRun(t)
	body := `{"mode":"unconfigured","endpoint":"","model":"","language":"","version":""}`
	if stdout, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+body); err != nil {
		t.Fatalf("install failed: %v\n%s\n%s", err, stdout, stderr)
	}
	if !strings.Contains(r.dockerLog(), "-d --name") {
		t.Fatal("service was not started")
	}
}

func TestInstallerFailuresNameStepCauseAndRecovery(t *testing.T) {
	token := "rw_synthetic-fixture-token"
	cases := []struct {
		name, stage, code, recoveryEN, recoveryFR string
		env                                       []string
		gpu                                       bool
	}{
		{"unsupported platform", "prerequisites", "unsupported_platform", "Linux machine", "machine Linux", []string{"FAKE_UNAME=Darwin"}, false},
		{"docker unavailable", "prerequisites", "docker_unavailable", "Start Docker", "Démarrez Docker", []string{"FAKE_DOCKER_FAIL=^info"}, false},
		{"registration rejected", "registration", "registration_rejected", "new installation command", "nouvelle commande", []string{"FAKE_CURL_STATUS=401"}, false},
		{"api unreachable", "registration", "network_unreachable", "Check the network", "Vérifiez le réseau", []string{"FAKE_CURL_STATUS=000"}, false},
		{"image pull", "image", "image_pull_failed", "registry access", "accès au registre", []string{"FAKE_DOCKER_FAIL=^(pull|image inspect)"}, false},
		{"volume ownership", "permissions", "volume_not_writable", "UID 10001", "UID 10001", []string{"FAKE_DOCKER_FAIL=chown"}, false},
		{"volume not writable", "permissions", "volume_not_writable", "not writable by UID 10001", "pas inscriptible par l’UID 10001", []string{`FAKE_DOCKER_FAIL=touch /models/\.snitch-write`}, false},
		{"hf cache", "permissions", "hf_cache_not_writable", "Hugging Face/Xet cache", "cache Hugging Face/Xet", []string{`FAKE_DOCKER_FAIL=xet/\.snitch-write`}, false},
		{"gpu", "gpu", "gpu_unavailable", "NVIDIA Container Toolkit", "NVIDIA Container Toolkit", []string{"FAKE_DOCKER_FAIL=--gpus all"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newInstallerRun(t)
			if c.gpu {
				if err := os.WriteFile(filepath.Join(r.bin, "nvidia-smi"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			stdout, stderr, err := r.run(t, token, append([]string{"FAKE_CURL_BODY=" + fakeConfig}, c.env...)...)
			if err == nil {
				t.Fatalf("expected failure\n%s", stdout)
			}
			for _, want := range []string{`step "` + c.stage + `"`, "[" + c.code + "]", c.recoveryEN, c.recoveryFR} {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr lacks %q:\n%s", want, stderr)
				}
			}
			if strings.Contains(stdout+stderr+r.dockerLog(), token) {
				t.Error("token exposed")
			}
		})
	}
}

func TestInstallerFailedServiceStartRestoresPreviousVersion(t *testing.T) {
	r := newInstallerRun(t)
	if _, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+fakeConfig); err != nil {
		t.Fatalf("first run: %v %s", err, stderr)
	}
	_, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+fakeConfig, `FAKE_DOCKER_FAIL=-d .*snitch-worker:fixture$`)
	if err == nil || !strings.Contains(stderr, "[service_start_failed]") || !strings.Contains(stderr, "previous version was restored") {
		t.Fatalf("unexpected result: %v\n%s", err, stderr)
	}
	if !strings.Contains(r.dockerLog(), "sha256:previous") {
		t.Fatal("previous image was not restarted")
	}
}

func (r *installerRun) events() string {
	data, _ := os.ReadFile(r.log + ".events")
	return string(data)
}

func (r *installerRun) queue() string {
	data, _ := os.ReadFile(filepath.Join(r.home, ".local/share/snitch-worker/wk_syntheticFixtureWorker123/events.queue"))
	return string(data)
}

func TestInstallerEmitsPhaseEventsWithoutSecrets(t *testing.T) {
	token := "rw_synthetic-fixture-token"
	r := newInstallerRun(t)
	if stdout, stderr, err := r.run(t, token, "FAKE_CURL_BODY="+fakeConfig); err != nil {
		t.Fatalf("install failed: %v\n%s\n%s", err, stdout, stderr)
	}
	events := r.events()
	for _, phase := range []string{"prerequisites", "registration", "image", "permissions", "service"} {
		if !strings.Contains(events, `"phase":"`+phase+`","status":"succeeded"`) {
			t.Errorf("no succeeded event for %s:\n%s", phase, events)
		}
	}
	if !regexp.MustCompile(`"occurred_at":"20\d\d-\d\d-\d\dT\d\d:\d\d:\d\dZ"`).MatchString(events) {
		t.Errorf("events lack their original timestamp:\n%s", events)
	}
	if strings.Contains(events+r.dockerLog(), token) || strings.Contains(events, "message") || r.queue() != "" {
		t.Errorf("events leak a secret or free text, or the queue was not flushed:\n%s", events)
	}
}

func TestInstallerQueuesEventsOfflineAndSendsThemWithOriginalTimestamp(t *testing.T) {
	token := "rw_synthetic-fixture-token"
	r := newInstallerRun(t)
	// Offline: the run fails at registration; the failure is kept locally, nothing is sent.
	if _, _, err := r.run(t, token, "FAKE_CURL_STATUS=000"); err == nil {
		t.Fatal("expected network failure")
	}
	queued := r.queue()
	if !strings.Contains(queued, " registration failed network_unreachable") || r.events() != "" {
		t.Fatalf("offline failure not queued (events=%q):\n%s", r.events(), queued)
	}
	var original string
	for _, line := range strings.Split(strings.TrimSpace(queued), "\n") {
		if f := strings.Fields(line); len(f) == 4 && f[1] == "registration" {
			original = f[0]
		}
	}
	// Back online: the queued event goes out first, with the timestamp it had when it happened.
	if stdout, stderr, err := r.run(t, token, "FAKE_CURL_BODY="+fakeConfig); err != nil {
		t.Fatalf("rerun failed: %v\n%s\n%s", err, stdout, stderr)
	}
	events := r.events()
	if !strings.Contains(events, `"phase":"registration","status":"failed","occurred_at":"`+original+`","cause_code":"network_unreachable"`) {
		t.Fatalf("queued event not delivered with its original timestamp %s:\n%s", original, events)
	}
	if r.queue() != "" {
		t.Fatal("queue not emptied after delivery")
	}
}

func TestInstallerReportsFailuresToSnitchOnceReachable(t *testing.T) {
	r := newInstallerRun(t)
	if _, _, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+fakeConfig, "FAKE_DOCKER_FAIL=^(pull|image inspect)"); err == nil {
		t.Fatal("expected image failure")
	}
	if !strings.Contains(r.events(), `"phase":"image","status":"failed"`) || !strings.Contains(r.events(), `"cause_code":"image_pull_failed"`) {
		t.Fatalf("failure not reported:\n%s", r.events())
	}
}

func (r *installerRun) runTTY(t *testing.T, token string, answers []string, env ...string) (transcript string, err error) {
	t.Helper()
	var script bytes.Buffer
	b64 := func(v string) string { return base64.StdEncoding.EncodeToString([]byte(v)) }
	if err = workerInstallScript.Execute(&script, map[string]string{
		"WorkerID": "wk_syntheticFixtureWorker123", "APIURL": b64("https://snitch.example.org"), "Token": b64(token),
		"Image": b64("ghcr.io/thomas-bsn/snitch-worker:fixture"), "TranscriptionMode": "unconfigured",
		"TranscriptionEndpoint": b64(""), "TranscriptionModel": b64(""), "TranscriptionLanguage": b64(""), "UpdaterScript": b64(workerUpdaterScript), "SupervisorToken": "", "SupervisorScript": "",
	}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "install.sh")
	if err = os.WriteFile(scriptPath, script.Bytes(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(r.bin, "nvidia-smi"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", scriptPath)
	cmd.Env = append([]string{"HOME=" + r.home, "PATH=" + r.bin + ":/usr/bin:/bin:/opt/homebrew/bin", "FAKE_LOG=" + r.log, "FAKE_CRON=" + r.log + ".cron", "FAKE_CURL_BODY=" + fakeConfig}, env...)
	return driveOnPTY(t, cmd, answers)
}

// driveOnPTY runs cmd under a real pseudo-terminal (opened in Go, no external interpreter) so
// that the /dev/tty prompts execute. answers are typed after each prompt; typed text is never
// echoed for the hidden token, so the transcript must not contain it.
func driveOnPTY(t *testing.T, cmd *exec.Cmd, answers []string) (string, error) {
	t.Helper()
	master, slavePath, err := openPTY()
	if err != nil {
		t.Fatalf("cannot open a pseudo-terminal: %v", err)
	}
	defer master.Close()
	slave, err := os.OpenFile(slavePath, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err = cmd.Start(); err != nil {
		slave.Close()
		t.Fatal(err)
	}
	slave.Close()

	var mu sync.Mutex
	var out []byte
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buf := make([]byte, 4096)
		for {
			n, rerr := master.Read(buf)
			mu.Lock()
			out = append(out, buf[:n]...)
			mu.Unlock()
			if rerr != nil {
				return
			}
		}
	}()
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	prompts := []string{"[y/N] ", "account: ", "(hidden): "}
	pos, fed := 0, 0
	deadline := time.After(20 * time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	var waitErr error
loop:
	for {
		select {
		case waitErr = <-waitDone:
			break loop
		case <-deadline:
			_ = cmd.Process.Kill()
			waitErr = <-waitDone
			break loop
		case <-tick.C:
			if fed < len(answers) && fed < len(prompts) {
				mu.Lock()
				i := strings.Index(string(out[pos:]), prompts[fed])
				if i >= 0 {
					pos += i + len(prompts[fed])
				}
				mu.Unlock()
				if i >= 0 {
					time.Sleep(200 * time.Millisecond)
					_, _ = master.Write([]byte(answers[fed] + "\n"))
					fed++
				}
			}
		}
	}
	// The child is gone: wait for the reader to drain the pty (best effort).
	select {
	case <-readDone:
	case <-time.After(time.Second):
	}
	mu.Lock()
	defer mu.Unlock()
	return string(out), waitErr
}

const hfFixtureToken = "hf_SyntheticFixtureToken0123456789"

// A new Worker holds no voice model (ADR 0038): the installation never asks about, downloads or tests one,
// whatever the platform, and whatever a person types at the terminal.
func TestInstallerDownloadsNoVoiceModelAndAsksNothingAboutThem(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{"native Linux with a GPU", nil},
		{"WSL2 with a GPU", []string{"FAKE_UNAME_R=" + wsl2Release}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newInstallerRun(t)
			out, err := r.runTTY(t, "rw_synthetic-fixture-token", []string{"y", "accept", hfFixtureToken}, tc.env...)
			if err != nil {
				t.Fatalf("run failed: %v\n%s", err, out)
			}
			for _, banned := range []string{"Download the voice models", "[y/N]", "(hidden)", "accept"} {
				if strings.Contains(out, banned) {
					t.Errorf("the installation still talks about voice models (%q):\n%s", banned, out)
				}
			}
			log := r.dockerLog()
			for _, banned := range []string{"snitch_worker.bootstrap", "voice_selftest", "provision"} {
				if strings.Contains(log, banned) {
					t.Errorf("docker was asked to prepare voice models (%q):\n%s", banned, log)
				}
			}
			events := r.events()
			for _, banned := range []string{`"phase":"voice-models"`, `"phase":"voice-test"`} {
				if strings.Contains(events, banned) {
					t.Errorf("event %s emitted without any model:\n%s", banned, events)
				}
			}
			if !strings.Contains(events, `"phase":"gpu","status":"succeeded"`) || !strings.Contains(events, `"phase":"service","status":"succeeded"`) {
				t.Errorf("the GPU check and the service must still run:\n%s", events)
			}
			workerEnv, _ := os.ReadFile(filepath.Join(r.home, ".local/share/snitch-worker/wk_syntheticFixtureWorker123/worker.env"))
			for _, banned := range []string{"HF_TOKEN", hfFixtureToken, "DOWNLOAD_ON_START", "CONFIRM_TERMS"} {
				if strings.Contains(string(workerEnv), banned) {
					t.Errorf("worker.env carries %q", banned)
				}
			}
			for _, want := range []string{"SNITCH_VOICE_DIARIZATION_MODEL_PATH=/models/diarization", "HF_XET_CACHE=/models/.cache/huggingface/xet"} {
				if !strings.Contains(string(workerEnv), want) {
					t.Errorf("worker.env lost the #262 setting %q:\n%s", want, workerEnv)
				}
			}
			if strings.Contains(out+events+r.dockerLog(), hfFixtureToken) {
				t.Fatal("the typed token leaked")
			}
		})
	}
}

const wsl2Release = "5.15.153.1-microsoft-standard-WSL2"

func (r *installerRun) fakeNvidia(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(r.bin, "nvidia-smi"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestInstallerNativeLinuxRunsNoWSLPreflight(t *testing.T) {
	r := newInstallerRun(t)
	r.fakeNvidia(t)
	if stdout, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+fakeConfig); err != nil {
		t.Fatalf("install failed: %v\n%s\n%s", err, stdout, stderr)
	}
	if log := r.dockerLog(); strings.Contains(log, "urlopen") || strings.Contains(log, "libcuda") {
		t.Fatalf("native Linux must keep the P0 contract, WSL preflights ran:\n%s", log)
	}
}

func TestInstallerWSL2DetectionAndHappyPath(t *testing.T) {
	r := newInstallerRun(t)
	r.fakeNvidia(t)
	stdout, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+fakeConfig, "FAKE_UNAME_R="+wsl2Release)
	if err != nil {
		t.Fatalf("WSL2 install failed: %v\n%s\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "WSL2 detected") {
		t.Errorf("WSL2 not announced:\n%s", stdout)
	}
	log := r.dockerLog()
	for _, want := range []string{"urlopen", "snitch.example.org", "gpu.local:9000", "libcuda", "-d --name"} {
		if !strings.Contains(log, want) {
			t.Errorf("docker log lacks %q:\n%s", want, log)
		}
	}
}

func TestInstallerWSL2DetectionFromProcVersion(t *testing.T) {
	r := newInstallerRun(t)
	proc := filepath.Join(t.TempDir(), "version")
	if err := os.WriteFile(proc, []byte("Linux version 5.15.0 (root@build) #1 SMP microsoft-standard-WSL2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+fakeConfig, "SNITCH_PROC_VERSION="+proc)
	if err != nil || !strings.Contains(stdout, "WSL2 detected") {
		t.Fatalf("proc/version detection failed: %v\n%s\n%s", err, stdout, stderr)
	}
}

func TestInstallerWSL1IsRefused(t *testing.T) {
	r := newInstallerRun(t)
	_, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+fakeConfig, "FAKE_UNAME_R=4.4.0-19041-Microsoft")
	if err == nil {
		t.Fatal("WSL1 must be refused")
	}
	for _, want := range []string{`step "prerequisites"`, "[unsupported_platform]", "wsl --set-version", "wsl --set-version"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr lacks %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(r.dockerLog(), "pull") {
		t.Error("nothing must run after the refusal")
	}
}

func TestInstallerWSL2DiagnosticsNamePreflightAndRecovery(t *testing.T) {
	cases := []struct {
		name, stage, code, preflight, recoveryEN, recoveryFR string
		env                                                  []string
	}{
		{"docker integration", "prerequisites", "docker_unavailable", "wsl-docker", "WSL integration", "intégration WSL", []string{"FAKE_DOCKER_FAIL=^info"}},
		{"snitch unreachable from runtime", "image", "network_unreachable", "snitch-reachability", "from inside Docker", "depuis Docker", []string{`FAKE_DOCKER_FAIL=urlopen.*snitch\.example\.org`}},
		{"transcription unreachable from runtime", "image", "network_unreachable", "transcription-reachability", "transcription endpoint", "point d’accès de transcription", []string{`FAKE_DOCKER_FAIL=urlopen.*gpu\.local`}},
		{"cuda", "gpu", "gpu_unavailable", "docker-cuda", "Windows NVIDIA driver", "pilote NVIDIA Windows", []string{"FAKE_DOCKER_FAIL=libcuda"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := newInstallerRun(t)
			r.fakeNvidia(t)
			_, stderr, err := r.run(t, "rw_synthetic-fixture-token", append([]string{"FAKE_CURL_BODY=" + fakeConfig, "FAKE_UNAME_R=" + wsl2Release}, c.env...)...)
			if err == nil {
				t.Fatal("expected failure")
			}
			for _, want := range []string{`step "` + c.stage + `"`, "[" + c.code + "]", "preflight " + c.preflight, c.recoveryEN, c.recoveryFR} {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr lacks %q:\n%s", want, stderr)
				}
			}
			// Before registration the failure is queued locally (ADR 0021); afterwards Snitch receives it.
			if reported := r.events() + r.queue(); !strings.Contains(reported, c.code) {
				t.Errorf("failure not recorded with %s:\n%s", c.code, reported)
			}
		})
	}
}

func TestInstallerWSL2LoopbackEndpointGetsAHint(t *testing.T) {
	r := newInstallerRun(t)
	body := strings.Replace(fakeConfig, "http://gpu.local:9000?a=1\\u0026b=2", "http://localhost:9000", 1)
	_, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+body, "FAKE_UNAME_R="+wsl2Release, `FAKE_DOCKER_FAIL=urlopen.*localhost`)
	if err == nil || !strings.Contains(stderr, "localhost") || !strings.Contains(stderr, "host.docker.internal") {
		t.Fatalf("loopback hint missing: %v\n%s", err, stderr)
	}
}

func TestInstallerWSL2RefusesWindowsMountStorage(t *testing.T) {
	r := newInstallerRun(t)
	// Synthetic Windows-mounted home: the launcher state must live on the Linux filesystem.
	mnt := filepath.Join(t.TempDir(), "mnt", "c", "Users", "fixture")
	if err := os.MkdirAll(mnt, 0o755); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+fakeConfig, "FAKE_UNAME_R="+wsl2Release, "SNITCH_WSL_MOUNT_PREFIX="+filepath.Dir(filepath.Dir(filepath.Dir(mnt)))+"/", "HOME="+mnt)
	if err == nil {
		t.Fatal("expected refusal on /mnt/c storage")
	}
	for _, want := range []string{`step "prerequisites"`, "[volume_not_writable]", "preflight wsl-storage", "cd ~", "cd ~"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr lacks %q:\n%s", want, stderr)
		}
	}
	if !strings.Contains(stderr, "Linux home") || !strings.Contains(stderr, "dossier personnel Linux") {
		t.Errorf("recovery text missing:\n%s", stderr)
	}
}

// The launcher follows the version chosen in Snitch, but only a released tag of the invitation's repository.
func TestInstallerUsesTargetVersionOfSameRepository(t *testing.T) {
	const good = `{"target_version":"v1.2.0","target_image":"ghcr.io/thomas-bsn/snitch-worker:v1.2.0"}`
	for name, tc := range map[string]struct {
		body, wantPull string
	}{
		"target followed":                  {good, "pull ghcr.io/thomas-bsn/snitch-worker:v1.2.0"},
		"foreign image refused":            {`{"target_version":"v1.2.0","target_image":"evil.example/x:v1.2.0"}`, "pull ghcr.io/thomas-bsn/snitch-worker:fixture"},
		"non release tag refused":          {`{"target_version":"latest","target_image":"ghcr.io/thomas-bsn/snitch-worker:latest"}`, "pull ghcr.io/thomas-bsn/snitch-worker:fixture"},
		"no target keeps invitation image": {`{"target_version":null,"target_image":null}`, "pull ghcr.io/thomas-bsn/snitch-worker:fixture"},
	} {
		t.Run(name, func(t *testing.T) {
			r := newInstallerRun(t)
			r.fakeNvidia(t)
			if stdout, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+fakeConfig, "FAKE_UPDATE_BODY="+tc.body); err != nil {
				t.Fatalf("install failed: %v\n%s\n%s", err, stdout, stderr)
			}
			if log := r.dockerLog(); !strings.Contains(log, tc.wantPull+"\n") {
				t.Fatalf("want %q in docker log:\n%s", tc.wantPull, log)
			}
		})
	}
}

// The launcher installs the local updater and schedules it every 10 minutes, once.
func TestInstallerSchedulesLocalUpdaterOnce(t *testing.T) {
	r := newInstallerRun(t)
	r.fakeNvidia(t)
	for i := 0; i < 2; i++ {
		if stdout, stderr, err := r.run(t, "rw_synthetic-fixture-token", "FAKE_CURL_BODY="+fakeConfig); err != nil {
			t.Fatalf("install %d failed: %v\n%s\n%s", i, err, stdout, stderr)
		}
	}
	updater := filepath.Join(r.home, ".local", "share", "snitch-worker", "wk_syntheticFixtureWorker123", "updater.sh")
	info, err := os.Stat(updater)
	if err != nil {
		t.Fatalf("updater not installed: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("updater mode %v, want 700", info.Mode().Perm())
	}
	cron, _ := os.ReadFile(r.log + ".cron")
	if got := strings.Count(string(cron), updater); got != 1 {
		t.Fatalf("updater scheduled %d times, want once:\n%s", got, cron)
	}
	if !strings.Contains(string(cron), `*/10 * * * * "`+updater+`"`) {
		t.Fatalf("schedule not every 10 minutes:\n%s", cron)
	}
}
