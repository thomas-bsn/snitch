package httpapi

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// voiceCatalogEntry is one model reference of the catalogue (ADR 0038). The catalogue lives in the table
// voice_model_catalog: no search, no free repository, because an unknown repository is not executable and
// loading it would run arbitrary code (pickle). Each entry pins a revision and names the approved adapter
// (Engine) allowed to run it.
type voiceCatalogEntry struct {
	ID                  string
	Kind                string // diarization, fingerprint or separation
	Engine              string
	Repo                string
	Revision            string
	Label               string
	License             string
	Gated               bool
	MaxSpeakers         int // 0: not limited by the model
	SizeBytes           int64
	VRAMBytes           int64 // 0: not known
	SupportsNumSpeakers bool
	Enabled             bool // a disabled reference can no longer be chosen; its history stays readable
	Supported           bool // false: no approved adapter can run it
	UnsupportedReason   string
}

const voiceCatalogColumns = `id,kind,engine,repo,revision,label,license,gated,max_speakers,size_bytes,vram_bytes,supports_num_speakers,enabled,supported,unsupported_reason`

func scanVoiceCatalogEntry(row pgx.Row) (voiceCatalogEntry, error) {
	var e voiceCatalogEntry
	err := row.Scan(&e.ID, &e.Kind, &e.Engine, &e.Repo, &e.Revision, &e.Label, &e.License, &e.Gated, &e.MaxSpeakers, &e.SizeBytes, &e.VRAMBytes, &e.SupportsNumSpeakers, &e.Enabled, &e.Supported, &e.UnsupportedReason)
	return e, err
}

// voiceCatalogList returns every reference, in a stable order (kind, then creation).
func voiceCatalogList(ctx context.Context, db pgQuerier) ([]voiceCatalogEntry, error) {
	rows, err := db.Query(ctx, `SELECT `+voiceCatalogColumns+` FROM voice_model_catalog ORDER BY CASE kind WHEN 'diarization' THEN 0 WHEN 'fingerprint' THEN 1 ELSE 2 END, created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []voiceCatalogEntry
	for rows.Next() {
		e, err := scanVoiceCatalogEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// voiceRealProvider tells whether a pipeline provider runs a catalogue model (the fake one runs none).
// The provider of a diarization model is its engine.
func voiceRealProvider(provider string) bool {
	return provider == "pyannote" || provider == "nemo"
}

// voiceCatalogByID reads one reference; ok is false when the id is not in the catalogue.
func voiceCatalogByID(ctx context.Context, db pgQuerier, id string) (voiceCatalogEntry, bool, error) {
	e, err := scanVoiceCatalogEntry(db.QueryRow(ctx, `SELECT `+voiceCatalogColumns+` FROM voice_model_catalog WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return voiceCatalogEntry{}, false, nil
	}
	return e, err == nil, err
}

func voiceCatalogByRepo(ctx context.Context, db pgQuerier, kind, repo string) (voiceCatalogEntry, bool, error) {
	e, err := scanVoiceCatalogEntry(db.QueryRow(ctx, `SELECT `+voiceCatalogColumns+` FROM voice_model_catalog WHERE kind=$1 AND repo=$2`, kind, repo))
	if errors.Is(err, pgx.ErrNoRows) {
		return voiceCatalogEntry{}, false, nil
	}
	return e, err == nil, err
}

// voiceActiveFingerprintRepo is the repository of the active speaker embedding model, empty when none is set.
func voiceActiveFingerprintRepo(ctx context.Context, db pgQuerier) (string, error) {
	var repo string
	err := db.QueryRow(ctx, `SELECT m.repo FROM voice_active_model a JOIN voice_model_catalog m ON m.id=a.model_id WHERE a.kind='fingerprint'`).Scan(&repo)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return repo, err
}

// choosable tells whether a reference may be chosen as a diarization model: enabled, supported, right type.
func (e voiceCatalogEntry) choosable() bool {
	return e.Kind == "diarization" && e.Enabled && e.Supported
}

// voiceFailureCauses is the closed vocabulary a Worker may report for a failed preparation.
var voiceFailureCauses = map[string]bool{
	"hf_token_missing": true, "hf_access_denied": true, "disk_full": true, "runtime_incompatible": true, "inference_failed": true,
	// Hugging Face failure families of the failure catalogue (ADR 0023); hf_access_denied is kept for older Workers.
	"hf_auth_required": true, "hf_not_found": true, "hf_network": true, "models_volume_unwritable": true, "hf_cache_error": true,
}
