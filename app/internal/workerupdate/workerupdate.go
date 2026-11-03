// Package workerupdate holds the pure rules of automatic Worker updates (ADR 0036): version
// validation, the image Snitch hands out, and the catalogue of published versions.
package workerupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// versionPattern accepts only released versions: vX.Y.Z. Anything else (latest, dates, branch names) is refused.
var versionPattern = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

// ValidVersion reports whether s is a released version tag.
func ValidVersion(s string) bool { return versionPattern.MatchString(s) }

// Less orders two valid versions by semantic precedence.
func Less(a, b string) bool {
	x, y := parts(a), parts(b)
	for i := range x {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return false
}

func parts(v string) [3]int {
	var out [3]int
	for i, p := range strings.Split(strings.TrimPrefix(v, "v"), ".") {
		out[i], _ = strconv.Atoi(p)
	}
	return out
}

// ImageFor returns the image reference for a version, built from the repository of the
// configured image: a reference carrying its own tag keeps only its repository. A digest
// reference is refused: the target is always a version tag.
func ImageFor(configured, version string) (string, error) {
	if !ValidVersion(version) {
		return "", ErrInvalidVersion
	}
	if strings.Contains(configured, "@") {
		return "", ErrInvalidImage
	}
	repo := configured
	if i := strings.LastIndex(configured, ":"); i > strings.LastIndex(configured, "/") {
		repo = configured[:i]
	}
	if repo == "" {
		return "", ErrInvalidImage
	}
	return repo + ":" + version, nil
}

var (
	ErrInvalidVersion = errors.New("invalid_worker_version")
	ErrInvalidImage   = errors.New("invalid_worker_image")
	ErrCatalog        = errors.New("worker_catalog_unavailable")
)

// Catalog lists the released versions of the Worker image in the registry. It caches the
// answer for a short time so that the administration screen does not hit the registry on every load.
type Catalog struct {
	Client   *http.Client
	Registry string // e.g. https://ghcr.io
	Repo     string // e.g. thomas-bsn/snitch-worker
	TTL      time.Duration
	Now      func() time.Time

	mu       sync.Mutex
	cached   []string
	cachedAt time.Time
}

// Versions returns released versions, newest first. On failure it returns ErrCatalog and keeps
// the last good answer if there is one.
func (c *Catalog) Versions(ctx context.Context) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	ttl := c.TTL
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	if c.cached != nil && now().Sub(c.cachedAt) < ttl {
		return append([]string(nil), c.cached...), nil
	}
	tags, err := c.fetchTags(ctx)
	if err != nil {
		if c.cached != nil {
			return append([]string(nil), c.cached...), nil
		}
		return []string{}, fmt.Errorf("%w: %v", ErrCatalog, err)
	}
	c.cached, c.cachedAt = tags, now()
	return append([]string(nil), tags...), nil
}

func (c *Catalog) fetchTags(ctx context.Context) ([]string, error) {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	base, err := url.Parse(c.Registry)
	if err != nil || base.Scheme != "https" {
		return nil, errors.New("registry must be HTTPS")
	}
	// Anonymous pull token: public packages only.
	tokenURL := base.ResolveReference(&url.URL{Path: "/token", RawQuery: url.Values{"scope": {"repository:" + c.Repo + ":pull"}}.Encode()})
	var token struct {
		Token string `json:"token"`
	}
	if err = getJSON(ctx, client, tokenURL.String(), "", &token); err != nil {
		return nil, err
	}
	var list struct {
		Tags []string `json:"tags"`
	}
	tagsURL := base.ResolveReference(&url.URL{Path: "/v2/" + c.Repo + "/tags/list"})
	if err = getJSON(ctx, client, tagsURL.String(), token.Token, &list); err != nil {
		return nil, err
	}
	out := []string{}
	for _, t := range list.Tags {
		if ValidVersion(t) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return Less(out[j], out[i]) })
	return out, nil
}

func getJSON(ctx context.Context, client *http.Client, target, bearer string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("registry answered %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(into)
}

// RegistryOf splits the configured image into the HTTPS registry origin and the repository path,
// e.g. ghcr.io/thomas-bsn/snitch-worker:latest -> https://ghcr.io, thomas-bsn/snitch-worker.
func RegistryOf(configured string) (registry, repo string, err error) {
	ref, err := ImageFor(configured, "v0.0.0")
	if err != nil {
		return "", "", err
	}
	ref = strings.TrimSuffix(ref, ":v0.0.0")
	host, path, ok := strings.Cut(ref, "/")
	if !ok || host == "" || path == "" {
		return "", "", ErrInvalidImage
	}
	return "https://" + host, path, nil
}
