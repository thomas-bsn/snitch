-- +goose Up
CREATE TABLE audio_plans (
 id bigserial PRIMARY KEY,
 run_id bigint NOT NULL REFERENCES clip_analysis_runs(id) ON DELETE CASCADE,
 clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
 input_version text NOT NULL CHECK (input_version ~ '^[a-f0-9]{64}$'),
 input jsonb NOT NULL CHECK (jsonb_typeof(input)='object'),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(run_id,input_version), UNIQUE(id,clip_id)
);
CREATE INDEX audio_plans_clip ON audio_plans(clip_id,id);
CREATE TABLE audio_plan_tracks (
 plan_id bigint NOT NULL REFERENCES audio_plans(id) ON DELETE CASCADE,
 track_index integer NOT NULL CHECK(track_index>=0),
 PRIMARY KEY(plan_id,track_index)
);
CREATE TABLE audio_plan_children (
 plan_id bigint NOT NULL,
 track_index integer NOT NULL,
 stage text NOT NULL CHECK(stage IN ('diagnostic','transcription','diarization','fingerprint')),
 job_id bigint NOT NULL UNIQUE REFERENCES jobs(id) ON DELETE CASCADE,
 input_version text NOT NULL,
 PRIMARY KEY(plan_id,track_index,stage),
 FOREIGN KEY(plan_id,track_index) REFERENCES audio_plan_tracks(plan_id,track_index) ON DELETE CASCADE
);
CREATE TABLE audio_track_diagnostics (
 plan_id bigint NOT NULL,
 track_index integer NOT NULL,
 attempt integer NOT NULL CHECK(attempt>=0),
 payload jsonb NOT NULL,
 PRIMARY KEY(plan_id,track_index,attempt),
 FOREIGN KEY(plan_id,track_index) REFERENCES audio_plan_tracks(plan_id,track_index) ON DELETE CASCADE
);
CREATE TABLE transcript_revisions (
 id text PRIMARY KEY,
 clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
 plan_id bigint REFERENCES audio_plans(id) ON DELETE CASCADE,
 legacy_job_id bigint UNIQUE REFERENCES jobs(id) ON DELETE CASCADE,
 manifest jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK ((plan_id IS NULL) <> (legacy_job_id IS NULL)),
 FOREIGN KEY(plan_id,clip_id) REFERENCES audio_plans(id,clip_id) ON DELETE CASCADE
);
CREATE INDEX transcript_revisions_clip ON transcript_revisions(clip_id);
-- Immutable identities may be removed by analysis/clip deletion, never overwritten.
-- +goose StatementBegin
CREATE FUNCTION audio_identity_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'immutable audio identity' USING ERRCODE='23514'; END $$;
-- +goose StatementEnd
CREATE TRIGGER audio_plans_immutable BEFORE UPDATE ON audio_plans FOR EACH ROW EXECUTE FUNCTION audio_identity_immutable();
CREATE TRIGGER audio_tracks_immutable BEFORE UPDATE ON audio_plan_tracks FOR EACH ROW EXECUTE FUNCTION audio_identity_immutable();
CREATE TRIGGER audio_children_immutable BEFORE UPDATE ON audio_plan_children FOR EACH ROW EXECUTE FUNCTION audio_identity_immutable();
CREATE TRIGGER audio_diagnostics_immutable BEFORE UPDATE ON audio_track_diagnostics FOR EACH ROW EXECUTE FUNCTION audio_identity_immutable();
CREATE TRIGGER transcript_revisions_immutable BEFORE UPDATE ON transcript_revisions FOR EACH ROW EXECUTE FUNCTION audio_identity_immutable();
-- Projection preserves array positions and the old job identity, including words.
-- Legacy unknown stream indices remain -1 rather than inventing a real ffprobe index.
-- +goose StatementBegin
CREATE FUNCTION legacy_transcript_manifest(j jobs) RETURNS jsonb LANGUAGE sql STABLE AS $$
 SELECT jsonb_build_object('id','legacy:'||j.id,'clip_id',j.clip_id,'plan_id',NULL,'legacy_job_id',j.id,
 'empty',jsonb_array_length(COALESCE(j.result->'segments','[]'::jsonb))=0,'time_base','clip','segments',
 COALESCE((SELECT jsonb_agg(jsonb_build_object('index',s.ord-1,'start',COALESCE(s.seg->'start','0'::jsonb),
 'end',COALESCE(s.seg->'end',s.seg->'start','0'::jsonb),'sources',jsonb_build_array(jsonb_build_object('track_index',COALESCE((j.result->>'audio_stream_index')::int,-1),'job_id',j.id,'segment_index',s.ord-1)),
 'words',COALESCE((SELECT jsonb_agg(jsonb_build_object('index',w.ord-1,'start',COALESCE(w.word->'start',s.seg->'start','0'::jsonb),'end',COALESCE(w.word->'end',w.word->'start',s.seg->'end','0'::jsonb),
 'sources',jsonb_build_array(jsonb_build_object('track_index',COALESCE((j.result->>'audio_stream_index')::int,-1),'job_id',j.id,'segment_index',s.ord-1,'word_index',w.ord-1))) ORDER BY w.ord)
 FROM jsonb_array_elements(COALESCE(s.seg->'words','[]'::jsonb)) WITH ORDINALITY w(word,ord)),'[]'::jsonb)) ORDER BY s.ord)
 FROM jsonb_array_elements(COALESCE(j.result->'segments','[]'::jsonb)) WITH ORDINALITY s(seg,ord)),'[]'::jsonb))
$$;
-- +goose StatementEnd
INSERT INTO transcript_revisions(id,clip_id,legacy_job_id,manifest)
SELECT 'legacy:'||j.id,j.clip_id,j.id,legacy_transcript_manifest(j) FROM jobs j
WHERE j.status='done' AND j.step LIKE 'transcription.audio.%';
-- +goose StatementBegin
CREATE FUNCTION capture_legacy_transcript() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.status='done' AND NEW.step LIKE 'transcription.audio.%' AND NOT EXISTS(SELECT 1 FROM audio_plan_children WHERE job_id=NEW.id) THEN
 INSERT INTO transcript_revisions(id,clip_id,legacy_job_id,manifest) VALUES('legacy:'||NEW.id,NEW.clip_id,NEW.id,legacy_transcript_manifest(NEW)) ON CONFLICT DO NOTHING;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER capture_legacy_transcript AFTER INSERT OR UPDATE OF status,result ON jobs FOR EACH ROW EXECUTE FUNCTION capture_legacy_transcript();
CREATE VIEW audio_visible_legacy_transcripts AS
 SELECT DISTINCT ON(j.clip_id) j.clip_id,j.id,j.result || jsonb_build_object('revision',r.manifest) AS result,j.updated_at
 FROM jobs j JOIN transcript_revisions r ON r.legacy_job_id=j.id
 WHERE j.status='done' AND j.step LIKE 'transcription.audio.%'
 AND NOT EXISTS(SELECT 1 FROM audio_plan_children c WHERE c.job_id=j.id)
 ORDER BY j.clip_id,j.updated_at DESC,j.id DESC;

-- +goose Down
DROP VIEW audio_visible_legacy_transcripts;
DROP TRIGGER capture_legacy_transcript ON jobs;
DROP FUNCTION capture_legacy_transcript();
DROP FUNCTION legacy_transcript_manifest(jobs);
DROP TABLE transcript_revisions,audio_track_diagnostics,audio_plan_children,audio_plan_tracks,audio_plans;
DROP FUNCTION audio_identity_immutable();
