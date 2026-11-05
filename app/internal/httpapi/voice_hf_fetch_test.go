package httpapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeHF serves synthetic Hugging Face metadata; the inspector of a test is pointed at it.
func fakeHF(t *testing.T, handler http.HandlerFunc) (*hfInspector, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	host := strings.TrimPrefix(ts.URL, "http://")
	return &hfInspector{host: host, scheme: "http", timeout: 2 * time.Second, maxBytes: 4096, allowInternal: true}, ts
}

const syntheticMeta = `{"id":"acme/voice","sha":"cd03eee90fbec18297ac31b8c21546e596b7f71c","gated":false,"tags":["nemo"],"siblings":[{"rfilename":"model.nemo"}],"cardData":{"license":"apache-2.0"}}`

func TestHFFetchReadsPublicMetadata(t *testing.T) {
	var path string
	in, _ := fakeHF(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if r.Header.Get("Authorization") != "" {
			t.Error("the inspection never sends a credential")
		}
		_, _ = w.Write([]byte(syntheticMeta))
	})
	meta, err := in.fetch(context.Background(), "acme/voice", "")
	if err != nil || meta.SHA != "cd03eee90fbec18297ac31b8c21546e596b7f71c" || meta.License() != "apache-2.0" || path != "/api/models/acme/voice" {
		t.Fatalf("%+v %v %q", meta, err, path)
	}
	if _, err = in.fetch(context.Background(), "acme/voice", "main"); err != nil || path != "/api/models/acme/voice/revision/main" {
		t.Fatalf("%v %q", err, path)
	}
}

func TestHFFetchFailures(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		want    error
	}{
		{"unknown repository", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }, errHFNotFound},
		{"private or denied", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) }, errHFDenied},
		{"forbidden", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }, errHFDenied},
		{"server error", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) }, errHFUnavailable},
		{"not json", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<html>")) }, errHFUnavailable},
		{"oversized response", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"id":"` + strings.Repeat("a", 5000) + `"}`))
		}, errHFTooLarge},
		{"too slow", func(w http.ResponseWriter, r *http.Request) { time.Sleep(600 * time.Millisecond) }, errHFUnavailable},
		{"no commit", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"id":"acme/voice","sha":"main"}`)) }, errHFUnavailable},
	}
	for _, c := range cases {
		in, _ := fakeHF(t, c.handler)
		in.timeout = 300 * time.Millisecond
		if _, err := in.fetch(context.Background(), "acme/voice", ""); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v want %v", c.name, err, c.want)
		}
	}
}

// A redirection to another host is never followed; a redirection inside the host (a renamed repository) is.
func TestHFFetchRedirects(t *testing.T) {
	var elsewhere atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elsewhere.Add(1)
		_, _ = w.Write([]byte(syntheticMeta))
	}))
	t.Cleanup(other.Close)
	in, _ := fakeHF(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/models/acme/offsite":
			http.Redirect(w, r, other.URL+"/api/models/acme/voice", http.StatusFound)
		case "/api/models/acme/old":
			http.Redirect(w, r, "/api/models/acme/voice", http.StatusMovedPermanently)
		default:
			_, _ = w.Write([]byte(syntheticMeta))
		}
	})
	if _, err := in.fetch(context.Background(), "acme/offsite", ""); !errors.Is(err, errHFUnavailable) || elsewhere.Load() != 0 {
		t.Fatalf("a redirection to another host must be refused: %v, %d hits", err, elsewhere.Load())
	}
	if _, err := in.fetch(context.Background(), "acme/old", ""); err != nil {
		t.Fatalf("a redirection inside the host is followed: %v", err)
	}
}

// Internal addresses are never dialled, whatever the name resolves to.
func TestHFFetchRefusesInternalAddresses(t *testing.T) {
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	t.Cleanup(ts.Close)
	in := &hfInspector{host: strings.TrimPrefix(ts.URL, "http://"), scheme: "http", timeout: time.Second, maxBytes: 4096}
	if _, err := in.fetch(context.Background(), "acme/voice", ""); !errors.Is(err, errHFUnavailable) || hits.Load() != 0 {
		t.Fatalf("loopback must be refused: %v, %d hits", err, hits.Load())
	}
	for _, ip := range []string{"127.0.0.1", "10.1.2.3", "172.16.0.9", "192.168.1.1", "169.254.169.254", "::1", "fe80::1", "fc00::1", "0.0.0.0", "100.64.0.1", "::ffff:127.0.0.1"} {
		if !blockedHFAddress(net.ParseIP(ip)) {
			t.Errorf("%s must be blocked", ip)
		}
	}
	for _, ip := range []string{"8.8.8.8", "13.224.0.1", "2606:4700::1111"} {
		if blockedHFAddress(net.ParseIP(ip)) {
			t.Errorf("%s is public", ip)
		}
	}
}
