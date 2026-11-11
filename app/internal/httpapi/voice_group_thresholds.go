package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Thresholds of a fingerprint group (ADR 0064, #393). The scale of a cosine score depends on the embedding
// model and on the score mode: the suggestion threshold, the automatic threshold and the ambiguity margin
// belong to one (model, model_version, dimensions) group and one score mode (voice_group_thresholds,
// append-only; the thresholds of a group for a mode are its highest version recorded for that mode, versions
// being numbered per group). Only an administrator applies them (PUT /api/admin/voice-matching), by hand or
// from the proposal of the measured accuracy; nothing applies them on its own.
//
// A group without thresholds applied for the active score mode only produces suggestions, with the provisional
// suggestion threshold and margin of voice_matching_settings: after a change of fingerprint model or of score
// mode, automatic attribution stays suspended until thresholds are applied for the new group and mode. Nothing
// is inherited from another group or another mode.

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

// loadGroupThresholds returns the thresholds applied for a score mode to every group that has some.
func loadGroupThresholds(ctx context.Context, q pgQuerier, mode scoreMode) (map[matchGroup]groupThresholds, error) {
	rows, err := q.Query(ctx, `SELECT DISTINCT ON (model,model_version,dimensions) `+groupThresholdColumns+`,model,model_version,dimensions
  FROM voice_group_thresholds WHERE score_mode=$1 ORDER BY model,model_version,dimensions,version DESC`, mode)
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

// appliedThresholds returns the thresholds applied to one group for one score mode, nil when there are none.
func appliedThresholds(ctx context.Context, q pgQuerier, g matchGroup, mode scoreMode) (*groupThresholds, error) {
	t, err := scanGroupThresholds(q.QueryRow(ctx, `SELECT `+groupThresholdColumns+` FROM voice_group_thresholds
  WHERE model=$1 AND model_version=$2 AND dimensions=$3 AND score_mode=$4 ORDER BY version DESC LIMIT 1`, g.model, g.version, g.dimensions, mode))
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	}
	return &t, nil
}

// groupThresholdHistory lists the twenty latest versions of one group, every mode together, the most recent
// first.
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
// active fingerprint group for the active score mode (or the provisional ones when it has none applied for that
// mode, or when there is no group yet).
func effectiveMatchingSettings(ctx context.Context, q pgQuerier) (matchingSettings, error) {
	settings, err := loadMatchingSettings(ctx, q)
	if err != nil {
		return settings, err
	}
	g, ok, err := activeFingerprintGroup(ctx, q)
	if err != nil || !ok {
		return settings, err
	}
	t, err := appliedThresholds(ctx, q, g, settings.ScoreMode)
	if err != nil {
		return settings, err
	}
	auto := settings.AutoEnabled
	settings = settings.forGroup(t)
	settings.AutoEnabled = auto // the switch itself, as the administrator left it
	settings.Group = g.view()
	return settings, nil
}

// applyGroupThresholds records new thresholds for one group and the score mode t.ScoreMode, as a new version of
// the group, unless they are already the thresholds of that mode. It reports whether anything changed.
func applyGroupThresholds(ctx context.Context, tx pgx.Tx, g matchGroup, t groupThresholds, author authorRef) (bool, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockVoiceGroupThresholds); err != nil {
		return false, err
	}
	current, err := appliedThresholds(ctx, tx, g, t.ScoreMode)
	if err != nil {
		return false, err
	}
	if current != nil && current.SuggestionThreshold == t.SuggestionThreshold && current.AutomaticThreshold == t.AutomaticThreshold && current.AmbiguityMargin == t.AmbiguityMargin {
		return false, nil
	}
	_, err = tx.Exec(ctx, `INSERT INTO voice_group_thresholds(model,model_version,dimensions,version,suggestion_threshold,automatic_threshold,ambiguity_margin,source,score_mode,author_id,author_username)
VALUES($1,$2,$3,(SELECT COALESCE(max(version),0)+1 FROM voice_group_thresholds WHERE model=$1 AND model_version=$2 AND dimensions=$3),$4,$5,$6,$7,$8,$9,$10)`,
		g.model, g.version, g.dimensions, t.SuggestionThreshold, t.AutomaticThreshold, t.AmbiguityMargin, t.Source, t.ScoreMode, author.ID, author.Username)
	return err == nil, err
}

// productionGroupExists tells whether a group holds production speaker fingerprints: thresholds are only
// applied to a group Snitch actually matches in.
func productionGroupExists(ctx context.Context, q pgQuerier, g matchGroup) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM voice_fingerprints WHERE NOT benchmark AND scope='speaker' AND model=$1 AND model_version=$2 AND dimensions=$3)`, g.model, g.version, g.dimensions).Scan(&ok)
	return ok, err
}
