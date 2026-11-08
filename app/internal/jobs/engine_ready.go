package jobs

// Engines whose code ships inside the Worker image: they need no approved runtime (ADR 0055). Every other
// engine only runs on a Worker whose runtime release is installed and ready (issue #350).
const embeddedEngines = `('','pyannote','nemo','silero-jit')`

// RuntimeReadyClause is the SQL condition that makes a model that a Worker reports ready really usable: its
// pinned revision is the catalogue's, and, for an engine that needs an approved runtime, that runtime is ready
// in the same Worker's engine inventory. wvm is the alias of worker_voice_models, m the one of
// voice_model_catalog. When the install order froze a release, that very release must be the ready one;
// otherwise any ready release of the engine and kind will do. A revoked release stays usable: a binding that
// exists keeps serving. Nothing here orders or downloads anything. Embedded engines keep their legacy contract.
func RuntimeReadyClause(wvm, m string) string {
	return `(` + m + `.engine IN ` + embeddedEngines + ` OR (
		          ` + wvm + `.revision=` + m + `.revision AND EXISTS (
		            SELECT 1 FROM worker_engine_inventory ei JOIN voice_engine_registry er ON er.digest=ei.digest
		            WHERE ei.worker_id=` + wvm + `.worker_id AND ei.state='ready'
		              AND er.engine=` + m + `.engine AND er.manifest->>'kind'=` + m + `.kind
		              AND (NOT EXISTS (SELECT 1 FROM worker_model_orders wo WHERE wo.id=` + wvm + `.order_id AND wo.runtime IS NOT NULL)
		                   OR EXISTS (SELECT 1 FROM worker_model_orders wo WHERE wo.id=` + wvm + `.order_id AND wo.runtime=er.manifest))))
		    )`
}
