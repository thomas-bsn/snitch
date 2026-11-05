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

	mu        sync.Mutex
	cached    []string
	cachedAt  time.Time
	lastTry   time.Time // last registry attempt, successful or not
	lastErr   error     // error of that attempt; nil when it succeeded
	retryFrom time.Time // no new registry call before this instant after a failure
}

// failureBackoff spaces registry calls while it is unreachable: every Worker poll would otherwise retry it.
const failureBackoff = time.Minute

// Status reports the last registry attempt: when it happened (zero if none yet) and whether it failed.
func (c *Catalog) Status() (checkedAt time.Time, failed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastTry, c.lastErr != nil
}

// Newest returns the highest released version, or "" when none is known. The registry order is irrelevant.
func Newest(versions []string) string {
	best := ""
	for _, v := range versions {
		if ValidVersion(v) && (best == "" || Less(best, v)) {
			best = v
		}
	}
	return best
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
	if now().Before(c.retryFrom) {
		if c.cached != nil {
			return append([]string(nil), c.cached...), nil
		}
		return []string{}, fmt.Errorf("%w: %v", ErrCatalog, c.lastErr)
	}
	tags, err := c.fetchTags(ctx)
	c.lastTry, c.lastErr = now(), err
	if err != nil {
		c.retryFrom = now().Add(failureBackoff)
		if c.cached != nil {
			return append([]string(nil), c.cached...), nil
		}
		return []string{}, fmt.Errorf("%w: %v", ErrCatalog, err)
	}
	c.cached, c.cachedAt, c.retryFrom = tags, now(), time.Time{}
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
	if _, err = getJSON(ctx, client, tokenURL.String(), "", &token); err != nil {
		return nil, err
	}
	// The registry paginates (Link header): every main publication adds sha-* and vX.Y.Z tags, so the newest
	// version can sit beyond the first page. Pages are bounded, and a page cap reached is a failure, not a partial list.
	out := []string{}
	next := base.ResolveReference(&url.URL{Path: "/v2/" + c.Repo + "/tags/list", RawQuery: "n=1000"}).String()
	for page := 0; next != ""; page++ {
		if page >= maxPages {
			return nil, errors.New("too many tag pages")
		}
		var list struct {
			Tags []string `json:"tags"`
		}
		link, err := getJSON(ctx, client, next, token.Token, &list)
		if err != nil {
			return nil, err
		}
		for _, t := range list.Tags {
			if ValidVersion(t) {
				out = append(out, t)
			}
		}
		next = ""
		if m := linkNext.FindStringSubmatch(link); m != nil {
			ref, perr := url.Parse(m[1])
			if perr != nil {
				return nil, perr
			}
			// Only follow a link that stays on the registry host: the bearer token must not leave it.
			resolved := base.ResolveReference(ref)
			if resolved.Host != base.Host {
				return nil, errors.New("pagination link leaves the registry")
			}
			next = resolved.String()
		}
	}
	sort.Slice(out, func(i, j int) bool { return Less(out[j], out[i]) })
	return out, nil
}

const maxPages = 20

var linkNext = regexp.MustCompile(`<([^>]+)>\s*;\s*rel="?next"?`)

// getJSON decodes a JSON answer and returns its Link header.
func getJSON(ctx context.Context, client *http.Client, target, bearer string, into any) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("registry answered %d", resp.StatusCode)
	}
	return resp.Header.Get("Link"), json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(into)
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
