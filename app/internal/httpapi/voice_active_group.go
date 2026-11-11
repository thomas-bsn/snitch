package httpapi

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// activeFingerprintGroup is the fingerprint group (model, version, dimensions) whose References the
// installation reads as its own: the group of the active fingerprint model (voice_active_model), with the
// version and dimensions of its latest production fingerprints. The version and dimensions are only known from
// the fingerprints themselves, so an installation without any production fingerprint has no group (false).
//
// When the active model has no production fingerprint yet, the group with the most production speaker
// fingerprints stands in: fingerprints of a model outside the catalogue (older installations, synthetic
// fixtures) are still matched, and their References still exist.
func activeFingerprintGroup(ctx context.Context, q pgQuerier) (matchGroup, bool, error) {
	repo, err := voiceActiveFingerprintRepo(ctx, q)
	if err != nil {
		return matchGroup{}, false, err
	}
	g := matchGroup{model: repo}
	if repo != "" {
		err = q.QueryRow(ctx, `SELECT model_version,dimensions FROM voice_fingerprints
  WHERE usable AND NOT benchmark AND scope='speaker' AND model=$1
  GROUP BY model_version,dimensions ORDER BY max(id) DESC LIMIT 1`, repo).Scan(&g.version, &g.dimensions)
		if err == nil {
			return g, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return matchGroup{}, false, err
		}
	}
	err = q.QueryRow(ctx, `SELECT model,model_version,dimensions FROM voice_fingerprints
  WHERE NOT benchmark AND scope='speaker'
  GROUP BY model,model_version,dimensions ORDER BY count(*) DESC,max(id) DESC LIMIT 1`).Scan(&g.model, &g.version, &g.dimensions)
	if errors.Is(err, pgx.ErrNoRows) {
		return matchGroup{}, false, nil
	}
	if err != nil {
		return matchGroup{}, false, err
	}
	return g, true, nil
}
