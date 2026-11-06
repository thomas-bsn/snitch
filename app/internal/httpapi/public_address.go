package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
)

// The public address of Snitch (scheme and host as the browser sees them) is deduced from each
// request, so that no setting is required to start. It feeds the origin check, the Discord
// return address, the address written in a worker deployment file and the Secure cookie flag.
//
// Request headers can be forged, so the address is built from the request itself (TLS state and
// Host header) and only the proxy headers X-Forwarded-Proto and X-Forwarded-Host sent by a
// trusted proxy are honored. A peer is trusted when its address (the TCP peer, never a header)
// is inside Config.TrustedProxies.// Behind a trusted HTTP proxy chain, same-origin browser metadata can recover HTTPS
// lost between proxies. It can only upgrade the scheme of the already established host.

var hostPattern = regexp.MustCompile(`^(?:[a-z0-9]|[a-z0-9][a-z0-9._-]*[a-z0-9])(?::[0-9]{1,5})?$|^\[[0-9a-f:.]+\](?::[0-9]{1,5})?$`)

// publicOrigin returns the origin (scheme://host[:port], no trailing slash) the client used.
// It is empty when the request carries no usable host.
func (s *Server) publicOrigin(r *http.Request) string {
	scheme, host := "http", strings.ToLower(r.Host)
	if r.TLS != nil {
		scheme = "https"
	}
	if s.trustedPeer(r) {
		if proto := firstForwardedValue(r, "X-Forwarded-Proto"); proto == "http" || proto == "https" {
			scheme = proto
		}
		if forwarded := firstForwardedValue(r, "X-Forwarded-Host"); hostPattern.MatchString(forwarded) {
			host = forwarded
		}
		if scheme == "http" && browserUsedHTTPS(r, host) {
			scheme = "https"
		}
	}
	if !hostPattern.MatchString(host) {
		return ""
	}
	// Browsers write origins without the default port.
	if name, port, err := net.SplitHostPort(host); err == nil && ((scheme == "https" && port == "443") || (scheme == "http" && port == "80")) {
		host = name
		if strings.Contains(name, ":") {
			host = "[" + name + "]"
		}
	}
	return scheme + "://" + host
}

// browserUsedHTTPS handles TLS termination followed by a proxy that replaces
// X-Forwarded-Proto with http. Fetch Metadata is browser-controlled; Origin (or
// Referer for GET requests) must agree with the host independently seen by Snitch.
// Never learn a host from these headers, accept a cross-site request, or downgrade TLS.
func browserUsedHTTPS(r *http.Request, host string) bool {
	if r.Header.Get("Sec-Fetch-Site") != "same-origin" {
		return false
	}
	value := r.Header.Get("Origin")
	isOrigin := value != ""
	if !isOrigin {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			return false
		}
		value = r.Referer()
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return false
	}
	if isOrigin && (u.Path != "" || u.RawQuery != "" || u.ForceQuery) {
		return false
	}
	// Compare HTTPS authorities without the default port, just as publicOrigin does.
	return strings.TrimSuffix(strings.ToLower(u.Host), ":443") == strings.TrimSuffix(host, ":443")
}

// secureCookies reports whether the session cookie carries the Secure attribute: forced by
// COOKIE_SECURE when provided, otherwise whenever the public address is HTTPS.
func (s *Server) secureCookies(r *http.Request) bool {
	if s.Config.Secure {
		return true
	}
	if s.Config.SecureExplicit {
		return false
	}
	return strings.HasPrefix(s.publicOrigin(r), "https://")
}

func (s *Server) trustedPeer(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	addr = addr.Unmap().WithZone("")
	for _, prefix := range s.Config.TrustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// firstForwardedValue returns the client-facing value of a proxy header (the first of a
// comma-separated list), lowercase. The trusted proxy must overwrite the header it forwards.
func firstForwardedValue(r *http.Request, name string) string {
	values := r.Header.Values(name)
	if len(values) == 0 {
		return ""
	}
	first, _, _ := strings.Cut(values[0], ",")
	return strings.ToLower(strings.TrimSpace(first))
}
