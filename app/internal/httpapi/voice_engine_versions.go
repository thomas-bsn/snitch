package httpapi

import (
	"context"
	"encoding/json"
	"errors"
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

// Recent releases of an on-demand engine (ADR 0056). The screen proposes the latest published versions of a
// family's library so an owner can move to a newer one without typing a number. The only host contacted is the
// Python package index, for a distribution name read from the family table (never from a request). A failure of
// the index never blocks the screen: the list is then empty and the default version stays the proposal.

const (
	pypiHost       = "pypi.org"
	pypiTimeout    = 4 * time.Second
	pypiMaxBytes   = 16 << 20
	pypiFreshFor   = time.Hour
	pypiRetryAfter = 5 * time.Minute
	recentVersions = 10
)

// stableRelease is a final release: digits and dots only (no pre-release, post-release, dev or local part).
var stableRelease = regexp.MustCompile(`^[0-9]+(\.[0-9]+){0,3}$`)

type pypiEntry struct {
	versions []string
	until    time.Time
}

// pypiIndex reads and caches the release list of a distribution. base is empty in production (https://pypi.org);
// a test points it at a fake server.
type pypiIndex struct {
	base    string
	timeout time.Duration
	mu      sync.Mutex
	cache   map[string]pypiEntry
}

func newPyPIIndex() *pypiIndex {
	return &pypiIndex{timeout: pypiTimeout, cache: map[string]pypiEntry{}}
}

var defaultPyPI = newPyPIIndex()

func (s *Server) pypiIndex() *pypiIndex {
	if s.pypi != nil {
		return s.pypi
	}
	return defaultPyPI
}

// recent returns the latest stable versions of a distribution, newest first, or nil when the index is unreachable.
func (p *pypiIndex) recent(ctx context.Context, distribution string) []string {
	p.mu.Lock()
	entry, ok := p.cache[distribution]
	p.mu.Unlock()
	if ok && time.Now().Before(entry.until) {
		return entry.versions
	}
	versions, err := p.fetch(ctx, distribution)
	until := time.Now().Add(pypiFreshFor)
	if err != nil {
		// Keep the last known list and retry soon, so an index outage neither slows every screen nor empties it.
		versions, until = entry.versions, time.Now().Add(pypiRetryAfter)
	}
	p.mu.Lock()
	p.cache[distribution] = pypiEntry{versions: versions, until: until}
	p.mu.Unlock()
	return versions
}

func (p *pypiIndex) fetch(ctx context.Context, distribution string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	base := p.base
	if base == "" {
		base = "https://" + pypiHost
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/pypi/"+url.PathEscape(distribution)+"/json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: p.timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 2 || (p.base == "" && req.URL.Host != pypiHost) {
			return errors.New("redirection refused")
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("package index answered " + strconv.Itoa(resp.StatusCode))
	}
	var body struct {
		Releases map[string][]pypiFile `json:"releases"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, pypiMaxBytes)).Decode(&body); err != nil {
		return nil, err
	}
	return latestStable(body.Releases), nil
}

type pypiFile struct {
	Yanked bool `json:"yanked"`
}

// latestStable keeps the final releases that still have a file which is not withdrawn, newest first.
func latestStable(releases map[string][]pypiFile) []string {
	var versions []string
	for version, files := range releases {
		if !stableRelease.MatchString(version) || !engineVersion.MatchString(version) {
			continue
		}
		for _, file := range files {
			if !file.Yanked {
				versions = append(versions, version)
				break
			}
		}
	}
	sort.Slice(versions, func(i, j int) bool { return compareVersions(versions[i], versions[j]) > 0 })
	if len(versions) > recentVersions {
		versions = versions[:recentVersions]
	}
	return versions
}

// compareVersions orders two dotted numeric versions (1.10 is newer than 1.9).
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			if x > y {
				return 1
			}
			return -1
		}
	}
	return 0
}
