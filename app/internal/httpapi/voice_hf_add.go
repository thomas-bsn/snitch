package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// Assisted addition of a Hugging Face repository to the catalogue (ADR 0038, #309). Two Super-administrator
// calls: the inspection reads public metadata and proposes; the addition re-reads the very commit that was
// shown, so what is pinned is what the person saw. The engine is an output of the adapter matrix, never an
// input.

func (s *Server) registerVoiceModelAddRoutes(mux *http.ServeMux) {
	mux.Handle("POST /api/admin/voice-models/inspection", s.protectedFor(isSuperAdministrator, http.HandlerFunc(s.inspectVoiceModel)))
	mux.Handle("POST /api/admin/voice-models", s.protectedFor(isSuperAdministrator, http.HandlerFunc(s.addVoiceModel)))
}

func (s *Server) hfInspector() *hfInspector {
	if s.hf != nil {
		return s.hf
	}
	return newHFInspector()
}

type voiceModelInspection struct {
	Repo              string   `json:"repo"`
	Revision          string   `json:"revision"`
	License           string   `json:"license"`
	Gated             bool     `json:"gated"`
	Engine            string   `json:"engine"`
	ProposedKind      *string  `json:"proposed_kind"`
	CompatibleKinds   []string `json:"compatible_kinds"`
	Supported         bool     `json:"supported"`
	UnsupportedReason string   `json:"unsupported_reason"`
	ExistingIDs       []string `json:"existing_ids"`
}

// hfFailure maps a failure of the inspector to the answer; false when err is not one of them.
func hfFailure(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, errHFSource):
		fail(w, http.StatusBadRequest, "invalid_hf_source")
	case errors.Is(err, errHFNotFound):
		fail(w, http.StatusNotFound, "voice_model_hf_not_found")
	case errors.Is(err, errHFDenied):
		fail(w, http.StatusUnprocessableEntity, "voice_model_hf_denied")
	case errors.Is(err, errHFUnavailable), errors.Is(err, errHFTooLarge):
		fail(w, http.StatusBadGateway, "voice_model_hf_unavailable")
	default:
		return false
	}
	return true
}

func (s *Server) inspectVoiceModel(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Source string `json:"source"`
	}
	if !decode(w, r, &input) {
		return
	}
	in := s.hfInspector()
	repo, revision, err := parseHFSource(input.Source, in.host, in.scheme)
	if err != nil {
		hfFailure(w, err)
		return
	}
	meta, err := in.fetch(r.Context(), repo, revision)
	if err != nil {
		if !hfFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	a := assessHF(meta)
	view := voiceModelInspection{
		Repo: repo, Revision: meta.SHA, License: clip(meta.License(), 100), Gated: meta.IsGated(), Engine: a.Engine,
		CompatibleKinds: append([]string{}, a.CompatibleKinds...), Supported: a.UnsupportedReason == "", UnsupportedReason: a.UnsupportedReason, ExistingIDs: []string{},
	}
	if a.ProposedKind != "" {
		view.ProposedKind = &a.ProposedKind
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT id FROM voice_model_catalog WHERE repo=$1 ORDER BY created_at, id`, repo)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			s.internal(w, err)
			return
		}
		view.ExistingIDs = append(view.ExistingIDs, id)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

var speakerLimit = regexp.MustCompile(`(?i)(\d{1,2})spk`)

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func (s *Server) addVoiceModel(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Source   string `json:"source"`
		Revision string `json:"revision"`
		Kind     string `json:"kind"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !slices.Contains([]string{"diarization", "fingerprint", "separation"}, input.Kind) || !hfCommit.MatchString(input.Revision) {
		fail(w, http.StatusBadRequest, "invalid_voice_model")
		return
	}
	in := s.hfInspector()
	repo, _, err := parseHFSource(input.Source, in.host, in.scheme)
	if err != nil {
		hfFailure(w, err)
		return
	}
	meta, err := in.fetch(r.Context(), repo, input.Revision)
	if err == nil && meta.SHA != input.Revision {
		err = errHFUnavailable // the answer is not the commit that was asked for
	}
	if err != nil {
		if !hfFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	a := assessHF(meta)
	if a.ProposedKind != "" && a.ProposedKind != input.Kind {
		fail(w, http.StatusConflict, "voice_model_kind_mismatch")
		return
	}
	supported, reason := a.supports(input.Kind)
	entry := voiceCatalogEntry{
		Kind: input.Kind, Repo: repo, Revision: meta.SHA, Label: clip(repo, 100), License: clip(meta.License(), 100), Gated: meta.IsGated(),
		Enabled: true, Supported: supported, UnsupportedReason: reason,
	}
	if supported {
		entry.Engine = a.Engine
		entry.Family = catalogFamily(a.Engine, input.Kind)
		entry.SupportsNumSpeakers = a.Engine == "pyannote" && input.Kind == "diarization"
		if m := speakerLimit.FindStringSubmatch(repo); m != nil && a.Engine == "nemo" {
			entry.MaxSpeakers, _ = strconv.Atoi(m[1])
		}
		if input.Kind == "separation" {
			// The assessment only supports a documented reference: its limits go to the catalogue with it.
			limits := separationModels[meta.ID]
			entry.SeparatedVoices, entry.SampleRateHz = limits.Voices, limits.SampleRateHz
		}
	}
	slug := strings.Trim(regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(strings.ToLower(repo), "-"), "-.")
	sum := sha256.Sum256([]byte(input.Kind + "/" + repo))
	ids := []string{clip(slug, 64), clip(slug, 55) + "-" + input.Kind[:4], clip(slug, 50) + "-" + hex.EncodeToString(sum[:4])}
	for _, id := range ids {
		entry.ID = id
		tag, err := s.Pool.Exec(r.Context(), `INSERT INTO voice_model_catalog (id,kind,engine,repo,revision,label,license,gated,max_speakers,size_bytes,vram_bytes,supports_num_speakers,enabled,supported,unsupported_reason,family,separated_voices,sample_rate_hz)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,0,0,$10,true,$11,$12,$13,$14,$15) ON CONFLICT (id) DO NOTHING`,
			entry.ID, entry.Kind, entry.Engine, entry.Repo, entry.Revision, entry.Label, entry.License, entry.Gated, entry.MaxSpeakers, entry.SupportsNumSpeakers, entry.Supported, entry.UnsupportedReason, entry.Family, entry.SeparatedVoices, entry.SampleRateHz)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // the same repository already holds this type
			fail(w, http.StatusConflict, "voice_model_exists")
			return
		}
		if err != nil {
			s.internal(w, err)
			return
		}
		if tag.RowsAffected() == 1 {
			reply(w, http.StatusCreated, voiceModelViewOf(entry))
			return
		}
	}
	fail(w, http.StatusConflict, "voice_model_exists")
}
