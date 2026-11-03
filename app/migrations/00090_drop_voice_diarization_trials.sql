-- +goose Up
-- La fonction d'essai de diarisation (ADR 0022, migrations 00057 et 00058) est retirée de la fiche vidéo.
DROP TABLE voice_diarization_trials;

-- +goose Down
-- Recrée la table vide ; les essais supprimés ne sont pas restaurés.
CREATE TABLE voice_diarization_trials (
    id bigserial PRIMARY KEY,
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    audio_stream_index integer NOT NULL CHECK (audio_stream_index >= 0),
    model_id text NOT NULL CHECK (length(model_id) BETWEEN 1 AND 64),
    model_repo text NOT NULL,
    model_revision text NOT NULL,
    model_engine text NOT NULL DEFAULT 'pyannote',
    num_speakers integer CHECK (num_speakers BETWEEN 1 AND 32),
    min_speakers integer CHECK (min_speakers BETWEEN 1 AND 32),
    max_speakers integer CHECK (max_speakers BETWEEN 1 AND 32),
    run integer NOT NULL DEFAULT 1,
    result jsonb,
    created_by text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz
);
CREATE UNIQUE INDEX voice_diarization_trials_key ON voice_diarization_trials
    (clip_id, audio_stream_index, model_id, COALESCE(num_speakers, 0), COALESCE(min_speakers, 0), COALESCE(max_speakers, 0));
