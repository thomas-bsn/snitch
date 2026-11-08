package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"snitch/internal/auth"
)

// Replacing the model of a type on a Worker (ADR 0038). Snitch drives every step; the Worker only runs orders:
//
//	preparing -> the new model is installed and verified (install order) while the old one stays in service
//	switching -> the new one is ready: Snitch switches the active model, then waits for the running jobs that use
//	             the old one to end
//	removing  -> Snitch has sent the remove order of the old one; the Worker reports the removal
//
// The old model is never removed on the Worker's own initiative, only by that order, and a failure of the new
// one (download, test, disk), a job still using the old one or a failed removal keep the old one and say so.
// Only the diarization model can be switched here: the fingerprint model needs a versioned transition (#311).

type replacement struct {
	WorkerID, Kind, Old, New, Phase, Cause string
	AuthorID                               *int64
	AuthorName                             string
}

const replacementColumns = `worker_id,kind,old_model_id,new_model_id,phase,cause_code,author_id,author_name`

func scanReplacement(row pgx.Row) (replacement, error) {
	var r replacement
	err := row.Scan(&r.WorkerID, &r.Kind, &r.Old, &r.New, &r.Phase, &r.Cause, &r.AuthorID, &r.AuthorName)
	return r, err
}

func replacementsIn(ctx context.Context, db pgQuerier, where string, args ...any) ([]replacement, error) {
	rows, err := db.Query(ctx, `SELECT `+replacementColumns+` FROM worker_model_replacements `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []replacement{}
	for rows.Next() {
		r, err := scanReplacement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// checkReplacement validates the replaces_model_id of an install order and returns the model replaced, or ""
// for an ordinary installation. Installing again the successor of a replacement (Retry) is the same replacement.
func checkReplacement(ctx context.Context, tx pgx.Tx, workerID string, entry voiceCatalogEntry, replaces string) (string, error) {
	existing, err := replacementsIn(ctx, tx, `WHERE worker_id=$1 AND kind=$2`, workerID, entry.Kind)
	if err != nil {
		return "", err
	}
	if replaces == "" {
		if len(existing) == 1 && existing[0].New == entry.ID && existing[0].Phase == "preparing" {
			return existing[0].Old, nil
		}
		return "", nil
	}
	if entry.Kind != "diarization" && entry.Kind != "fingerprint" {
		return "", orderFailure{http.StatusConflict, "voice_replacement_kind_unsupported"}
	}
	old, ok, err := voiceCatalogByID(ctx, tx, replaces)
	if err != nil {
		return "", err
	}
	if !ok || old.Kind != entry.Kind || old.ID == entry.ID {
		return "", orderFailure{http.StatusBadRequest, "invalid_voice_model"}
	}
	var state string
	switch err = tx.QueryRow(ctx, `SELECT state FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, old.ID).Scan(&state); {
	case errors.Is(err, pgx.ErrNoRows):
		return "", orderFailure{http.StatusConflict, "voice_replaced_model_not_ready"}
	case err != nil:
		return "", err
	case state != "ready":
		return "", orderFailure{http.StatusConflict, "voice_replaced_model_not_ready"}
	}
	if entry.Kind == "fingerprint" {
		// Changing the fingerprint model makes the stored fingerprints incomparable: its replacement on a Worker
		// only prepares the target of a confirmed transition (ADR 0044), which alone decides the switch.
		t, open, err := openTransition(ctx, tx)
		if err != nil {
			return "", err
		}
		if !open || t.From != old.ID || t.To != entry.ID {
			return "", orderFailure{http.StatusConflict, "voice_transition_required"}
		}
	}
	if len(existing) == 1 && (existing[0].New != entry.ID || existing[0].Old != old.ID || existing[0].Phase != "preparing") {
		var failed bool
		if existing[0].Old == old.ID && existing[0].Phase == "preparing" {
			err = tx.QueryRow(ctx, `SELECT state='failed' FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, existing[0].New).Scan(&failed)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return "", err
			}
		}
		if !failed {
			return "", orderFailure{http.StatusConflict, "voice_replacement_in_progress"}
		}
	}
	return old.ID, nil
}

