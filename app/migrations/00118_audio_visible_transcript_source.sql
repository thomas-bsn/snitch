-- +goose Up
-- Resolve the first source directly from the immutable ordered manifest. Expanding
-- the complete segment/word views twice just to find this ID inflated planner
-- costs enough to trigger JIT for short OLTP reads. Keep database JIT policy intact.
CREATE OR REPLACE VIEW audio_visible_transcripts AS
 SELECT r.clip_id,COALESCE(r.legacy_job_id,(r.manifest->'segments'->0->'sources'->0->>'job_id')::bigint,0) id,
 COALESCE(j.result,'{}'::jsonb) || jsonb_build_object('revision',r.manifest,'segments',
 COALESCE((SELECT jsonb_agg(s.seg ORDER BY s.visible_index) FROM audio_visible_segments s WHERE s.clip_id=r.clip_id),'[]'::jsonb)) result,r.visible_at updated_at
 FROM audio_visible_revisions r LEFT JOIN jobs j
 ON j.id=COALESCE(r.legacy_job_id,(r.manifest->'segments'->0->'sources'->0->>'job_id')::bigint);

-- +goose Down
CREATE OR REPLACE VIEW audio_visible_transcripts AS
 SELECT r.clip_id,COALESCE(r.legacy_job_id,(SELECT s.job_id FROM audio_visible_segments s WHERE s.clip_id=r.clip_id ORDER BY visible_index LIMIT 1),0) id,
 COALESCE(j.result,'{}'::jsonb) || jsonb_build_object('revision',r.manifest,'segments',
 COALESCE((SELECT jsonb_agg(s.seg ORDER BY s.visible_index) FROM audio_visible_segments s WHERE s.clip_id=r.clip_id),'[]'::jsonb)) result,r.visible_at updated_at
 FROM audio_visible_revisions r LEFT JOIN jobs j ON j.id=COALESCE(r.legacy_job_id,(SELECT s.job_id FROM audio_visible_segments s WHERE s.clip_id=r.clip_id ORDER BY visible_index LIMIT 1));
