package httpapi

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
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
if [[ "$*" == *"snitch_worker.bootstrap check"* ]]; then exit "${FAKE_CHECK_RC:-1}"; fi
if [[ "$*" == *"snitch_worker.bootstrap provision"* ]]; then
 prev=''; for a in "$@"; do [ "$prev" = --env-file ] && cp "$a" "$FAKE_LOG.provision-env"; prev=$a; done
 printf '%s' "${FAKE_PROVISION_OUT:-}"; exit "${FAKE_PROVISION_RC:-0}"
fi
if [ "$1 $2" = "container inspect" ] && [ ! -e "$FAKE_LOG.container" ]; then exit 1; fi
if [ "$1" = run ] && [[ "$*" == *" -d "* ]]; then : > "$FAKE_LOG.container"; fi
if [ "$1" = rm ]; then rm -f "$FAKE_LOG.container"; fi
if [ "$1" = inspect ]; then echo sha256:previous; fi
exit 0
`)
	write("uname", `echo "${FAKE_UNAME:-Linux}"`)
	write("curl", `out=''; data=''; url=''; while [ $# -gt 0 ]; do case "$1" in -o) out=$2; shift;; --data-binary) data=${2#@}; shift;; http*) url=$1;; esac; shift; done
cat >/dev/null
status=${FAKE_CURL_STATUS:-200}
case "$url" in *install-events)
 if [ "$status" = 000 ]; then printf 000; exit 7; fi
 cat "$data" >> "$FAKE_LOG.events"; printf '\n' >> "$FAKE_LOG.events"
 printf '%s' "${FAKE_EVENTS_STATUS:-204}"; exit 0;;
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
		"TranscriptionEndpoint": b64(""), "TranscriptionModel": b64(""), "TranscriptionLanguage": b64(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "-s")
	cmd.Stdin = &script
	cmd.Env = append([]string{"HOME=" + r.home, "PATH=" + r.bin + ":/usr/bin:/bin", "FAKE_LOG=" + r.log}, env...)
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
	for _, want := range []string{"SNITCH_WORKER_TOKEN=" + token, "TRANSCRIPTION_MODEL=fixture-model", "TRANSCRIPTION_API_URL=http://gpu.local:9000?a=1&b=2"} {
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
	original := strings.Fields(strings.Split(strings.TrimSpace(queued), "\n")[0])[0]
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

// Terminal driver: runs the launcher under a real pseudo-terminal (python3 pty) so that the
// /dev/tty prompts execute. answers are typed after each prompt; typed text is never echoed
// for the hidden token, so the transcript must not contain it.
const ptyDriver = `import os, pty, sys, time, select
script, answers = sys.argv[1], sys.argv[2:]
pid, fd = pty.fork()
if pid == 0:
    os.execvp("bash", ["bash", script])
out = b""
queue = list(answers)
deadline = time.time() + 20
prompts = [b"[y/N] ", b"account: ", b"(hidden): "]
pos = 0
fed = 0
while time.time() < deadline:
    r, _, _ = select.select([fd], [], [], 0.2)
    if r:
        try:
            data = os.read(fd, 4096)
        except OSError:
            break
        if not data:
            break
        out += data
    if fed < len(queue) and fed < len(prompts):
        i = out.find(prompts[fed], pos)
        if i >= 0:
            pos = i + len(prompts[fed])
            time.sleep(0.2)
            os.write(fd, (queue[fed] + "\n").encode())
            fed += 1
try:
    os.kill(pid, 9)
except OSError:
    pass
_, status = os.waitpid(pid, 0)
sys.stdout.buffer.write(out)
sys.exit(os.waitstatus_to_exitcode(status))
`

func (r *installerRun) runTTY(t *testing.T, token string, answers []string, env ...string) (transcript string, err error) {
	t.Helper()
	if _, lookErr := exec.LookPath("python3"); lookErr != nil {
		t.Skip("python3 required for the pseudo-terminal driver")
	}
	var script bytes.Buffer
	b64 := func(v string) string { return base64.StdEncoding.EncodeToString([]byte(v)) }
	if err = workerInstallScript.Execute(&script, map[string]string{
		"WorkerID": "wk_syntheticFixtureWorker123", "APIURL": b64("https://snitch.example.org"), "Token": b64(token),
		"Image": b64("ghcr.io/thomas-bsn/snitch-worker:fixture"), "TranscriptionMode": "unconfigured",
		"TranscriptionEndpoint": b64(""), "TranscriptionModel": b64(""), "TranscriptionLanguage": b64(""),
	}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	scriptPath, driverPath := filepath.Join(dir, "install.sh"), filepath.Join(dir, "driver.py")
	if err = os.WriteFile(scriptPath, script.Bytes(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(driverPath, []byte(ptyDriver), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(r.bin, "nvidia-smi"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", append([]string{"-I", driverPath, scriptPath}, answers...)...)
	cmd.Env = append([]string{"HOME=" + r.home, "PATH=" + r.bin + ":/usr/bin:/bin:/opt/homebrew/bin", "FAKE_LOG=" + r.log, "FAKE_CURL_BODY=" + fakeConfig}, env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

const hfFixtureToken = "hf_SyntheticFixtureToken0123456789"

func TestInstallerVoiceHappyPathOnTerminal(t *testing.T) {
	r := newInstallerRun(t)
	out, err := r.runTTY(t, "rw_synthetic-fixture-token", []string{"y", "accept", hfFixtureToken})
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, out)
	}
	if strings.Contains(out, hfFixtureToken) {
		t.Fatal("the token was echoed on the terminal")
	}
	envFile, _ := os.ReadFile(r.log + ".provision-env")
	if !strings.Contains(string(envFile), "HF_TOKEN="+hfFixtureToken) || !strings.Contains(string(envFile), "SNITCH_VOICE_CONFIRM_TERMS=1") {
		t.Fatalf("provisioning did not receive the token through its private env file:\n%s", envFile)
	}
	workerEnv, _ := os.ReadFile(filepath.Join(r.home, ".local/share/snitch-worker/wk_syntheticFixtureWorker123/worker.env"))
	if strings.Contains(string(workerEnv), "HF_TOKEN") || strings.Contains(string(workerEnv), hfFixtureToken) {
		t.Fatal("the token reached the normal runtime configuration")
	}
	if strings.Contains(r.dockerLog()+r.events()+r.queue(), hfFixtureToken) {
		t.Fatal("the token is in a command line or event")
	}
	files, _ := filepath.Glob(filepath.Join(r.home, ".local/share/snitch-worker/wk_syntheticFixtureWorker123/bootstrap.*"))
	if len(files) != 0 {
		t.Fatalf("temporary token file left behind: %v", files)
	}
	events := r.events()
	for _, want := range []string{`"phase":"voice-models","status":"succeeded"`, `"phase":"voice-test","status":"succeeded"`} {
		if !strings.Contains(events, want) {
			t.Errorf("events lack %s:\n%s", want, events)
		}
	}
	if !strings.Contains(r.dockerLog(), "snitch_worker.voice_selftest") {
		t.Error("synthetic inference was not run")
	}
}

func TestInstallerVoiceFailuresKeepTranscriptionAndNameTheCause(t *testing.T) {
	for _, cause := range []string{"hf_access_denied", "network_unreachable", "volume_not_writable", "hf_cache_not_writable", "gpu_unavailable"} {
		t.Run(cause, func(t *testing.T) {
			r := newInstallerRun(t)
			out, err := r.runTTY(t, "rw_synthetic-fixture-token", []string{"y", "accept", hfFixtureToken}, "FAKE_PROVISION_RC=1", "FAKE_PROVISION_OUT=cause="+cause+"\n")
			if err != nil {
				t.Fatalf("voice failure must not fail the installation: %v\n%s", err, out)
			}
			events := r.events()
			if !strings.Contains(events, fmt.Sprintf(`"phase":"voice-models","status":"failed","occurred_at":`)) || !strings.Contains(events, `"cause_code":"`+cause+`"`) {
				t.Errorf("cause %s not reported:\n%s", cause, events)
			}
			if strings.Contains(events, `"phase":"voice-test"`) {
				t.Error("voice-test must not run after a failed download")
			}
			if !strings.Contains(events, `"phase":"service","status":"succeeded"`) || strings.Contains(out+events, hfFixtureToken) {
				t.Errorf("service must start and nothing may leak:\n%s\n%s", out, events)
			}
		})
	}
	t.Run("unexpected output is unknown", func(t *testing.T) {
		r := newInstallerRun(t)
		if _, err := r.runTTY(t, "rw_synthetic-fixture-token", []string{"y", "accept", hfFixtureToken}, "FAKE_PROVISION_RC=1", "FAKE_PROVISION_OUT=cause=hf_SyntheticFixtureToken0123456789"); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(r.events(), `"cause_code":"unknown"`) || strings.Contains(r.events(), hfFixtureToken) {
			t.Fatalf("unknown cause not normalised:\n%s", r.events())
		}
	})
}

func TestInstallerVoiceRequiresExplicitTermsAcceptance(t *testing.T) {
	r := newInstallerRun(t)
	out, err := r.runTTY(t, "rw_synthetic-fixture-token", []string{"y", "no"})
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Contains(r.dockerLog(), "bootstrap provision") || strings.Contains(out, "(hidden)") {
		t.Fatal("the token was requested or a download started without explicit terms acceptance")
	}
	if !strings.Contains(r.events(), `"phase":"service","status":"succeeded"`) {
		t.Error("installation must still complete")
	}
}

func TestInstallerVoiceDeclinedAndModelsAlreadyPresent(t *testing.T) {
	r := newInstallerRun(t)
	if out, err := r.runTTY(t, "rw_synthetic-fixture-token", []string{"n"}); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Contains(r.dockerLog(), "bootstrap provision") || strings.Contains(r.dockerLog(), "voice_selftest") {
		t.Fatal("declining must download and test nothing")
	}
	// Models present (manifest reuse): no prompt, no token, no download, but the synthetic inference still runs.
	r2 := newInstallerRun(t)
	out, err := r2.runTTY(t, "rw_synthetic-fixture-token", nil, "FAKE_CHECK_RC=0")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Contains(out, "[y/N]") || strings.Contains(r2.dockerLog(), "bootstrap provision") || !strings.Contains(r2.dockerLog(), "voice_selftest") {
		t.Fatalf("existing models must be reused and still verified:\n%s\n%s", out, r2.dockerLog())
	}
}

func TestInstallerVoiceSelftestFailureIsNotReady(t *testing.T) {
	r := newInstallerRun(t)
	if out, err := r.runTTY(t, "rw_synthetic-fixture-token", []string{"y", "accept", hfFixtureToken}, "FAKE_DOCKER_FAIL=voice_selftest"); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	events := r.events()
	if !strings.Contains(events, `"phase":"voice-models","status":"succeeded"`) || !strings.Contains(events, `"phase":"voice-test","status":"failed"`) || strings.Contains(events, `"phase":"voice-test","status":"succeeded"`) {
		t.Fatalf("models downloaded without a passing inference must not count as ready:\n%s", events)
	}
}
