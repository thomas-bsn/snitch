package jobs

import (
	"regexp"
	"strings"
)

// Masking of the job log lines before they are stored (ADR 0028). The rules mirror the Worker's
// scrub in worker/snitch_worker/log.py, so that both sides keep the same content: tokens, Bearer
// headers, URLs with credentials or signatures, and clip paths never reach the database.
const (
	logMasked     = "[masked]"
	logMaskedPath = "[masked_path]"
)

var (
	logURL       = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://)[^\s"]*`)
	logURLScheme = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://`)
	logSecrets   = []logMask{
		// A Hugging Face token is "hf_" and letters or digits; "hf_not_found" is a stable code.
		{regexp.MustCompile(`\bhf_[A-Za-z0-9]{6,}`), logMasked},
		{regexp.MustCompile(`(?i)\bbearer\s+[^\s"',;]+`), "Bearer " + logMasked},
		{regexp.MustCompile(`\b(?:lt|rw)_[A-Za-z0-9][A-Za-z0-9_-]{7,}`), logMasked},
	}
	logPaths = []logMask{
		{regexp.MustCompile(`(?:[A-Za-z]:)?/(?:tmp|private|var|Users|home|opt|srv|mnt|data|Volumes)/[^\s"]*`), logMaskedPath},
		{regexp.MustCompile(`(?i)[^\s"/]*\.(?:wav|mp3|m4a|mp4|mov|mkv|webm|ogg|flac|aac|media)\b`), logMaskedPath},
	}
)

type logMask struct {
	pattern     *regexp.Regexp
	replacement string
}

// ScrubLogLine masks the sensitive values of one log line. A URL keeps its scheme and host
// only: no userinfo, path or query survives.
func ScrubLogLine(text string) string {
	text = logURL.ReplaceAllStringFunc(text, maskLogURL)
	for _, masks := range [][]logMask{logSecrets, logPaths} {
		for _, mask := range masks {
			text = mask.pattern.ReplaceAllString(text, mask.replacement)
		}
	}
	return text
}

func maskLogURL(match string) string {
	scheme := logURLScheme.FindString(match)
	rest := match[len(scheme):]
	host := rest
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		host = rest[:i]
	}
	if i := strings.LastIndex(host, "@"); i >= 0 {
		host = host[i+1:]
	}
	if strings.ContainsAny(rest, "/?#") {
		return scheme + host + "/" + logMasked
	}
	return scheme + host
}