// voiceModelInUseIn tells whether a job that is not finished still uses the model on one Worker: a job pinned (by
// its pipeline version) to its repository, or one that asks for it explicitly. Only the jobs that Worker holds
// (lease_owner) count, plus the waiting ones that no other Worker could take because it is the only holder of the
// model: a job running elsewhere never blocks this Worker. Pending jobs count: they would otherwise be left waiting
// for a model nobody holds any more.
func voiceModelInUseIn(ctx context.Context, db pgQuerier, entry voiceCatalogEntry, workerID string) (bool, error) {
	// $1 is the model, $2 the Worker. A waiting job counts only when no other live Worker holds the model ready.
	const scope = `AND ((j.status='leased' AND j.lease_owner=$2) OR (j.status IN ('pending','retry_wait') AND NOT EXISTS (
		SELECT 1 FROM worker_voice_models o JOIN workers w ON w.id=o.worker_id WHERE o.model_id=$1 AND o.state='ready' AND o.worker_id<>$2 AND w.revoked_at IS NULL)))`
	var busy bool
	if entry.Kind == "fingerprint" {
		// Fingerprint jobs are not pinned: the active model serves the production ones, a requested model its own.
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs j WHERE j.step LIKE 'voice.fingerprint.%' AND j.status IN ('pending','leased','retry_wait') `+scope+`
			AND (j.requested_model_id=$1 OR (j.requested_model_id IS NULL AND EXISTS (SELECT 1 FROM voice_active_model a WHERE a.kind='fingerprint' AND a.model_id=$1))))`, entry.ID, workerID).Scan(&busy)
		return busy, err
	}
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs j LEFT JOIN voice_pipeline_versions pin ON pin.version=substring(j.input_version FROM '-p(voice-[0-9a-f]+)')
		WHERE j.step LIKE 'voice.diarization.%' AND j.status IN ('pending','leased','retry_wait') `+scope+` AND (j.requested_model_id=$1 OR pin.config->>'model'=$3))`, entry.ID, workerID, entry.Repo).Scan(&busy)
	return busy, err
}

