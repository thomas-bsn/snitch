package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Thresholds of a fingerprint group (ADR 0064, #393). The scale of a cosine score depends on the embedding
// model: the suggestion threshold, the automatic threshold and the ambiguity margin belong to one (model,
// model_version, dimensions) group (voice_group_thresholds, append-only, the highest version of a group being
// its thresholds). Only an administrator applies them (PUT /api/admin/voice-matching), by hand or from the
// proposal of the measured accuracy; nothing applies them on its own.
//
// A group without applied thresholds only produces suggestions, with the provisional suggestion threshold and
// margin of voice_matching_settings: after a change of fingerprint model, automatic attribution stays
// suspended until thresholds are applied to the new group.

const lockVoiceGroupThresholds = "snitch.voice_group_thresholds"

// Where applied thresholds come from.
const (
	thresholdsFromMigration     = "migration"
	thresholdsFromAdministrator = "administrator"
	thresholdsFromProposal      = "proposal"
)

type groupThresholds struct {
	Version             int       `json:"version"`
	SuggestionThreshold float64   `json:"suggestion_threshold"`
	AutomaticThreshold  float64   `json:"automatic_threshold"`
	AmbiguityMargin     float64   `json:"ambiguity_margin"`
	Source              string    `json:"source"`
	ScoreMode           scoreMode `json:"score_mode"`
	Author              *string   `json:"author"`
	CreatedAt           time.Time `json:"created_at"`
}

const groupThresholdColumns = `version,suggestion_threshold,automatic_threshold,ambiguity_margin,source,score_mode,author_username,created_at`

func scanGroupThresholds(row pgx.Row, extra ...any) (groupThresholds, error) {
	var t groupThresholds
	err := row.Scan(append([]any{&t.Version, &t.SuggestionThreshold, &t.AutomaticThreshold, &t.AmbiguityMargin, &t.Source, &t.ScoreMode, &t.Author, &t.CreatedAt}, extra...)...)
	return t, err
}

// loadGroupThresholds returns the applied thresholds of every group that has some.
func loadGroupThresholds(ctx context.Context, q pgQuerier) (map[matchGroup]groupThresholds, error) {
	rows, err := q.Query(ctx, `SELECT DISTINCT ON (model,model_version,dimensions) `+groupThresholdColumns+`,model,model_version,dimensions
  FROM voice_group_thresholds ORDER BY model,model_version,dimensions,version DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[matchGroup]groupThresholds{}
	for rows.Next() {
		var g matchGroup
		t, scanErr := scanGroupThresholds(rows, &g.model, &g.version, &g.dimensions)
		if scanErr != nil {
			return nil, scanErr
		}
		out[g] = t
	}
	return out, rows.Err()
}

// groupThresholdHistory lists the twenty latest versions of one group, the most recent first.
func groupThresholdHistory(ctx context.Context, q pgQuerier, g matchGroup) ([]groupThresholds, error) {
	rows, err := q.Query(ctx, `SELECT `+groupThresholdColumns+` FROM voice_group_thresholds WHERE model=$1 AND model_version=$2 AND dimensions=$3 ORDER BY version DESC LIMIT 20`, g.model, g.version, g.dimensions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []groupThresholds{}
	for rows.Next() {
		t, scanErr := scanGroupThresholds(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// forGroup is the setting one group is matched with: its applied thresholds, or, without any, the provisional
// suggestion threshold and margin with the automation off for this group, so that it only ever suggests.
func (s matchingSettings) forGroup(t *groupThresholds) matchingSettings {
	if t == nil {
		s.AutoEnabled = false
		s.ThresholdsApplied = false
		return s
	}
	s.SuggestionThreshold, s.AutomaticThreshold, s.AmbiguityMargin = t.SuggestionThreshold, t.AutomaticThreshold, t.AmbiguityMargin
	s.ThresholdsApplied = true
	return s
}

func thresholdsOf(all map[matchGroup]groupThresholds, g matchGroup) *groupThresholds {
	if t, ok := all[g]; ok {
		return &t
	}
	return nil
}

// effectiveMatchingSettings is the setting every user sees: the automation switch, and the thresholds of the
// active fingerprint group (or the provisional ones when it has none applied, or when there is no group yet).
func effectiveMatchingSettings(ctx context.Context, q pgQuerier) (matchingSettings, error) {
	settings, err := loadMatchingSettings(ctx, q)
	if err != nil {
		return settings, err
	}
	g, ok, err := activeFingerprintGroup(ctx, q)
	if err != nil || !ok {
		return settings, err
	}
	var t *groupThresholds
	applied, err := scanGroupThresholds(q.QueryRow(ctx, `SELECT `+groupThresholdColumns+` FROM voice_group_thresholds WHERE model=$1 AND model_version=$2 AND dimensions=$3 ORDER BY version DESC LIMIT 1`, g.model, g.version, g.dimensions))
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return settings, err
	default:
		t = &applied
	}
	auto := settings.AutoEnabled
	settings = settings.forGroup(t)
	settings.AutoEnabled = auto // the switch itself, as the administrator left it
	settings.Group = g.view()
	return settings, nil
}

// applyGroupThresholds records new thresholds for one group, as a new version, unless they are the current
// ones. It reports whether anything changed.
func applyGroupThresholds(ctx context.Context, tx pgx.Tx, g matchGroup, t groupThresholds, author authorRef) (bool, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockVoiceGroupThresholds); err != nil {
		return false, err
	}
	var version int
	var suggestion, automatic, margin float64
	err := tx.QueryRow(ctx, `SELECT version,suggestion_threshold,automatic_threshold,ambiguity_margin FROM voice_group_thresholds WHERE model=$1 AND model_version=$2 AND dimensions=$3 ORDER BY version DESC LIMIT 1`, g.model, g.version, g.dimensions).Scan(&version, &suggestion, &automatic, &margin)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return false, err
	case suggestion == t.SuggestionThreshold && automatic == t.AutomaticThreshold && margin == t.AmbiguityMargin:
		return false, nil
	}
	_, err = tx.Exec(ctx, `INSERT INTO voice_group_thresholds(model,model_version,dimensions,version,suggestion_threshold,automatic_threshold,ambiguity_margin,source,score_mode,author_id,author_username)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,(SELECT mode FROM voice_score_modes ORDER BY version DESC LIMIT 1),$9,$10)`,
		g.model, g.version, g.dimensions, version+1, t.SuggestionThreshold, t.AutomaticThreshold, t.AmbiguityMargin, t.Source, author.ID, author.Username)
	return err == nil, err
}

// productionGroupExists tells whether a group holds production speaker fingerprints: thresholds are only
// applied to a group Snitch actually matches in.
func productionGroupExists(ctx context.Context, q pgQuerier, g matchGroup) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM voice_fingerprints WHERE NOT benchmark AND scope='speaker' AND model=$1 AND model_version=$2 AND dimensions=$3)`, g.model, g.version, g.dimensions).Scan(&ok)
	return ok, err
}
