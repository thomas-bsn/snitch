package workers

import (
	"regexp"
	"strings"
)

// Model identifies one model a worker uses for a role. It carries no secret:
// the endpoint is host[:port] only and every field is bounded and restricted to
// safe characters.
type Model struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Version  string `json:"version,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

// Models is the declaration a worker sends with its pulse. Every role is optional.
type Models struct {
	Transcription *Model `json:"transcription,omitempty"`
	Diarization   *Model `json:"diarization,omitempty"`
	Fingerprint   *Model `json:"fingerprint,omitempty"`
}

var (
	modelNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:/+-]{0,127}$`)
	versionPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.+-]{0,63}$`)
	endpointPattern  = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?(:[0-9]{1,5})?$`)
	// Credentials must never be stored, even in a field that otherwise looks safe.
	secretPattern = regexp.MustCompile(`(?i)(^|[^a-z0-9])(hf_|sk-|rw_|ghp_|xox[a-z]-|bearer|token|secret|password|passwd|api[_-]?key)|[A-Za-z0-9]{32,}`)
)

func (m *Model) valid() bool {
	if m == nil {
		return true
	}
	for _, value := range []string{m.Provider, m.Model, m.Version, m.Endpoint} {
		if secretPattern.MatchString(value) {
			return false
		}
	}
	if !modelNamePattern.MatchString(m.Provider) || !modelNamePattern.MatchString(m.Model) {
		return false
	}
	if m.Version != "" && !versionPattern.MatchString(m.Version) {
		return false
	}
	return m.Endpoint == "" || (endpointPattern.MatchString(m.Endpoint) && validPort(m.Endpoint))
}

func validPort(endpoint string) bool {
	_, port, found := strings.Cut(endpoint, ":")
	if !found {
		return true
	}
	if len(port) > 1 && port[0] == '0' {
		return false
	}
	value := 0
	for _, digit := range port {
		value = value*10 + int(digit-'0')
	}
	return value >= 1 && value <= 65535
}

// Valid reports whether every declared role passes validation.
func (m Models) Valid() bool {
	return m.Transcription.valid() && m.Diarization.valid() && m.Fingerprint.valid()
}
