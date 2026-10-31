package httpapi

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"snitch/internal/setup"
)

// setupSteps are the optional steps of the guided installation, in the order the wizard shows them.
var setupSteps = []string{"location", "discord", "semantic_search", "worker", "import"}

type setupStepState struct {
	Step  string `json:"step"`
	State string `json:"state"`
}

type setupProgress struct {
	Steps     []setupStepState `json:"steps"`
	Dismissed bool             `json:"dismissed"`
}

// setupConfigured reads what is really configured on the instance for each optional step.
func (s *Server) setupConfigured(ctx context.Context) (map[string]bool, error) {
	id, err := s.Store.Get(ctx, "discord_client_id")
	if err != nil {
		return nil, err
	}
	secret, err := s.Store.Get(ctx, "discord_client_secret")
	if err != nil {
		return nil, err
	}
	var semantic, worker, imported, location bool
	if err := s.Pool.QueryRow(ctx, `SELECT
 EXISTS(SELECT 1 FROM embedding_index_state WHERE id=1 AND active_profile_id IS NOT NULL),
 EXISTS(SELECT 1 FROM workers WHERE revoked_at IS NULL),
 EXISTS(SELECT 1 FROM clips),
 EXISTS(SELECT 1 FROM clip_metadata_settings WHERE id=true AND country<>'' AND timezone<>'')`).Scan(&semantic, &worker, &imported, &location); err != nil {
		return nil, err
	}
	return map[string]bool{"location": location, "discord": id != "" && secret != "", "semantic_search": semantic, "worker": worker, "import": imported}, nil
}

// loadSetupProgress merges the recorded steps with the real state: a configured step is done whatever was
// recorded, a step the person went through without configuring it is skipped, any other is still to do.
func (s *Server) loadSetupProgress(ctx context.Context) (setupProgress, error) {
	configured, err := s.setupConfigured(ctx)
	if err != nil {
		return setupProgress{}, err
	}
	recorded := map[string]bool{}
	rows, err := s.Pool.Query(ctx, `SELECT step FROM setup_steps`)
	if err != nil {
		return setupProgress{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var step string
		if err := rows.Scan(&step); err != nil {
			return setupProgress{}, err
		}
		recorded[step] = true
	}
	if err := rows.Err(); err != nil {
		return setupProgress{}, err
	}
	out := setupProgress{Steps: make([]setupStepState, 0, len(setupSteps))}
	for _, step := range setupSteps {
		state := "todo"
		if configured[step] {
			state = "done"
		} else if recorded[step] {
			state = "skipped"
		}
		out.Steps = append(out.Steps, setupStepState{Step: step, State: state})
	}
	if err := s.Pool.QueryRow(ctx, `SELECT setup_banner_dismissed FROM installation WHERE id=1`).Scan(&out.Dismissed); err != nil {
		return setupProgress{}, err
	}
	return out, nil
}

func (s *Server) getSetupProgress(w http.ResponseWriter, r *http.Request) {
	progress, err := s.loadSetupProgress(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, progress)
}

func (s *Server) putSetupStep(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Step  *string `json:"step"`
		State *string `json:"state"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Step == nil || in.State == nil || !slices.Contains(setupSteps, *in.Step) || (*in.State != "done" && *in.State != "skipped") {
		fail(w, 400, "invalid_request")
		return
	}
	if _, err := s.Pool.Exec(r.Context(), `INSERT INTO setup_steps(step,state) VALUES($1,$2) ON CONFLICT (step) DO UPDATE SET state=EXCLUDED.state,updated_at=now()`, *in.Step, *in.State); err != nil {
		s.internal(w, err)
		return
	}
	s.getSetupProgress(w, r)
}

func (s *Server) dismissSetupBanner(w http.ResponseWriter, r *http.Request) {
	if _, err := s.Pool.Exec(r.Context(), `UPDATE installation SET setup_banner_dismissed=true WHERE id=1`); err != nil {
		s.internal(w, err)
		return
	}
	s.getSetupProgress(w, r)
}

// checkSetupCode tells, as soon as it is typed, whether the installation code is right and still valid. It
// changes nothing: the code stays usable once, by the creation of the first account.
func (s *Server) checkSetupCode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code *string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Code == nil {
		fail(w, 400, "invalid_request")
		return
	}
	if err := setup.CheckCode(r.Context(), s.Pool, *in.Code); errors.Is(err, setup.ErrClosed) {
		fail(w, 409, "setup_closed_or_invalid_code")
		return
	} else if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 200, map[string]string{"status": "valid"})
}