// advanceReplacementsIn moves the replacements of one Worker as far as the facts allow. It runs under the
// voice model lock, after a report of the Worker and at each of its pulls, so a job that ends is noticed.
func (s *Server) advanceReplacementsIn(ctx context.Context, tx pgx.Tx, workerID string) error {
	list, err := replacementsIn(ctx, tx, `WHERE worker_id=$1 AND phase IN ('preparing','switching')`, workerID)
	if err != nil {
		return err
	}
	for _, rep := range list {
		if rep.Phase == "preparing" {
			var state string
			err = tx.QueryRow(ctx, `SELECT state FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, rep.WorkerID, rep.New).Scan(&state)
			if errors.Is(err, pgx.ErrNoRows) || (err == nil && state != "ready") {
				continue
			}
			if err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `UPDATE worker_model_replacements SET phase='switching',updated_at=now() WHERE worker_id=$1 AND kind=$2`, rep.WorkerID, rep.Kind); err != nil {
				return err
			}
			rep.Phase = "switching"
		}
		if rep.Phase == "switching" && rep.Cause == "" {
			if err = s.settleReplacement(ctx, tx, rep); err != nil {
				return err
			}
		}
	}
	return nil
}

// settleReplacement decides the switch, then the removal, of a replacement whose new model is ready.
func (s *Server) settleReplacement(ctx context.Context, tx pgx.Tx, rep replacement) error {
	next, ok, err := voiceCatalogByID(ctx, tx, rep.New)
	if err != nil || !ok {
		return err
	}
	old, ok, err := voiceCatalogByID(ctx, tx, rep.Old)
	if err != nil || !ok {
		return err
	}
	active, err := voiceActiveModelIDIn(ctx, tx)
	if err != nil {
		return err
	}
	if rep.Kind == "fingerprint" {
		// The switch of the fingerprint model is the transition's: the old model stays on the Worker until the
		// essential migration is complete, then its removal follows the usual rules.
		var phase string
		err = tx.QueryRow(ctx, `SELECT phase FROM voice_fingerprint_transitions WHERE from_model_id=$1 AND to_model_id=$2 ORDER BY id DESC LIMIT 1`, old.ID, next.ID).Scan(&phase)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && phase != "switched") {
			return nil
		}
		if err != nil {
			return err
		}
	} else if active == old.ID {
		// The switch is Snitch's decision, taken once a connected Worker holds the new model ready.
		_, _, err = switchVoicePipelineIn(ctx, tx, &next, func(current voicePipeline) voicePipeline {
			current.Provider, current.Model, current.ModelVersion, current.Fake = next.Engine, next.Repo, next.Revision, nil
			return current
		})
		if errors.Is(err, errVoiceModelNotReady) {
			return nil
		}
		if err != nil {
			return err
		}
		s.Log.Info("voice model switched", "worker_id", rep.WorkerID, "from", old.ID, "to", next.ID)
	}
	busy, err := voiceModelInUseIn(ctx, tx, old, rep.WorkerID)
	if err != nil || busy {
		return err
	}
	id := "ord_" + auth.Token()[:32]
	if _, err = tx.Exec(ctx, `INSERT INTO worker_model_orders(id,worker_id,kind,action,model_id,revision,idempotency_key,author_id,author_name)
		SELECT $1,$2,$3,'remove',$4,COALESCE(NULLIF(m.revision,''),c.revision),$5,$6,$7 FROM worker_voice_models m JOIN voice_model_catalog c ON c.id=m.model_id WHERE m.worker_id=$2 AND m.model_id=$4`,
		id, rep.WorkerID, rep.Kind, old.ID, "remove-"+id, rep.AuthorID, rep.AuthorName); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE worker_voice_models SET state='removing',order_id=$3,updated_at=now() WHERE worker_id=$1 AND model_id=$2`, rep.WorkerID, old.ID, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE worker_model_replacements SET phase='removing',remove_order_id=$3,updated_at=now() WHERE worker_id=$1 AND kind=$2`, rep.WorkerID, rep.Kind, id); err != nil {
		return err
	}
	s.Log.Info("voice model removal ordered", "order_id", id, "worker_id", rep.WorkerID, "model_id", old.ID)
	return nil
}

// advanceWorkerReplacements is called at each pull of a Worker: cheap when it has no replacement to move.
func (s *Server) advanceWorkerReplacements(ctx context.Context, workerID string) error {
	var open bool
	if err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM worker_model_replacements WHERE worker_id=$1 AND phase IN ('preparing','switching') AND cause_code='')`, workerID).Scan(&open); err != nil || !open {
		return err
	}
	return s.withVoiceModelLock(ctx, func(tx pgx.Tx) error { return s.advanceReplacementsIn(ctx, tx, workerID) })
}

// removalDone forgets the old model once the Worker reports it removed: the replacement is complete.
func removalDone(ctx context.Context, tx pgx.Tx, workerID, modelID string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2`, workerID, modelID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM worker_model_replacements WHERE worker_id=$1 AND old_model_id=$2`, workerID, modelID)
	return err
}

// removalFailed keeps the old model, which is still on the Worker, and says why; Retry orders the removal again.
func removalFailed(ctx context.Context, tx pgx.Tx, workerID, modelID, cause string) error {
	if _, err := tx.Exec(ctx, `UPDATE worker_voice_models SET state='ready',updated_at=now() WHERE worker_id=$1 AND model_id=$2 AND state='removing'`, workerID, modelID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE worker_model_replacements SET phase='switching',cause_code=$3,updated_at=now() WHERE worker_id=$1 AND old_model_id=$2`, workerID, modelID, cause)
	return err
}

// retryReplacementRemoval clears the failure of a removal so that the next step orders it again. It reports
// whether the model is the successor of a replacement whose removal had failed.
func (s *Server) retryReplacementRemoval(ctx context.Context, workerID, modelID string) (bool, error) {
	found := false
	err := s.withVoiceModelLock(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE worker_model_replacements SET cause_code='',updated_at=now() WHERE worker_id=$1 AND new_model_id=$2 AND phase='switching' AND cause_code<>''`, workerID, modelID)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		found = true
		return s.advanceReplacementsIn(ctx, tx, workerID)
	})
	return found, err
}

