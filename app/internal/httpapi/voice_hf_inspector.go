package httpapi

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// The Hugging Face inspector (ADR 0038, ticket #309) reads the public metadata of one repository pasted by a
// Super-administrator. It downloads no weight and runs no code. The only host it ever contacts is the exact
// Hugging Face API host; every other host, scheme, port or credential is refused before a request is made.

const (
	hfHost   = "huggingface.co"
	hfScheme = "https"
)

var (
	errHFSource = errors.New("invalid Hugging Face source")

	hfSegment  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$`)
	hfRevision = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
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
	switch {
	case len(parts) == 2:
		return repo, "", nil
	case len(parts) >= 4 && (parts[2] == "tree" || parts[2] == "blob" || parts[2] == "resolve") && hfRevision.MatchString(parts[3]):
		return repo, parts[3], nil
	}
	return "", "", errHFSource
}
