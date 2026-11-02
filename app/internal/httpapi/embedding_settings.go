package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"snitch/internal/auth"
)

type embeddingSettings struct {
	URL                  string `json:"url"`
	Model                string `json:"model"`
	URLFromEnvironment   bool   `json:"url_from_environment"`
	ModelFromEnvironment bool   `json:"model_from_environment"`
}

func (s *Server) embeddingSettings(ctx context.Context) (embeddingSettings, error) {
	out := embeddingSettings{URL: s.Config.EmbeddingsURL, Model: s.Config.EmbeddingsModel, URLFromEnvironment: s.Config.EmbeddingsURL != "", ModelFromEnvironment: s.Config.EmbeddingsModel != ""}
	var err error
	if out.URL == "" {
		out.URL, err = s.Store.Get(ctx, "embeddings_url")
		if err != nil {
			return out, err
		}
	}
	if out.Model == "" {
		out.Model, err = s.Store.Get(ctx, "embeddings_model")
	}
	return out, err
}

func (s *Server) getEmbeddingSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.embeddingSettings(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, settings)
}

// embeddingValidationTTL is how long a successful Test lets the same configuration be saved.
const embeddingValidationTTL = 5 * time.Minute

type embeddingSettingsInput struct {
	URL             string `json:"url"`
	Model           string `json:"model"`
	ValidationToken string `json:"validation_token"`
}

// readEmbeddingInput decodes, normalizes and validates the requested values, and resolves the effective ones:
// values fixed by the environment win and are the ones tested and saved.
func (s *Server) readEmbeddingInput(w http.ResponseWriter, r *http.Request) (input embeddingSettingsInput, current, effective embeddingSettings, ok bool) {
	if !decode(w, r, &input) {
		return
	}
	input.URL, input.Model = strings.TrimSpace(input.URL), strings.TrimSpace(input.Model)
	u, err := url.Parse(input.URL)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" || len(input.URL) > 2048 || input.Model == "" || len(input.Model) > 128 {
		fail(w, http.StatusBadRequest, "invalid_embedding_settings")
		return
	}
	if current, err = s.embeddingSettings(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	effective = embeddingSettings{URL: input.URL, Model: input.Model}
	if current.URLFromEnvironment {
		effective.URL = current.URL
	}
	if current.ModelFromEnvironment {
		effective.Model = current.Model
	}
	return input, current, effective, true
}

// probeEmbeddingProvider calls the real provider with a test input and returns the vector size.
func (s *Server) probeEmbeddingProvider(ctx context.Context, effective embeddingSettings) (int, bool) {
	body, _ := json.Marshal(embeddingBatch{Model: effective.Model, Input: []string{"health"}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, effective.URL, strings.NewReader(string(body)))
	if err != nil {
		return 0, false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	var result embeddingResponse
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&result) != nil || len(result.Data) != 1 || len(result.Data[0].Embedding) == 0 {
		return 0, false
	}
	return len(result.Data[0].Embedding), true
}

// testEmbeddingSettings checks the configuration against the real provider without persisting any setting or
// touching the index; on success it hands back a short-lived single-use proof bound to the tested values and user.
func (s *Server) testEmbeddingSettings(w http.ResponseWriter, r *http.Request) {
	_, _, effective, ok := s.readEmbeddingInput(w, r)
	if !ok {
		return
	}
	dimensions, ok := s.probeEmbeddingProvider(r.Context(), effective)
	if !ok {
		fail(w, http.StatusServiceUnavailable, "embedding_provider_unavailable")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	token := "ev_" + auth.Token()
	expires := time.Now().Add(embeddingValidationTTL)
	if _, err := s.Pool.Exec(r.Context(), `WITH purge AS (DELETE FROM embedding_validations WHERE expires_at<now())
INSERT INTO embedding_validations(token_hash,user_id,url,model,expires_at) VALUES($1,$2,$3,$4,$5)`, auth.Hash(token), ss.User.ID, effective.URL, effective.Model, expires); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"validation_token": token, "dimensions": dimensions, "model": effective.Model, "expires_at": expires})
}

