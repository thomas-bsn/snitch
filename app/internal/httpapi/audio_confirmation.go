package httpapi

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"snitch/internal/audioplans"
	"snitch/internal/httpapi/contract"
)

// The confirmation of the losses of a plan whose publication is held (ADR 0054). The preview that was confirmed
// when the plan started is checked again at publication: a correction, a restoration or a voice validation made
// since stops it ("audio_publication_changed"), and so does a missing confirmation. These routes show the
// current losses and record a new confirmation; the orchestrator then publishes by itself.

func (s *Server) planPublicationView(r *http.Request, clipID, planID int64) (contract.AudioPublicationPreview, int, string) {
	store := audioplans.New(s.Pool)
	var clip int64
	if err := s.Pool.QueryRow(r.Context(), `SELECT clip_id FROM audio_plans WHERE id=$1`, planID).Scan(&clip); errors.Is(err, pgx.ErrNoRows) || err == nil && clip != clipID {
		return contract.AudioPublicationPreview{}, http.StatusNotFound, "audio_plan_not_found"
	} else if err != nil {
		return contract.AudioPublicationPreview{}, http.StatusInternalServerError, ""
	}
	preview, err := store.PreviewClipPublication(r.Context(), clipID, planID)
	if err != nil {
		return contract.AudioPublicationPreview{}, http.StatusInternalServerError, ""
	}
	out := contract.AudioPublicationPreview{PlanId: planID, Guard: preview.Guard, Losses: preview.AudioLossPreview}
	if stored, err := store.GetConfirmation(r.Context(), planID); err == nil {
		out.ConfirmationCurrent = stored.Guard == preview.Guard && stored.ExpectedRevisionID == preview.ExpectedRevisionId
		out.Confirmed = out.ConfirmationCurrent && stored.Confirmed
	}
	return out, 0, ""
}

func (s *Server) getAudioPublication(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	planID, ok := planIDOf(w, r)
	if !ok {
		return
	}
	view, status, code := s.planPublicationView(r, clip.ID, planID)
	if status != 0 {
		if code == "" {
			s.internal(w, errors.New("read audio publication"))
			return
		}
		fail(w, status, code)
		return
	}
	reply(w, http.StatusOK, view)
}

func (s *Server) confirmAudioPublication(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	planID, ok := planIDOf(w, r)
	if !ok {
		return
	}
	var input contract.AudioPublicationConfirm
	if !decode(w, r, &input) {
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	// Same lock as the corrections: what is confirmed is exactly what the publication will check.
	if _, err = tx.Exec(r.Context(), `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, clip.ID); err != nil {
		s.internal(w, err)
		return
	}
	var owner int64
	var active bool
	err = tx.QueryRow(r.Context(), `SELECT p.clip_id,r.status='active' AND NOT EXISTS(SELECT 1 FROM audio_plans n WHERE n.run_id=r.id AND n.id>p.id) FROM audio_plans p JOIN clip_analysis_runs r ON r.id=p.run_id WHERE p.id=$1`, planID).Scan(&owner, &active)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && owner != clip.ID {
		fail(w, http.StatusNotFound, "audio_plan_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if !active {
		fail(w, http.StatusConflict, "audio_plan_superseded")
		return
	}
	store := audioplans.New(tx)
	preview, err := store.PreviewClipPublication(r.Context(), clip.ID, planID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if input.Guard != preview.Guard {
		fail(w, http.StatusConflict, "audio_preview_changed")
		return
	}
	if preview.ConfirmationRequired && !input.ConfirmLosses {
		fail(w, http.StatusConflict, "audio_confirmation_required")
		return
	}
	if err = store.RecordConfirmation(r.Context(), audioplans.Confirmation{PlanID: planID, ExpectedRevisionID: preview.ExpectedRevisionId, Guard: preview.Guard, Confirmed: preview.ConfirmationRequired && input.ConfirmLosses, ConfirmedBy: sessionUserID(r)}); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	s.reconcileAnalysis(r.Context(), clip.ID)
	view, status, code := s.planPublicationView(r, clip.ID, planID)
	if status != 0 {
		if code == "" {
			s.internal(w, errors.New("read audio publication"))
			return
		}
		fail(w, status, code)
		return
	}
	reply(w, http.StatusOK, view)
}
