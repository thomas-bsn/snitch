-- +goose Up
-- Amplitude peaks of one audio track of a clip (ADR 0034), computed by the Worker during
-- the audio preparation of a transcription or a diarization and stored with the job that
-- produced them. Peaks carry no voice information. A new completed job for the track
-- replaces them; deleting the job or the clip removes them.
CREATE TABLE clip_waveforms (
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    audio_stream_index integer NOT NULL CHECK (audio_stream_index >= 0),
    job_id bigint NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    peaks jsonb NOT NULL CHECK (jsonb_typeof(peaks) = 'array' AND jsonb_array_length(peaks) BETWEEN 1 AND 4000),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (clip_id, audio_stream_index)
);
-- +goose Down
DROP TABLE clip_waveforms;
