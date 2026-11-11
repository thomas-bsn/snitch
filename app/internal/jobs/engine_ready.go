package jobs

// Engines whose code ships inside the Worker image: they need no approved runtime (ADR 0055). Every other
// engine only runs on a Worker whose runtime release is installed and ready (issue #350).
const embeddedEngines = `('','pyannote','nemo','silero-jit')`

// RuntimeReadyClause is the SQL condition that makes a model that a Worker reports ready really usable: its
// pinned revision is the catalogue's, and, for an engine that needs an approved runtime, that runtime is ready
// in the same Worker's engine inventory. wvm is the alias of worker_voice_models, m the one of
// voice_model_catalog. The published binding takes precedence; without one, an install order that froze a
// release requires that very release to be ready;
// otherwise any ready release of the engine and kind will do. A model that names an on-demand family (ADR 0056)
// is never embedded, whatever its adapter is called: the ready engine must be the on-demand one of that family
// and of the model's kind. A separation model follows the same rule (#401); that makes it usable by a job that
// asks for it, and by none of the library: no step of an analysis separates and no separation model can be made
// active. A revoked release stays usable: a binding that exists keeps serving. Nothing here orders or downloads anything. Embedded engines keep their legacy contract.
func RuntimeReadyClause(wvm, m string) string {
	return `((` + m + `.family='' AND ` + m + `.engine IN ` + embeddedEngines + `) OR (
		          ` + wvm + `.revision=` + m + `.revision AND EXISTS (
		            SELECT 1 FROM worker_engine_inventory ei JOIN voice_engine_registry er ON er.digest=ei.digest
		            WHERE ei.worker_id=` + wvm + `.worker_id AND ei.state='ready'
		              AND ((` + m + `.family='' AND er.engine=` + m + `.engine)
		                   OR (` + m + `.family<>'' AND er.source='on_demand' AND er.manifest->>'family'=` + m + `.family))
		              AND er.manifest->>'kind'=` + m + `.kind
		              AND (EXISTS (SELECT 1 FROM worker_model_runtime_bindings rb WHERE rb.worker_id=` + wvm + `.worker_id AND rb.model_id=` + wvm + `.model_id AND rb.digest=ei.digest)
		                   OR (NOT EXISTS (SELECT 1 FROM worker_model_runtime_bindings rb WHERE rb.worker_id=` + wvm + `.worker_id AND rb.model_id=` + wvm + `.model_id)
		                       AND (NOT EXISTS (SELECT 1 FROM worker_model_orders wo WHERE wo.id=` + wvm + `.order_id AND wo.runtime IS NOT NULL)
		                            OR EXISTS (SELECT 1 FROM worker_model_orders wo WHERE wo.id=` + wvm + `.order_id AND wo.runtime=er.manifest))))))
		    )`
}
