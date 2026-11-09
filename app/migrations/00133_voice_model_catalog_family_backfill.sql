-- +goose Up
-- Existing catalogue rows never got a family (00132 added the column empty), so a model that was in the catalogue
-- before the on-demand engines could not use them (ADR 0056). Give a family only where the adapter maps to
-- exactly one: the TitaNet adapter runs on NeMo, the ECAPA-TDNN adapter on SpeechBrain, the WeSpeaker adapter
-- only exists as an on-demand family. The seeded Sortformer and any other `nemo` diarization row are NOT touched:
-- an embedded Sortformer and a newer NeMo model share the adapter name and only the owner can tell them apart
-- (PATCH /api/admin/voice-models/{id} sets or clears the family). A row that already runs on a signed Release
-- on some Worker keeps its empty family too: moving it would close its dispatch there until the engine is ordered.
-- The statement is idempotent and only fills empty families.
UPDATE voice_model_catalog c SET family = CASE c.engine WHEN 'nemo-titanet' THEN 'nemo' WHEN 'speechbrain-ecapa' THEN 'speechbrain' ELSE 'wespeaker' END, updated_at = now()
WHERE c.family = '' AND c.kind = 'fingerprint' AND c.engine IN ('nemo-titanet', 'speechbrain-ecapa', 'wespeaker')
  AND NOT EXISTS (SELECT 1 FROM worker_model_runtime_bindings b WHERE b.model_id = c.id)
  AND NOT EXISTS (SELECT 1 FROM worker_model_orders o WHERE o.model_id = c.id AND o.runtime IS NOT NULL);

-- +goose Down
-- Not reversible on purpose: nothing tells a family filled here from one the owner chose. The column stays as it is.
SELECT 1;
