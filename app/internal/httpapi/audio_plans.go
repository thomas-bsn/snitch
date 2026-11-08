package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"snitch/internal/audioplans"
)

func (s *Server) getAudioPlan(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("plan_id"), 10, 64)
	if err != nil || id < 1 {
		fail(w, 400, "invalid_plan_id")
		return
	}
	state, err := audioplans.New(s.Pool).GetPlan(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && state.Plan.ClipId != clip.ID {
		fail(w, 404, "audio_plan_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, state)
}
