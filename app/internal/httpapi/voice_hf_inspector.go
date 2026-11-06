package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// The Hugging Face inspector (ADR 0038, ticket #309) reads the public metadata of one repository pasted by a
// Super-administrator. It downloads no weight and runs no code. The only host it ever contacts is the exact
// Hugging Face API host; every other host, scheme, port or credential is refused before a request is made.

const (
	hfHost      = "huggingface.co"
	hfScheme    = "https"
	hfTimeout   = 8 * time.Second
	hfMaxBytes  = 1 << 20
	hfRedirects = 2
)

var (
	errHFSource      = errors.New("invalid Hugging Face source")
	errHFNotFound    = errors.New("Hugging Face repository not found")
	errHFDenied      = errors.New("Hugging Face access denied")
	errHFUnavailable = errors.New("Hugging Face unavailable")
	errHFTooLarge    = errors.New("Hugging Face response too large")

	hfSegment  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$`)
	hfRevision = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	hfCommit   = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// parseHFSource turns "owner/name" or a URL of the allowed host into a repository and an optional revision
// (a /tree/, /blob/ or /resolve/ URL names one). host and scheme are the only ones accepted.
func parseHFSource(input, host, scheme string) (repo, revision string, err error) {
	s := strings.TrimSpace(input)
	if s == "" || strings.ContainsAny(s, " \t\r\n?#\\@%") {
		return "", "", errHFSource
	}
	path := s
	if strings.Contains(s, "://") || strings.HasPrefix(strings.ToLower(s), host+"/") {
		raw := s
		if !strings.Contains(s, "://") {
			raw = scheme + "://" + s
		}
		u, perr := url.Parse(raw)
		if perr != nil || u.Scheme != scheme || u.Host != host || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return "", "", errHFSource
		}
		path = u.Path
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || !hfSegment.MatchString(parts[0]) || !hfSegment.MatchString(parts[1]) || parts[0] == ".." || parts[1] == ".." {
		return "", "", errHFSource
	}
	repo = parts[0] + "/" + parts[1]
	if len(repo) > 100 {
		return "", "", errHFSource
	}
	switch {
	case len(parts) == 2:
		return repo, "", nil
	case len(parts) >= 4 && (parts[2] == "tree" || parts[2] == "blob" || parts[2] == "resolve") && hfRevision.MatchString(parts[3]):
		return repo, parts[3], nil
	}
	return "", "", errHFSource
}

// hfInspector reads repository metadata. Production uses newHFInspector; only tests build another one
// (a fake server on a loopback address).
type hfInspector struct {
	host, scheme  string
	timeout       time.Duration
	maxBytes      int64
	allowInternal bool // tests only: the fake server listens on loopback
}

func newHFInspector() *hfInspector {
	return &hfInspector{host: hfHost, scheme: hfScheme, timeout: hfTimeout, maxBytes: hfMaxBytes}
}

// hfMetadata is the public part of the API answer that the matrix reads. Nothing else is kept.
type hfMetadata struct {
	ID          string          `json:"id"`
	SHA         string          `json:"sha"`
	Gated       json.RawMessage `json:"gated"`
	Tags        []string        `json:"tags"`
	PipelineTag string          `json:"pipeline_tag"`
	LibraryName string          `json:"library_name"`
	Siblings    []struct {
		Name string `json:"rfilename"`
	} `json:"siblings"`
	CardData struct {
		License json.RawMessage `json:"license"`
	} `json:"cardData"`
	Config struct {
		AutoMap json.RawMessage `json:"auto_map"`
	} `json:"config"`
}

// IsGated is true for "auto" and "manual" gating.
func (m hfMetadata) IsGated() bool {
	g := strings.TrimSpace(string(m.Gated))
	return g != "" && g != "false" && g != "null"
}

// License is the declared license, empty when absent or not a plain string.
func (m hfMetadata) License() string {
	var s string
	if json.Unmarshal(m.CardData.License, &s) != nil {
		return ""
	}
	return s
}

// blockedHFAddress reports an address that must never be dialled: loopback, private, link-local, carrier-grade
// NAT, unspecified or multicast ranges, v4-mapped ones included.
func blockedHFAddress(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
		if v4[0] == 100 && v4[1]&0xC0 == 64 { // 100.64.0.0/10
			return true
		}
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

func (i *hfInspector) client() *http.Client {
	// Control runs on the resolved address, so a name that resolves to an internal address is refused too.
	dialer := &net.Dialer{Timeout: i.timeout, Control: func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		if !i.allowInternal && blockedHFAddress(net.ParseIP(host)) {
			return errors.New("internal address refused")
		}
		return nil
	}}
	return &http.Client{
		Timeout:   i.timeout,
		Transport: &http.Transport{DialContext: dialer.DialContext, Proxy: nil, DisableKeepAlives: true, TLSHandshakeTimeout: i.timeout, ResponseHeaderTimeout: i.timeout},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Host != i.host || req.URL.Scheme != i.scheme || len(via) > hfRedirects {
				return errors.New("redirection refused")
			}
			return nil
		},
	}
}

// fetch reads the metadata of repo at revision (empty: the default branch). No credential is sent.
func (i *hfInspector) fetch(ctx context.Context, repo, revision string) (hfMetadata, error) {
	path := "/api/models/" + repo
	if revision != "" {
		path += "/revision/" + revision
	}
	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, (&url.URL{Scheme: i.scheme, Host: i.host, Path: path}).String(), nil)
	if err != nil {
		return hfMetadata{}, errHFUnavailable
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "snitch-model-inspector")
	resp, err := i.client().Do(req)
	if err != nil {
		return hfMetadata{}, errHFUnavailable
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return hfMetadata{}, errHFNotFound
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return hfMetadata{}, errHFDenied
	case resp.StatusCode != http.StatusOK:
		return hfMetadata{}, errHFUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, i.maxBytes+1))
	if err != nil {
		return hfMetadata{}, errHFUnavailable
	}
	if int64(len(body)) > i.maxBytes {
		return hfMetadata{}, errHFTooLarge
	}
	var meta hfMetadata
	if json.Unmarshal(body, &meta) != nil || !hfCommit.MatchString(meta.SHA) {
		return hfMetadata{}, errHFUnavailable
	}
	return meta, nil
}
