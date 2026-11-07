-- +goose Up
CREATE TABLE audio_publications (
 clip_id bigint PRIMARY KEY REFERENCES clips(id) ON DELETE CASCADE,
 revision_id text NOT NULL REFERENCES transcript_revisions(id) ON DELETE CASCADE,
 published_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE VIEW audio_visible_revisions AS
 SELECT r.*,p.published_at visible_at FROM audio_publications p JOIN transcript_revisions r ON r.id=p.revision_id
 UNION ALL
 SELECT r.*,j.updated_at visible_at FROM audio_visible_legacy_transcripts j JOIN transcript_revisions r ON r.legacy_job_id=j.id
 WHERE NOT EXISTS(SELECT 1 FROM audio_publications p WHERE p.clip_id=j.clip_id);
-- Source identities, not visible ranks, remain the keys of corrections and vectors.
CREATE VIEW audio_visible_segments AS
 SELECT r.clip_id,r.id revision_id,(m.seg->>'index')::int visible_index,
 (src->>'job_id')::bigint job_id,(src->>'segment_index')::int idx,(src->>'track_index')::int stream,
 (m.seg->>'start')::double precision seg_start,(m.seg->>'end')::double precision seg_end,
 (j.result->'segments'->((src->>'segment_index')::int)) ||
 jsonb_build_object('start',m.seg->'start','end',m.seg->'end','sources',m.seg->'sources',
 'transcription_job_id',(src->>'job_id')::bigint,'source_segment_index',(src->>'segment_index')::int,'audio_stream_index',(src->>'track_index')::int,
 'words',COALESCE((SELECT jsonb_agg((wj.result->'segments'->((ws->>'segment_index')::int)->'words'->((ws->>'word_index')::int)) ||
 jsonb_build_object('start',w.word->'start','end',w.word->'end','sources',w.word->'sources') ORDER BY (w.word->>'index')::int)
 FROM jsonb_array_elements(m.seg->'words') w(word)
 CROSS JOIN LATERAL (SELECT w.word->'sources'->0 ws) first_word
 JOIN jobs wj ON wj.id=(ws->>'job_id')::bigint),'[]'::jsonb)) seg
 FROM audio_visible_revisions r CROSS JOIN LATERAL jsonb_array_elements(r.manifest->'segments') m(seg)
 CROSS JOIN LATERAL (SELECT m.seg->'sources'->0 src) first_source
 JOIN jobs j ON j.id=(src->>'job_id')::bigint;
CREATE VIEW audio_visible_transcripts AS
 SELECT r.clip_id,COALESCE(r.legacy_job_id,(SELECT s.job_id FROM audio_visible_segments s WHERE s.clip_id=r.clip_id ORDER BY visible_index LIMIT 1),0) id,
 COALESCE(j.result,'{}'::jsonb) || jsonb_build_object('revision',r.manifest,'segments',
 COALESCE((SELECT jsonb_agg(s.seg ORDER BY s.visible_index) FROM audio_visible_segments s WHERE s.clip_id=r.clip_id),'[]'::jsonb)) result,r.visible_at updated_at
 FROM audio_visible_revisions r LEFT JOIN jobs j ON j.id=COALESCE(r.legacy_job_id,(SELECT s.job_id FROM audio_visible_segments s WHERE s.clip_id=r.clip_id ORDER BY visible_index LIMIT 1));

-- +goose Down
DROP VIEW audio_visible_transcripts,audio_visible_segments,audio_visible_revisions;
DROP TABLE audio_publications;
