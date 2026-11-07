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
// icon. The name is reduced to the words that identify the game: a build qualifier ("Among Us 3d Demo"), a scene
// release tag ("Bread.and.fred-Goldberg") or an edition suffix ("Dark Souls Remastered") do not prevent the match.
// A short generic name is never guessed into a game ("Content" does not become "Content Warning"), and a name on
// the system-folder list is refused outright. Private folders are not asked for by the page (their name is not
// sent out).

const (
	folderLogoMissTTL  = 7 * 24 * time.Hour
	folderLogoMaxBytes = 2 << 20
)

// folderLogoLookups bounds the calls to Steam: a page of folders asks for all its logos at once.
var folderLogoLookups = make(chan struct{}, 2)

var (
	// A scene release tag at the end of a folder name ("Bread.and.fred-Goldberg").
	folderReleaseTag = regexp.MustCompile(`(?i)[-._]+(goldberg|rune|codex|plaza|skidrow|empress|tenoke|flt|cpy|razor1911|reloaded|prophet|hoodlum|elamigos|fitgirl|dodi|steamrip|gog)\s*$`)
	// Bracketed tags: "[FitGirl Repack]", "(2019)", "{GOG}".
	folderBracket = regexp.MustCompile(`[\[\(\{][^\]\)\}]*[\]\)\}]`)
	// A version at the end ("v1.2.3", "build 12345", "update 2").
	folderVersionTail = regexp.MustCompile(`(?i)[\s._-]+(v\d+([\s._]\d+)*|build[\s._]+\d+|update[\s._]+\d+)\s*$`)
)

// Words that describe a build, not the game. They are dropped from folder names only: a game may legitimately
// carry "Demo" in its store title, and the exact title should still win over a longer prefix.
var folderNoise = map[string]bool{
	"3d": true, "demo": true, "client": true, "beta": true, "playtest": true, "launcher": true,
	"repack": true, "rip": true, "crack": true, "cracked": true, "portable": true, "gog": true,
}

// Release groups that sometimes appear as separate trailing words after punctuation was reduced to spaces.
var folderReleaseGroups = map[string]bool{
	"goldberg": true, "rune": true, "codex": true, "plaza": true, "skidrow": true, "empress": true,
	"tenoke": true, "flt": true, "cpy": true, "razor1911": true, "reloaded": true, "prophet": true,
	"hoodlum": true, "elamigos": true, "fitgirl": true, "dodi": true, "steamrip": true,
}

// Folder names that exist on every machine and must never be shown a game cover, even if a game happens to
// carry the same name. Keys are the reduced names (see folderLogoKey).
var folderLogoExcluded = map[string]bool{
	"common": true, "content": true, "desktop": true, "discord": true, "eadesktop": true, "steam": true,
	"captures": true, "capture": true, "clips": true, "clip": true, "videos": true, "video": true,
	"360videos": true, "music": true, "musique": true, "documents": true, "mesdocuments": true,
	"downloads": true, "telechargements": true, "pictures": true, "images": true, "mesimages": true,
	"screenshots": true, "capturesdecran": true, "recordings": true, "enregistrements": true,
	"enregistrementsdecran": true, "savedgames": true, "jeuxsauvegardes": true, "backups": true,
	"backup": true, "cache": true, "temp": true, "tmp": true, "programfiles": true, "programdata": true,
	"users": true, "public": true, "appdata": true, "baseprofile": true, "bet": true, "games": true,
	"game": true, "jeux": true, "gameoftheyear": true, "goty": true, "edits": true, "edit": true, "editor": true, "exports": true,
	"imports": true, "bluestacks": true, "badlion": true, "csolmanager": true,
}

// Words that only qualify an edition after the game title ("Remastered", "Definitive Edition").
var folderEditionWords = map[string]bool{
	"remastered": true, "remaster": true, "definitive": true, "edition": true, "deluxe": true,
	"ultimate": true, "complete": true, "gold": true, "enhanced": true, "goty": true, "game": true,
	"of": true, "the": true, "year": true, "director": true, "directors": true, "cut": true,
	"collection": true, "legendary": true, "premium": true, "standard": true, "founders": true,
	"vr": true, "hd": true, "rehydrated": true, "anniversary": true, "legacy": true,
	"anthology": true, "trilogy": true, "special": true, "extended": true, "redux": true,
	"remake": true, "reboot": true, "classic": true,
}

// Words that mark a store result as something other than the game itself (demo, soundtrack, DLC pack).
var folderExtraWords = map[string]bool{
	"soundtrack": true, "ost": true, "demo": true, "beta": true, "playtest": true, "pack": true,
	"pass": true, "bundle": true, "dlc": true, "upgrade": true, "supporter": true, "artbook": true,
	"expansion": true, "addon": true, "skin": true, "skins": true, "costume": true, "costumes": true,
	"weapon": true, "weapons": true, "training": true, "mode": true,
}

