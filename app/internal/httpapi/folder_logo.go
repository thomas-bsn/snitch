package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
)

// The logo of the game a folder is named after. The folder name is looked up once in the Steam store; the image
// is kept in folder_logos, so browsing never calls out and an instance without Internet simply shows the folder
// icon. Only a name that equals the title of a game is accepted: a folder called "Common" or "Clips" is never
// guessed into a game. Private folders are not asked for by the page (their name is not sent out).

const (
	folderLogoMissTTL  = 7 * 24 * time.Hour
	folderLogoMaxBytes = 2 << 20
)

// folderLogoLookups bounds the calls to Steam: a page of folders asks for all its logos at once.
var folderLogoLookups = make(chan struct{}, 2)

var folderNoise = regexp.MustCompile(`\b(3d|demo|client|beta|playtest|launcher)\b`)

// folderNameKey reduces a folder name or a game title to what they have in common: lower case letters and digits,
// without the words that only qualify a build ("Among Us 3d Demo" is "Among Us").
func folderNameKey(name string) string {
	name = folderNoise.ReplaceAllString(strings.ToLower(name), "")
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Server) folderLogo(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	key := folderNameKey(name)
	if !s.Config.FolderLogos || len(name) > 255 || len(key) < 3 || len(key) > 255 {
		fail(w, http.StatusNotFound, "folder_logo_not_found")
		return
	}
	var found bool
	var contentType string
	var image []byte
	var checked time.Time
	err := s.Pool.QueryRow(r.Context(), `SELECT found,content_type,image,checked_at FROM folder_logos WHERE name_key=$1`, key).Scan(&found, &contentType, &image, &checked)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	if err == nil && (found || time.Since(checked) < folderLogoMissTTL) {
		s.writeFolderLogo(w, found, contentType, image)
		return
	}
	select {
	case folderLogoLookups <- struct{}{}:
		defer func() { <-folderLogoLookups }()
	case <-r.Context().Done():
		return
	}
	contentType, image, err = s.steamLogo(r.Context(), key, name)
	if err != nil {
		// Steam unreachable or refusing: nothing is remembered, the next visit asks again.
		s.Log.Debug("folder logo lookup failed", "error", err)
		fail(w, http.StatusNotFound, "folder_logo_not_found")
		return
	}
	found = image != nil
	if _, err = s.Pool.Exec(r.Context(), `INSERT INTO folder_logos(name_key,found,content_type,image) VALUES($1,$2,$3,$4)
		ON CONFLICT(name_key) DO UPDATE SET found=EXCLUDED.found,content_type=EXCLUDED.content_type,image=EXCLUDED.image,checked_at=now()`, key, found, contentType, image); err != nil {
		s.internal(w, err)
		return
	}
	s.writeFolderLogo(w, found, contentType, image)
}

func (s *Server) writeFolderLogo(w http.ResponseWriter, found bool, contentType string, image []byte) {
	if !found {
		fail(w, http.StatusNotFound, "folder_logo_not_found")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(image)
}

// steamLogo finds the game whose title is the folder name and downloads its cover. A nil image with no error
// means no game carries that name.
func (s *Server) steamLogo(ctx context.Context, key, name string) (string, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	search := s.Config.SteamStoreURL + "/api/storesearch/?l=english&cc=us&term=" + url.QueryEscape(name)
	body, _, err := s.getLimited(ctx, search, 1<<20)
	if err != nil {
		return "", nil, err
	}
	var result struct {
		Items []struct {
			Type string `json:"type"`
			Name string `json:"name"`
			ID   int64  `json:"id"`
		} `json:"items"`
	}
	if err = json.Unmarshal(body, &result); err != nil {
		return "", nil, err
	}
	for _, item := range result.Items {
		if item.Type != "app" || item.ID <= 0 || folderNameKey(item.Name) != key {
			continue
		}
		for _, file := range []string{"library_600x900.jpg", "header.jpg"} {
			image, contentType, err := s.getLimited(ctx, s.Config.SteamCDNURL+"/steam/apps/"+strconv.FormatInt(item.ID, 10)+"/"+file, folderLogoMaxBytes)
			if err == nil && strings.HasPrefix(contentType, "image/jpeg") {
				return "image/jpeg", image, nil
			}
		}
	}
	return "", nil, nil
}

// getLimited reads a 200 answer of at most limit bytes.
func (s *Server) getLimited(ctx context.Context, rawURL string, limit int64) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Snitch folder logos (https://github.com/thomas-bsn/snitch)")
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.New("unexpected status " + strconv.Itoa(resp.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(body)) > limit {
		return nil, "", errors.New("answer too large")
	}
	return body, resp.Header.Get("Content-Type"), nil
}
