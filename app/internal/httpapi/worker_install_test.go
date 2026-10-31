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
	data := struct{ WorkerID, APIURL, Token, Image string }{
		"wk_syntheticFixtureWorker123",
		base64.StdEncoding.EncodeToString([]byte("https://snitch.example.org")),
		base64.StdEncoding.EncodeToString([]byte(token)),
		base64.StdEncoding.EncodeToString([]byte("ghcr.io/thomas-bsn/snitch-worker:fixture")),
	}
	if err := workerInstallScript.Execute(&script, data); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "-n")
	cmd.Stdin = bytes.NewReader(script.Bytes())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("installer is not valid bash: %v: %s", err, out)
	}
	for _, fragment := range []string{"docker pull", "docker volume create", "--user 10001:10001", "HF_TOKEN", "--restart unless-stopped", "SNITCH_WORKER_TOKEN"} {
		if !strings.Contains(script.String(), fragment) {
			t.Errorf("installer lacks %q", fragment)
		}
	}
	for _, fragment := range []string{"git clone", "compose.yaml", "chmod 777"} {
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
