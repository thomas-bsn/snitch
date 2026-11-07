package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// registerSegmentExportRoutes registers the administrator export of the corrected
// segments (phase 6). The export only reads; Snitch computes no error rate.
func (s *Server) registerSegmentExportRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/segment-corrections/export", s.protected(true, http.HandlerFunc(s.exportSegmentCorrections)))
}

const (
	exportDefaultPageSize = 100
	exportMaxPageSize     = 500
)

// segmentExportItem is one corrected segment of the visible transcription of a clip:
// the original worker text next to the current corrected text.
type segmentExportItem struct {
	RevisionID         string          `json:"revision_id"`
	VisibleIndex       int             `json:"visible_index"`
	Sources            json.RawMessage `json:"sources"`
	ClipID             int64           `json:"clip_id"`
	TranscriptionJobID int64           `json:"transcription_job_id"`
	AudioStreamIndex   int             `json:"audio_stream_index"`
	Index              int             `json:"index"`
	Start              float64         `json:"start"`
	End                float64         `json:"end"`
	OriginalText       string          `json:"original_text"`
	CorrectedText      string          `json:"corrected_text"`
	Author             authorRef       `json:"author"`
	CorrectedAt        time.Time       `json:"corrected_at"`
	Version            int             `json:"version"`
}

// exportSegmentCorrections lists every currently corrected segment (a text correction
// that is not obsolete, on the visible transcription), for every clip: administrators
// see all clips like the rest of the API. Default format is a paginated JSON page;
// format=ndjson streams the whole export as one JSON object per line, as a download.
func (s *Server) exportSegmentCorrections(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	format := query.Get("format")
	if format != "" && format != "json" && format != "ndjson" {
		fail(w, http.StatusBadRequest, "invalid_format")
		return
	}
	page, pageSize := 1, exportDefaultPageSize
	var err error
	if raw := query.Get("page"); raw != "" {
		if page, err = strconv.Atoi(raw); err != nil || page < 1 || page > 100000 {
			fail(w, http.StatusBadRequest, "invalid_pagination")
			return
		}
	}
	if raw := query.Get("page_size"); raw != "" {
		if pageSize, err = strconv.Atoi(raw); err != nil || pageSize < 1 || pageSize > exportMaxPageSize {
			fail(w, http.StatusBadRequest, "invalid_pagination")
			return
		}
	}
	ctx := r.Context()
	const selection = `SELECT revision_id,visible_index,seg->'sources',clip_id,job_id,stream,idx,seg_start,seg_end,original_text,text,author_id,COALESCE(author_username,''),corrected_at,version FROM eff_segments WHERE corrected ORDER BY clip_id,visible_index`
	with := `WITH ` + visibleTranscriptsCTE("") + `, ` + effectiveSegmentsCTE() + ` `
	scan := func(row interface{ Scan(...any) error }) (segmentExportItem, error) {
		var item segmentExportItem
		err := row.Scan(&item.RevisionID, &item.VisibleIndex, &item.Sources, &item.ClipID, &item.TranscriptionJobID, &item.AudioStreamIndex, &item.Index, &item.Start, &item.End, &item.OriginalText, &item.CorrectedText, &item.Author.ID, &item.Author.Username, &item.CorrectedAt, &item.Version)
		return item, err
	}

	if format == "ndjson" {
		rows, err := s.Pool.Query(ctx, with+selection)
		if err != nil {
			s.internal(w, err)
			return
		}
		defer rows.Close()
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition", `attachment; filename="segment-corrections.ndjson"`)
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		for rows.Next() {
			item, err := scan(rows)
			if err != nil {
				return // The status is already sent: the stream just stops.
			}
			if err = encoder.Encode(item); err != nil {
				return
			}
		}
		return
	}

	var total int
	if err = s.Pool.QueryRow(ctx, with+`SELECT count(*) FROM eff_segments WHERE corrected`).Scan(&total); err != nil {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(ctx, with+selection+` LIMIT $1 OFFSET $2`, pageSize, (page-1)*pageSize)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []segmentExportItem{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			s.internal(w, err)
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items, "page": page, "page_size": pageSize, "total": total})
}
