package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// clipProgress is the state of the steps only the server knows: the transcription and
// the semantic indexing of the visible transcription. The voice steps are read with the
// clip's voice state. A clip is always imported.
type clipProgress struct {
	Transcription string `json:"transcription"`
	Indexing      string `json:"indexing"`
}

func (s *Server) clipProgress(ctx context.Context, clipID, visibleJobID int64, hasTranscript bool) (clipProgress, error) {
	progress := clipProgress{Transcription: "none", Indexing: "none"}
	if !hasTranscript {
		var status string
		err := s.Pool.QueryRow(ctx, `SELECT status FROM jobs WHERE clip_id=$1 AND step LIKE 'transcription.audio.%' ORDER BY id DESC LIMIT 1`, clipID).Scan(&status)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return progress, err
		}
		switch status {
		case "pending", "retry_wait":
			progress.Transcription = "queued"
		case "leased":
			progress.Transcription = "running"
		case "failed", "dead":
			progress.Transcription = "failed"
		}
		return progress, nil
	}
	progress.Transcription = "done"
	settings, err := s.embeddingSettings(ctx)
	if err != nil {
		return progress, err
	}
	if settings.URL == "" {
		progress.Indexing = "unavailable"
		return progress, nil
	}
	var profileID *int64
	if err := s.Pool.QueryRow(ctx, `SELECT active_profile_id FROM embedding_index_state WHERE id=1`).Scan(&profileID); err != nil {
		return progress, err
	}
	if profileID == nil {
		progress.Indexing = "unavailable"
		return progress, nil
	}
	var state string
	err = s.Pool.QueryRow(ctx, `SELECT CASE WHEN count(*) FILTER(WHERE t.state='failed')>0 THEN 'failed' WHEN count(*) FILTER(WHERE t.state='leased')>0 THEN 'leased' WHEN count(*) FILTER(WHERE t.state IS NULL OR t.state<>'done')>0 THEN 'pending' ELSE 'done' END FROM (SELECT DISTINCT job_id FROM audio_visible_segments WHERE clip_id=$2) v LEFT JOIN embedding_tasks t ON t.transcription_job_id=v.job_id AND t.profile_id=$1`, *profileID, clipID).Scan(&state)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return progress, err
	}
	switch state {
	case "done":
		progress.Indexing = "done"
		var pending bool
		if err = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM segment_corrections WHERE transcription_job_id IN (SELECT job_id FROM audio_visible_segments WHERE clip_id=$1) AND reindex_pending AND stale_at IS NULL)`, clipID).Scan(&pending); err != nil {
			return progress, err
		}
		if pending {
			progress.Indexing = "queued"
		}
	case "leased":
		progress.Indexing = "running"
	case "failed":
		progress.Indexing = "failed"
	default:
		progress.Indexing = "queued"
	}
	return progress, nil
}

type filterOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name,omitempty"`
	User string `json:"username,omitempty"`
}

// listClipFilterOptions gives the Sources and the owners the Library filters offer: only
// those of clips the caller may see, so nobody learns of a Source or an account whose
// clips stay hidden from them.
func (s *Server) listClipFilterOptions(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	admin := isAdministrator(ss.User.Role)
	visible := "clip_readable(c.id,c.owner_id,c.visibility,$1,$2)"
	collect := func(query string, named bool) ([]filterOption, error) {
		rows, err := s.Pool.Query(r.Context(), query, ss.User.ID, admin)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []filterOption{}
		for rows.Next() {
			var item filterOption
			var label string
			if err = rows.Scan(&item.ID, &label); err != nil {
				return nil, err
			}
			if named {
				item.Name = label
			} else {
				item.User = label
			}
			items = append(items, item)
		}
		return items, rows.Err()
	}
	sources, err := collect(`SELECT DISTINCT cs.id,cs.name FROM clip_sources cs JOIN source_files sf ON sf.source_id=cs.id JOIN clips c ON c.id=sf.clip_id WHERE `+visible+` ORDER BY cs.name,cs.id`, true)
	if err != nil {
		s.internal(w, err)
		return
	}
	owners, err := collect(`SELECT DISTINCT u.id,u.username FROM users u JOIN clips c ON c.owner_id=u.id WHERE `+visible+` ORDER BY u.username,u.id`, false)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"sources": sources, "owners": owners})
}
