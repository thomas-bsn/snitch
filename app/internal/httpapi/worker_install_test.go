package httpapi

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

func TestWorkerInstallerScriptIsExecutableWithoutRepositoryFiles(t *testing.T) {
	token := "rw_synthetic-fixture-token"
	var script bytes.Buffer
	endpointB64 := base64.StdEncoding.EncodeToString([]byte("http://host.docker.internal:8000"))
	modelB64 := base64.StdEncoding.EncodeToString([]byte("whisper-large-v3"))
	languageB64 := base64.StdEncoding.EncodeToString([]byte("fr"))
	data := struct {
		WorkerID              string
		APIURL                string
		Token                 string
		Image                 string
		TranscriptionMode     string
		TranscriptionEndpoint string
		TranscriptionModel    string
		TranscriptionLanguage string
		UpdaterScript         string
		SupervisorToken       string
		SupervisorScript      string
	}{
		WorkerID:              "wk_syntheticFixtureWorker123",
		APIURL:                base64.StdEncoding.EncodeToString([]byte("https://snitch.example.org")),
		Token:                 base64.StdEncoding.EncodeToString([]byte(token)),
		Image:                 base64.StdEncoding.EncodeToString([]byte("ghcr.io/thomas-bsn/snitch-worker:fixture")),
		TranscriptionMode:     "enabled",
		TranscriptionEndpoint: endpointB64,
		TranscriptionModel:    modelB64,
		TranscriptionLanguage: languageB64,
		UpdaterScript:         base64.StdEncoding.EncodeToString([]byte(workerUpdaterScript)),
		SupervisorToken:       "",
		SupervisorScript:      "",
	}
	if err := workerInstallScript.Execute(&script, data); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "-n")
	cmd.Stdin = bytes.NewReader(script.Bytes())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("installer is not valid bash: %v: %s", err, out)
	}
	// The template contains base64-encoded values; check for the encoded forms.
	for _, fragment := range []string{"docker pull", "docker volume create", "--user 10001:10001", "--restart unless-stopped", "SNITCH_WORKER_TOKEN", endpointB64, modelB64, languageB64, "base64 -d"} {
		if !strings.Contains(script.String(), fragment) {
			t.Errorf("installer lacks %q", fragment)
		}
	}
	for _, fragment := range []string{"git clone", "compose.yaml", "chmod 777", "HF_TOKEN", "Download the voice models", "bootstrap", "DOWNLOAD_ON_START"} {
		if strings.Contains(script.String(), fragment) {
			t.Errorf("installer unexpectedly requires %q", fragment)
		}
	}
	if strings.Contains(script.String(), token) {
		t.Fatal("plain Worker token appears in the script")
	}
}

func TestWorkerInstallRouteRejectsInvalidCodesWithoutDatabaseLookup(t *testing.T) {
	s := &Server{}
	for _, code := range []string{"short", strings.Repeat("a", 44)} {
		req := httptest.NewRequest(http.MethodGet, "/api/worker/install/"+code, nil)
		req.SetPathValue("code", code)
		rec := httptest.NewRecorder()
		s.serveWorkerInstall(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("code %q: status %d", code, rec.Code)
		}
	}
}
