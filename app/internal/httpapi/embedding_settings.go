package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

func (s *Server) putEmbeddingSettings(w http.ResponseWriter, r *http.Request) {
	var input embeddingSettings
	if !decode(w, r, &input) {
		return
	}
	input.URL, input.Model = strings.TrimSpace(input.URL), strings.TrimSpace(input.Model)
	u, err := url.Parse(input.URL)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" || len(input.URL) > 2048 || input.Model == "" || len(input.Model) > 128 {
		fail(w, http.StatusBadRequest, "invalid_embedding_settings")
		return
	}
	current, err := s.embeddingSettings(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	effective := input
	if current.URLFromEnvironment {
		effective.URL = current.URL
	}
	if current.ModelFromEnvironment {
		effective.Model = current.Model
	}
	body, _ := json.Marshal(embeddingBatch{Model: effective.Model, Input: []string{"health"}})
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, effective.URL, strings.NewReader(string(body)))
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "embedding_provider_unavailable")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "embedding_provider_unavailable")
		return
	}
	defer resp.Body.Close()
	var result embeddingResponse
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&result) != nil || len(result.Data) != 1 || len(result.Data[0].Embedding) == 0 {
		fail(w, http.StatusServiceUnavailable, "embedding_provider_unavailable")
		return
	}
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
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
