package httpapi

import (
	"bytes"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
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
if [ "$1 $2" = "container inspect" ] && [ ! -e "$FAKE_LOG.container" ]; then exit 1; fi
if [ "$1" = run ] && [[ "$*" == *" -d "* ]]; then : > "$FAKE_LOG.container"; fi
if [ "$1" = rm ]; then rm -f "$FAKE_LOG.container"; fi
if [ "$1" = inspect ]; then echo sha256:previous; fi
exit 0
`)
	write("uname", `echo "${FAKE_UNAME:-Linux}"`)
	write("curl", `out=''; while [ $# -gt 0 ]; do case "$1" in -o) out=$2; shift;; esac; shift; done
cat >/dev/null
status=${FAKE_CURL_STATUS:-200}
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
