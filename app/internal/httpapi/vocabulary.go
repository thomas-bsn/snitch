package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type vocabularyTerm struct {
	ID                int64    `json:"id"`
	Term              string   `json:"term"`
	Variants          []string `json:"variants"`
	PreferredSpelling string   `json:"preferred_spelling"`
	Active            bool     `json:"active"`
}

func normalizeVocabularyTerm(v *vocabularyTerm) bool {
	v.Term = strings.TrimSpace(v.Term)
	v.PreferredSpelling = strings.TrimSpace(v.PreferredSpelling)
	if v.Term == "" || v.PreferredSpelling == "" || len(v.Term) > 160 || len(v.PreferredSpelling) > 160 || !utf8.ValidString(v.Term) || !utf8.ValidString(v.PreferredSpelling) || len(v.Variants) > 30 {
		return false
	}
	seen := map[string]bool{strings.ToLower(v.Term): true}
	variants := make([]string, 0, len(v.Variants))
	for _, variant := range v.Variants {
		variant = strings.TrimSpace(variant)
		key := strings.ToLower(variant)
		if variant == "" || len(variant) > 160 || !utf8.ValidString(variant) || seen[key] {
			return false
		}
		seen[key] = true
		variants = append(variants, variant)
	}
	v.Variants = variants
	return true
}

func (s *Server) listVocabulary(w http.ResponseWriter, r *http.Request) {
	items, err := s.activeVocabulary(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT id,term,variants,preferred_spelling,active FROM transcription_vocabulary_terms ORDER BY lower(term),id`)
	if err != nil {
		s.internal(w, err)
		return
	}
	all := make([]vocabularyTerm, 0)
	defer rows.Close()
	for rows.Next() {
		var t vocabularyTerm
		if err = rows.Scan(&t.ID, &t.Term, &t.Variants, &t.PreferredSpelling, &t.Active); err != nil {
			s.internal(w, err)
			return
		}
		all = append(all, t)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	version := vocabularyVersion(items)
	reply(w, 200, map[string]any{"items": all, "version": version})
}

func (s *Server) createVocabulary(w http.ResponseWriter, r *http.Request) {
	var t vocabularyTerm
	if !decode(w, r, &t) {
		return
	}
	if !normalizeVocabularyTerm(&t) {
		fail(w, 400, "invalid_vocabulary_term")
		return
	}
	err := s.Pool.QueryRow(r.Context(), `INSERT INTO transcription_vocabulary_terms(term,variants,preferred_spelling,active) VALUES($1,$2,$3,$4) RETURNING id`, t.Term, t.Variants, t.PreferredSpelling, t.Active).Scan(&t.ID)
	if err != nil {
		if uniqueViolation(err) {
			fail(w, 409, "duplicate_vocabulary_term")
			return
		}
		s.internal(w, err)
		return
	}
	reply(w, 201, t)
}

func (s *Server) updateVocabulary(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_vocabulary_id")
		return
	}
	var t vocabularyTerm
	if !decode(w, r, &t) {
		return
	}
	if !normalizeVocabularyTerm(&t) {
		fail(w, 400, "invalid_vocabulary_term")
		return
	}
	t.ID = id
	err = s.Pool.QueryRow(r.Context(), `UPDATE transcription_vocabulary_terms SET term=$2,variants=$3,preferred_spelling=$4,active=$5,updated_at=now() WHERE id=$1 RETURNING id`, id, t.Term, t.Variants, t.PreferredSpelling, t.Active).Scan(&t.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "vocabulary_term_not_found")
		return
	}
	if err != nil {
		if uniqueViolation(err) {
			fail(w, 409, "duplicate_vocabulary_term")
			return
		}
		s.internal(w, err)
		return
	}
	reply(w, 200, t)
}

func (s *Server) deleteVocabulary(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_vocabulary_id")
		return
	}
	tag, err := s.Pool.Exec(r.Context(), `DELETE FROM transcription_vocabulary_terms WHERE id=$1`, id)
	if err != nil {
		s.internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "vocabulary_term_not_found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) activeVocabulary(ctx context.Context) ([]vocabularyTerm, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,term,variants,preferred_spelling,active FROM transcription_vocabulary_terms WHERE active ORDER BY lower(term),id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]vocabularyTerm, 0)
	for rows.Next() {
		var t vocabularyTerm
		if err = rows.Scan(&t.ID, &t.Term, &t.Variants, &t.PreferredSpelling, &t.Active); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func vocabularyVersion(items []vocabularyTerm) string {
	raw, _ := json.Marshal(items)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:8])
}

func uniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
