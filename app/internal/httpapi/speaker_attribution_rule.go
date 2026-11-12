package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"snitch/internal/segmentparts"
)

// Speaker attribution rule (ADR 0065, #398): how the words of a transcript are given to the Local speakers, by
// word (default) or by whole segment as before. The rule is applied when a transcript is read, so changing it
// rewrites nothing and is reversible at once. Every change is a new version with its author; recording the active
// rule again records nothing.

const lockSpeakerAttributionRule = "snitch.speaker_attribution_rule"

// attributionRule is the name of a rule in the contract and in the database.
type attributionRule string

const (
	attributionBySegment attributionRule = "segment"
	attributionByWord    attributionRule = "word"
)

func (r attributionRule) valid() bool { return r == attributionBySegment || r == attributionByWord }

// version is the version of the attribution function the rule selects.
func (r attributionRule) version() segmentparts.Rule {
	if r == attributionByWord {
		return segmentparts.RuleWord
	}
	return segmentparts.RuleSegment
}

type attributionRuleVersion struct {
	Version   int             `json:"version"`
	Rule      attributionRule `json:"rule"`
	Author    *string         `json:"author"`
	CreatedAt time.Time       `json:"created_at"`
}

type attributionRuleSettings struct {
	attributionRuleVersion
	History []attributionRuleVersion `json:"history"`
}

func (s *Server) registerSpeakerAttributionRuleRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/speaker-attribution-rule", s.protected(true, http.HandlerFunc(s.getSpeakerAttributionRule)))
	mux.Handle("PUT /api/admin/speaker-attribution-rule", s.protected(true, http.HandlerFunc(s.putSpeakerAttributionRule)))
}

// activeAttributionRule reads the active rule (highest version) with its version.
func activeAttributionRule(ctx context.Context, q pgQuerier) (int, attributionRule, error) {
	var version int
	var rule attributionRule
	err := q.QueryRow(ctx, `SELECT version,rule FROM speaker_attribution_rules ORDER BY version DESC LIMIT 1`).Scan(&version, &rule)
	return version, rule, err
}

// loadAttributionRule reads the active rule and the twenty latest versions.
func loadAttributionRule(ctx context.Context, q pgQuerier) (attributionRuleSettings, error) {
	var out attributionRuleSettings
	rows, err := q.Query(ctx, `SELECT version,rule,author_username,created_at FROM speaker_attribution_rules ORDER BY version DESC LIMIT 20`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.History = []attributionRuleVersion{}
	for rows.Next() {
		var v attributionRuleVersion
		if err = rows.Scan(&v.Version, &v.Rule, &v.Author, &v.CreatedAt); err != nil {
			return out, err
		}
		out.History = append(out.History, v)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if len(out.History) == 0 {
		return out, errors.New("no speaker attribution rule")
	}
	out.attributionRuleVersion = out.History[0]
	return out, nil
}

func (s *Server) getSpeakerAttributionRule(w http.ResponseWriter, r *http.Request) {
	out, err := loadAttributionRule(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, out)
}

func (s *Server) putSpeakerAttributionRule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Rule attributionRule `json:"rule"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !input.Rule.valid() {
		fail(w, http.StatusBadRequest, "invalid_speaker_attribution_rule")
		return
	}
	ctx := context.WithoutCancel(r.Context())
	author := s.authorOf(r)
	tx, err := s.beginLocked(ctx, lockSpeakerAttributionRule)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	version, current, err := activeAttributionRule(ctx, tx)
	if err != nil {
		s.internal(w, err)
		return
	}
	if current != input.Rule {
		if _, err = tx.Exec(ctx, `INSERT INTO speaker_attribution_rules(version,rule,author_id,author_username) VALUES($1,$2,$3,$4)`, version+1, input.Rule, author.ID, author.Username); err != nil {
			s.internal(w, err)
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	out, err := loadAttributionRule(ctx, s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, out)
}