// cancelVoiceModelReplacement is "Cancel the replacement": allowed while the new model is not in service. A
// download or test under way cannot be interrupted, so it must end (or fail) first.
func (s *Server) cancelVoiceModelReplacement(w http.ResponseWriter, r *http.Request) {
	workerID, kind := r.PathValue("id"), r.PathValue("kind")
	err := s.withVoiceModelLock(r.Context(), func(tx pgx.Tx) error {
		list, err := replacementsIn(r.Context(), tx, `WHERE worker_id=$1 AND kind=$2`, workerID, kind)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			return orderFailure{http.StatusNotFound, "voice_replacement_not_found"}
		}
		rep := list[0]
		if rep.Phase != "preparing" {
			return orderFailure{http.StatusConflict, "voice_replacement_switched"}
		}
		var state, status string
		err = tx.QueryRow(r.Context(), `SELECT m.state,COALESCE(o.status,'') FROM worker_voice_models m LEFT JOIN worker_model_orders o ON o.id=m.order_id WHERE m.worker_id=$1 AND m.model_id=$2`, workerID, rep.New).Scan(&state, &status)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return err
		case state == "failed" || (state == "requested" && status == "pending"):
			if _, err = tx.Exec(r.Context(), `UPDATE worker_model_orders SET status='cancelled',cause_code='',finished_at=now() WHERE id=(SELECT order_id FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2) AND status='pending'`, workerID, rep.New); err != nil {
				return err
			}
			// A failed attempt can still hold partial files. Retain its inventory for explicit cleanup.
			if _, err = tx.Exec(r.Context(), `DELETE FROM worker_voice_models WHERE worker_id=$1 AND model_id=$2 AND state='requested'`, workerID, rep.New); err != nil {
				return err
			}
		default:
			return orderFailure{http.StatusConflict, "voice_replacement_in_progress"}
		}
		_, err = tx.Exec(r.Context(), `DELETE FROM worker_model_replacements WHERE worker_id=$1 AND kind=$2`, workerID, kind)
		return err
	})
	if err != nil {
		if !s.writeOrderFailure(w, err) {
			s.internal(w, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// heldModelCells is the model of each type on each Worker for the "Models per Worker" view: the model that is
// not failed wins over a failed attempt, then the latest; during a replacement it is the successor, shown as
// switching (the switch is decided, the old one is kept while a job uses it) or removing.
func (s *Server) heldModelCells(ctx context.Context) (map[[2]string]workerModelCell, error) {
	reps, err := replacementsIn(ctx, s.Pool, ``)
	if err != nil {
		return nil, err
	}
	byKey := map[[2]string]replacement{}
	for _, rep := range reps {
		byKey[[2]string{rep.WorkerID, rep.Kind}] = rep
	}
	rows, err := s.Pool.Query(ctx, `SELECT m.worker_id,c.kind,m.model_id,c.label,c.repo,m.revision,m.state,m.cause_code,m.bytes_received,m.bytes_total,m.occupied_bytes,COALESCE(m.order_id,'')
		FROM worker_voice_models m JOIN voice_model_catalog c ON c.id=m.model_id ORDER BY m.worker_id, c.kind, (m.state='failed'), m.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	held := map[[2]string]workerModelCell{}
	for rows.Next() {
		var workerID string
		var c workerModelCell
		if err = rows.Scan(&workerID, &c.Kind, &c.ModelID, &c.Label, &c.Repo, &c.Revision, &c.State, &c.CauseCode, &c.BytesReceived, &c.BytesTotal, &c.OccupiedBytes, &c.OrderID); err != nil {
			return nil, err
		}
		key := [2]string{workerID, c.Kind}
		if rep, replacing := byKey[key]; replacing {
			if c.ModelID == rep.New {
				held[key] = c
			}
			continue
		}
		if _, seen := held[key]; !seen {
			held[key] = c
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for key, rep := range byKey {
		c, ok := held[key]
		if !ok {
			continue
		}
		c.ReplacesModelID = rep.Old
		c.ReplacementPhase = rep.Phase
		switch {
		case rep.Phase == "removing":
			c.State = "removing"
		case rep.Phase == "switching" && rep.Cause != "":
			c.State, c.CauseCode = "failed", rep.Cause
		case rep.Phase == "switching":
			c.State = "switching"
			old, found, err := voiceCatalogByID(ctx, s.Pool, rep.Old)
			if err != nil {
				return nil, err
			}
			if found {
				if c.WaitingForJobs, err = voiceModelInUseIn(ctx, s.Pool, old, rep.WorkerID); err != nil {
					return nil, err
				}
			}
		}
		held[key] = c
	}
	return held, nil
}

// orderRemovalOn is the administrator's explicit uninstall on one Worker: one remove order for that Worker, if it
// holds the model (ready or failed), which the Worker runs and reports. Nothing is deleted on Snitch's side before.
// It reports whether the Worker held the model.
func orderRemovalOn(ctx context.Context, tx pgx.Tx, entry voiceCatalogEntry, workerID string, author session) (bool, error) {
	var revision string
	err := tx.QueryRow(ctx, `SELECT COALESCE(NULLIF(m.revision,''),$3) FROM worker_voice_models m JOIN workers w ON w.id=m.worker_id
		WHERE m.model_id=$1 AND m.worker_id=$2 AND m.state IN ('ready','failed') AND w.revoked_at IS NULL`, entry.ID, workerID, entry.Revision).Scan(&revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	id := "ord_" + auth.Token()[:32]
	if _, err = tx.Exec(ctx, `INSERT INTO worker_model_orders(id,worker_id,kind,action,model_id,revision,idempotency_key,author_id,author_name) VALUES($1,$2,$3,'remove',$4,$5,$6,$7,$8)`,
		id, workerID, entry.Kind, entry.ID, revision, "remove-"+id, author.User.ID, author.User.Username); err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, `UPDATE worker_voice_models SET state='removing',order_id=$3,updated_at=now() WHERE worker_id=$1 AND model_id=$2`, workerID, entry.ID, id)
	return err == nil, err
}
