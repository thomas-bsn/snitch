package httpapi

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// The active fingerprint group (ADR 0044, ADR 0064): the (model, model_version, dimensions) group the Worker
// produces fingerprints in today, so the one whose References, thresholds and figures the screens show.
//
// The active model (voice_active_model) only names a repository: the version and the dimensions are those of
// its fingerprints. The group is that of the latest production speaker fingerprint of the active model; when
// the active model has none yet, that of the latest production fingerprint of a model outside the catalogue
// (development and synthetic workers, whose results are production too). A blank installation, or one whose
// new model has not produced anything yet, has no active group. Migration 00147 uses the same rule.

// fingerprintGroupView is a fingerprint group as the API shows it.
type fingerprintGroupView struct {
	Model        string `json:"model"`
	ModelVersion string `json:"model_version"`
	Dimensions   int    `json:"dimensions"`
}

func (g matchGroup) view() *fingerprintGroupView {
	return &fingerprintGroupView{Model: g.model, ModelVersion: g.version, Dimensions: g.dimensions}
}

func (v fingerprintGroupView) group() matchGroup {
	return matchGroup{model: v.Model, version: v.ModelVersion, dimensions: v.Dimensions}
}

// activeFingerprintGroup returns the active fingerprint group; false when there is none.
func activeFingerprintGroup(ctx context.Context, q pgQuerier) (matchGroup, bool, error) {
	for _, clause := range []string{
		`f.model=(SELECT m.repo FROM voice_active_model a JOIN voice_model_catalog m ON m.id=a.model_id WHERE a.kind='fingerprint')`,
		`NOT EXISTS (SELECT 1 FROM voice_model_catalog c WHERE c.kind='fingerprint' AND c.repo=f.model)`,
	} {
		var g matchGroup
		err := q.QueryRow(ctx, `SELECT f.model,f.model_version,f.dimensions FROM voice_fingerprints f WHERE NOT f.benchmark AND f.scope='speaker' AND `+clause+` ORDER BY f.id DESC LIMIT 1`).Scan(&g.model, &g.version, &g.dimensions)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return matchGroup{}, false, err
		}
		return g, true, nil
	}
	return matchGroup{}, false, nil
}
