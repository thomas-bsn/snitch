package httpapi

import "net/http"

// The administration setting "Analyse complète automatique" (#215, ADR 0035). It is on unless the
// stored value is "false"; an Administrator reads and changes it here, and it applies to the next
// clip added to the library.

func (s *Server) getAutomaticAnalysis(w http.ResponseWriter, r *http.Request) {
	value, err := s.Store.Get(r.Context(), settingAutoFullAnalysis)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]bool{"enabled": value != "false"})
}

func (s *Server) putAutomaticAnalysis(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Enabled == nil {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	value := "true"
	if !*input.Enabled {
		value = "false"
	}
	if err := s.Store.Set(r.Context(), settingAutoFullAnalysis, value); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]bool{"enabled": *input.Enabled})
}