var romanNumbers = map[string]string{
	"i": "1", "ii": "2", "iii": "3", "iv": "4", "v": "5", "vi": "6", "vii": "7", "viii": "8",
	"ix": "9", "x": "10", "xi": "11", "xii": "12", "xiii": "13", "xiv": "14", "xv": "15",
}

// folderLiteralTokens reduces a folder name to the words that identify the game, keeping roman numerals as they
// are written. "Among Us 3d Demo" becomes ["among", "us"], "Bread.and.fred-Goldberg" becomes
// ["bread", "and", "fred"].
func folderLiteralTokens(name string) []string {
	name = folderBracket.ReplaceAllString(name, " ")
	name = folderReleaseTag.ReplaceAllString(name, "")
	name = folderVersionTail.ReplaceAllString(name, "")
	tokens := wordTokens(name)
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !folderNoise[token] {
			out = append(out, token)
		}
	}
	for len(out) > 0 && folderReleaseGroups[out[len(out)-1]] {
		out = out[:len(out)-1]
	}
	return out
}

// folderNameTokens is the search and matching form: roman numerals become digits ("dark souls iii" becomes
// "dark souls 3").
func folderNameTokens(name string) []string {
	return romanDigits(folderLiteralTokens(name))
}

// gameTitleTokens reduces a store title to what it has in common with a folder name: punctuation, trademarks
// and separators disappear, "&" becomes "and", and a roman numeral becomes a digit.
func gameTitleTokens(title string) []string {
	return romanDigits(gameTitleLiteralTokens(title))
}

func gameTitleLiteralTokens(title string) []string {
	return wordTokens(title)
}

func wordTokens(name string) []string {
	name = strings.NewReplacer("&", " and ", "’", "", "'", "").Replace(name)
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
}

func romanDigits(tokens []string) []string {
	// The first word is left alone: "I Am Fish" and "V Rising" are titles, not numerals.
	for i := 1; i < len(tokens); i++ {
		if digit, ok := romanNumbers[tokens[i]]; ok {
			tokens[i] = digit
		}
	}
	return tokens
}

// folderLogoKey is the primary key of the cache: the reduced folder name without separators.
func folderLogoKey(name string) string {
	return strings.Join(folderNameTokens(name), "")
}

// folderLogoSearchName is the human-readable reduced name sent to the store search.
func folderLogoSearchName(name string) string {
	return strings.Join(folderNameTokens(name), " ")
}

// folderLogoOverride returns the Steam app configured for this folder name, if any.
func (s *Server) folderLogoOverride(name string) (int64, bool) {
	key := folderLogoKey(name)
	for configured, id := range s.Config.FolderLogoOverrides {
		if folderLogoKey(configured) == key {
			return id, true
		}
	}
	return 0, false
}