func (s *Server) putEmbeddingSettings(w http.ResponseWriter, r *http.Request) {
	input, current, effective, ok := s.readEmbeddingInput(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	// The server never trusts the button state: Save needs the proof of this exact configuration, consumed here
	// (a Save refused further down rolls back and keeps it).
	ss := r.Context().Value(sessionKey{}).(session)
	var provenURL, provenModel string
	var expires time.Time
	err = tx.QueryRow(ctx, `DELETE FROM embedding_validations WHERE token_hash=$1 AND user_id=$2 RETURNING url,model,expires_at`, auth.Hash(input.ValidationToken), ss.User.ID).Scan(&provenURL, &provenModel, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusConflict, "embedding_validation_required")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if time.Now().After(expires) {
		_ = tx.Commit(ctx) // an expired proof is spent either way
		fail(w, http.StatusConflict, "embedding_validation_expired")
		return
	}
	if provenURL != effective.URL || provenModel != effective.Model {
		fail(w, http.StatusConflict, "embedding_validation_mismatch")
		return
	}
	var active, candidate *int64
	if err = tx.QueryRow(ctx, `SELECT active_profile_id,candidate_profile_id FROM embedding_index_state WHERE id=1 FOR UPDATE`).Scan(&active, &candidate); err != nil {
		s.internal(w, err)
		return
	}
	var activeModel string
	if active != nil {
		if err = tx.QueryRow(ctx, `SELECT model FROM embedding_profiles WHERE id=$1`, *active).Scan(&activeModel); err != nil {
			s.internal(w, err)
			return
		}
	}
	if active == nil || activeModel != effective.Model {
		if candidate != nil {
			var state string
			if err = tx.QueryRow(ctx, `SELECT state FROM embedding_profiles WHERE id=$1`, *candidate).Scan(&state); err != nil {
				s.internal(w, err)
				return
			}
			if state != "failed" {
				fail(w, http.StatusConflict, "embedding_rebuild_in_progress")
				return
			}
			if _, err = tx.Exec(ctx, fmt.Sprintf(`DROP INDEX IF EXISTS transcript_embeddings_p%d_hnsw`, *candidate)); err == nil {
				_, err = tx.Exec(ctx, `UPDATE embedding_index_state SET candidate_profile_id=NULL WHERE id=1`)
			}
			if err == nil {
				_, err = tx.Exec(ctx, `DELETE FROM embedding_profiles WHERE id=$1`, *candidate)
			}
			if err != nil {
				s.internal(w, err)
				return
			}
		}
		var id int64
		state := "building"
		if active == nil {
			state = "active"
		}
		if err = tx.QueryRow(ctx, `INSERT INTO embedding_profiles(model,state) VALUES($1,$2) RETURNING id`, effective.Model, state).Scan(&id); err != nil {
			s.internal(w, err)
			return
		}
		if _, err = tx.Exec(ctx, `WITH latest AS (SELECT DISTINCT ON (clip_id) id FROM jobs WHERE status='done' AND step LIKE 'transcription.audio.%' ORDER BY clip_id,updated_at DESC,id DESC) INSERT INTO embedding_tasks(profile_id,transcription_job_id) SELECT $1,id FROM latest ON CONFLICT DO NOTHING`, id); err != nil {
			s.internal(w, err)
			return
		}
		if active == nil {
			_, err = tx.Exec(ctx, `UPDATE embedding_index_state SET active_profile_id=$1,updated_at=now() WHERE id=1`, id)
		} else {
			_, err = tx.Exec(ctx, `UPDATE embedding_index_state SET candidate_profile_id=$1,updated_at=now() WHERE id=1`, id)
		}
		if err != nil {
			s.internal(w, err)
			return
		}
	}
	store := s.Store.WithTx(tx)
	if !current.URLFromEnvironment {
		err = store.Set(ctx, "embeddings_url", input.URL)
	}
	if err == nil && !current.ModelFromEnvironment {
		err = store.Set(ctx, "embeddings_model", input.Model)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	s.startEmbeddingWorkers()
	s.getEmbeddingSettings(w, r)
}
