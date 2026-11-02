//go:build integration

package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// The browser and application both speak HTTP to this external-provider fixture.
func browserDiscord(t *testing.T, s *Server) {
	t.Helper()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/authorize":
			if r.URL.Query().Get("scope") != "identify" {
				http.Error(w, "unexpected scope", 400)
				return
			}
			callback := r.URL.Query().Get("redirect_uri")
			state := r.URL.Query().Get("state")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, "<h1>Discord simulé</h1>")
			for _, id := range []string{"123456", "654321"} {
				fmt.Fprintf(w, `<a href="%s?state=%s&amp;code=%s">Autoriser %s</a>`, callback, url.QueryEscape(state), id, id)
			}
			fmt.Fprintf(w, `<a href="%s?state=%s&amp;error=access_denied">Refuser</a>`, callback, url.QueryEscape(state))
		case "/token":
			r.ParseForm()
			reply(w, 200, map[string]string{"access_token": "fixture-" + r.Form.Get("code")})
		case "/me":
			// Synthetic display names: id 654321 has a global name, the others only a username.
			id := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer fixture-")
			global := ""
			if id == "654321" {
				global = "Camille Synthetique"
			}
			reply(w, 200, map[string]string{"id": id, "username": "administrator", "global_name": global})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(provider.Close)
	s.Config.DiscordAuthorizeURL = provider.URL + "/authorize"
	s.Config.DiscordTokenURL = provider.URL + "/token"
	s.Config.DiscordUserURL = provider.URL + "/me"
}