func (s *Server) folderLogo(w http.ResponseWriter, r *http.Request) {
	// A custom logo chosen for this folder wins over the automatic Steam cover.
	if s.tryCustomFolderLogo(w, r) {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	key := folderLogoKey(name)
	if !s.Config.FolderLogos || len(name) > 255 || len(key) < 3 || len(key) > 255 || folderLogoExcluded[key] {
		fail(w, http.StatusNotFound, "folder_logo_not_found")
		return
	}
	overrideID, _ := s.folderLogoOverride(name)
	var found bool
	var contentType string
	var image []byte
	var checked time.Time
	var cachedAppID int64
	err := s.Pool.QueryRow(r.Context(), `SELECT found,content_type,image,checked_at,app_id FROM folder_logos WHERE name_key=$1`, key).
		Scan(&found, &contentType, &image, &checked, &cachedAppID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, err)
		return
	}
	// A row fetched for a previous (or no) override is not reused: changing the override takes effect at once.
	if err == nil && cachedAppID == overrideID && (found || time.Since(checked) < folderLogoMissTTL) {
		s.writeFolderLogo(w, found, contentType, image)
		return
	}
	select {
	case folderLogoLookups <- struct{}{}:
		defer func() { <-folderLogoLookups }()
	case <-r.Context().Done():
		return
	}
	contentType, image, err = s.steamLogo(r.Context(), overrideID, name)
	if err != nil {
		// Steam unreachable or refusing: nothing is remembered, the next visit asks again.
		s.Log.Debug("folder logo lookup failed", "error", err)
		fail(w, http.StatusNotFound, "folder_logo_not_found")
		return
	}
	found = image != nil
	if _, err = s.Pool.Exec(r.Context(), `INSERT INTO folder_logos(name_key,found,content_type,image,app_id) VALUES($1,$2,$3,$4,$5)
		ON CONFLICT(name_key) DO UPDATE SET found=EXCLUDED.found,content_type=EXCLUDED.content_type,image=EXCLUDED.image,app_id=EXCLUDED.app_id,checked_at=now()`, key, found, contentType, image, overrideID); err != nil {
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

// steamLogo finds the game a folder is named after and downloads its cover. A nil image with no error means no
// game carries that name. An explicit app id (FOLDER_LOGOS_OVERRIDES) skips the search entirely.
func (s *Server) steamLogo(ctx context.Context, appID int64, name string) (string, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if appID <= 0 {
		search := s.Config.SteamStoreURL + "/api/storesearch/?l=english&cc=us&term=" + url.QueryEscape(folderLogoSearchName(name))
		body, _, err := s.getLimited(ctx, search, 1<<20)
		if err != nil {
			return "", nil, err
		}
		var result struct {
			Items []steamSearchItem `json:"items"`
		}
		if err = json.Unmarshal(body, &result); err != nil {
			return "", nil, err
		}
		game := chooseFolderGame(name, result.Items)
		if game == nil {
			return "", nil, nil
		}
		appID = game.ID
	}
	for _, file := range []string{"library_600x900.jpg", "header.jpg"} {
		image, contentType, err := s.getLimited(ctx, s.Config.SteamCDNURL+"/steam/apps/"+strconv.FormatInt(appID, 10)+"/"+file, folderLogoMaxBytes)
		if err == nil && strings.HasPrefix(contentType, "image/jpeg") {
			return "image/jpeg", image, nil
		}
	}
	return "", nil, nil
}

type steamSearchItem struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	ID        int64  `json:"id"`
	Metascore string `json:"metascore"`
}

// chooseFolderGame picks the store result that is the game the folder is named after. A result written with the
// same numeral style comes first ("Modern Warfare 2" keeps the 2009 game rather than the 2022 "II"); then an
// exact title wins, a folder that is a prefix of a longer title (the game, then its subtitle) is accepted for a
// name specific enough not to be a system folder, and a title that is a prefix of the folder is accepted only
// when the extra words are an edition. A result with a metascore is preferred among equally good prefixes, which
// keeps the main game over its spinoffs when the store ranks the spinoff first.
func chooseFolderGame(name string, items []steamSearchItem) *steamSearchItem {
	folder := folderNameTokens(name)
	folderLiteral := folderLiteralTokens(name)
	folderKey := strings.Join(folder, "")
	if folderLogoExcluded[folderKey] {
		return nil
	}
	var best *steamSearchItem
	bestRank, bestLiteral, bestMeta, bestExtra := 0, false, 0, 0
	for i := range items {
		item := &items[i]
		if item.Type != "app" || item.ID <= 0 {
			continue
		}
		title := gameTitleTokens(item.Name)
		titleKey := strings.Join(title, "")
		rank, extra, ok := folderGameMatch(folder, folderKey, title, titleKey)
		if !ok {
			continue
		}
		literal := hasTokenPrefix(gameTitleLiteralTokens(item.Name), folderLiteral)
		meta := 0
		if item.Metascore != "" {
			meta = 1
		}
		if best == nil || (literal && !bestLiteral) ||
			(literal == bestLiteral && rank < bestRank) ||
			(literal == bestLiteral && rank == bestRank && meta > bestMeta) ||
			(literal == bestLiteral && rank == bestRank && meta == bestMeta && extra < bestExtra) {
			best, bestRank, bestLiteral, bestMeta, bestExtra = item, rank, literal, meta, extra
		}
	}
	return best
}

// folderGameMatch returns the rank of a match: 0 for the same title, 1 when the folder starts the title,
// 2 when the folder only adds an edition suffix to the title.
func folderGameMatch(folder []string, folderKey string, title []string, titleKey string) (rank, extra int, ok bool) {
	if folderKey == titleKey {
		return 0, 0, true
	}
	if hasTokenPrefix(title, folder) {
		// A one-word folder ("Content", "Clips") or a very short one ("Bet") is too generic to grow into a
		// longer title; those are exactly the false positives this feature must avoid.
		if len(folder) < 2 || len(folderKey) < 8 || folderLogoExcluded[folderKey] {
			return 0, 0, false
		}
		rest := title[len(folder):]
		if folderHasExtraWords(rest) {
			return 0, 0, false
		}
		return 1, len(rest), true
	}
	if hasTokenPrefix(folder, title) {
		rest := folder[len(title):]
		for _, token := range rest {
			if !folderEditionWords[token] {
				return 0, 0, false
			}
		}
		return 2, len(rest), true
	}
	return 0, 0, false
}

func hasTokenPrefix(tokens, prefix []string) bool {
	if len(prefix) == 0 || len(prefix) > len(tokens) {
		return false
	}
	for i, token := range prefix {
		if tokens[i] != token {
			return false
		}
	}
	return true
}

func folderHasExtraWords(tokens []string) bool {
	for _, token := range tokens {
		if folderExtraWords[token] {
			return true
		}
	}
	return false
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
