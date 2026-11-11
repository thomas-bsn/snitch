package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"snitch/internal/events"
	"snitch/internal/logging"

	"github.com/jackc/pgx/v5"
)

// Voice matching in the background (#392, spec voix-breves section 7).
//
// A change of references, fingerprints or settings records a request (table voice_matching_requests): in the
// transaction of the change when the caller has one (requestMatching), right after its commit otherwise
// (matchingChanged). The change answers at once; a background pass then matches the library against the latest
// committed state:
//   - a pass takes every request it finds once it holds the matching lock, so the requests made while another
//     pass ran, or while it waited, make a single pass;
//   - a request is deleted only by the pass that read it, after that pass committed: a pass cut by a stop or a
//     failure leaves its requests, resumed at the next start or retry, without any user action;
//   - one request for the whole library (clip NULL) makes a full pass; otherwise each requested clip is matched
//     on its own (new fingerprints of that clip).
//
// The analysis step "matching" keeps matching its clip synchronously (rematchVoices): it runs in the analysis
// loop, already in the background, and its run needs the success of the step. Merging, separating or deleting a
// Person, erasing a voice profile and switching the fingerprint model take the same lock (voiceMatchingLock) and
// so wait for a pass in progress; a validation takes no matching lock and never waits for one.
//
// A server that runs no background loop (not started, or matchingInline for tests) runs the pass inline, in the
// caller, right after recording the request: the same code path, without the wait.

// wholeLibrary is the clip id that stands for the whole library in a matching request (clip_id NULL).
const wholeLibrary int64 = 0

// requestMatching records a matching request; clipID wholeLibrary asks for the whole library. Called inside the transaction
// of a change, the request commits with it; wakeMatching must follow the commit.
func requestMatching(ctx context.Context, q pgQuerier, clipID int64) error {
	var clip *int64
	if clipID > 0 {
		clip = &clipID
	}
	_, err := q.Exec(ctx, `INSERT INTO voice_matching_requests(clip_id) VALUES($1)`, clip)
	return err
}

// libraryMatchingChanged asks for a matching pass over the whole library after a committed change of
// references or settings (matchingChanged).
func (s *Server) libraryMatchingChanged(ctx context.Context) { s.matchingChanged(ctx, wholeLibrary) }

// matchingChanged asks for a matching pass after a change of references or fingerprints, already committed.
// clipID wholeLibrary covers the whole library; otherwise only the speakers of that clip (new fingerprints). It never waits
// for the pass: a failure is logged and never undoes the change, and the request stays until a pass succeeds.
func (s *Server) matchingChanged(ctx context.Context, clipID int64) {
	if err := requestMatching(context.WithoutCancel(ctx), s.Pool, clipID); err != nil {
		s.Log.Error("voice matching request failed", append([]any{"clip_id", clipID}, logging.Failure(err)...)...)
	}
	s.wakeMatching(ctx)
}

// wakeMatching tells the background loop that requests wait. Without a loop, the pass runs now, in the caller.
func (s *Server) wakeMatching(ctx context.Context) {
	if s.matchingWake == nil {
		if err := s.settleMatching(context.WithoutCancel(ctx)); err != nil {
			s.Log.Error("voice matching failed", logging.Failure(err)...)
		}
		return
	}
	select {
	case s.matchingWake <- struct{}{}:
	default: // a wake-up is already waiting: the pass will see every request
	}
}

// rematchNow runs one full pass at once and returns its figures: the explicit rematch of an administrator and
// the automation switch, whose answers report what matching did. The requests waiting are consumed by it.
func (s *Server) rematchNow(ctx context.Context) (matchStats, error) {
	_, stats, err := s.matchingPass(ctx, true)
	return stats, err
}

// settleMatching runs passes until no request is left.
func (s *Server) settleMatching(ctx context.Context) error {
	for {
		ran, _, err := s.matchingPass(ctx, false)
		if err != nil || !ran {
			return err
		}
	}
}

// serveVoiceMatching is the background loop: a pass when woken, and on every tick, which retries a failed pass
// and resumes the requests left by a stop (or recorded by another process).
func (s *Server) serveVoiceMatching(ctx context.Context) {
	retry := s.matchingRetry
	if retry <= 0 {
		retry = 30 * time.Second
	}
	// A pass cut by a stop may have left its mark; its requests are still there and are resumed below.
	if _, err := s.Pool.Exec(ctx, `UPDATE voice_matching_status SET running_since=NULL WHERE running_since IS NOT NULL`); err != nil && ctx.Err() == nil {
		s.Log.Error("voice matching status reset failed", logging.Failure(err)...)
	}
	ticker := time.NewTicker(retry)
	defer ticker.Stop()
	for {
		if err := s.settleMatching(ctx); err != nil && ctx.Err() == nil {
			s.Log.Error("voice matching failed", logging.Failure(err)...)
		}
		select {
		case <-ctx.Done():
			return
		case <-s.matchingWake:
		case <-ticker.C:
		}
	}
}

// matchingPass takes the waiting requests and runs one pass for them, under the matching lock; full forces a pass
// over the whole library even without request. ran is false when there was nothing to do.
func (s *Server) matchingPass(ctx context.Context, full bool) (ran bool, stats matchStats, err error) {
	voiceMatchingLock.Lock()
	defer voiceMatchingLock.Unlock()
	ids, clips, libraryAsked, err := waitingMatching(ctx, s)
	if err != nil {
		return false, stats, err
	}
	full = full || libraryAsked
	if !full && len(clips) == 0 {
		return false, stats, nil
	}
	started := time.Now()
	if _, err = s.Pool.Exec(ctx, `UPDATE voice_matching_status SET running_since=now() WHERE id`); err != nil {
		return true, stats, err
	}
	if full {
		stats, err = s.rematchLocked(ctx, wholeLibrary)
	} else {
		for _, clip := range clips {
			var one matchStats
			if one, err = s.rematchLocked(ctx, clip); err != nil {
				break
			}
			stats.add(one)
		}
	}
	if err != nil {
		// A stop is no failure: the requests stay and the next start resumes them.
		_, _ = s.Pool.Exec(context.WithoutCancel(ctx), `UPDATE voice_matching_status SET running_since=NULL,last_failed_at=CASE WHEN $1 THEN last_failed_at ELSE now() END WHERE id`, ctx.Err() != nil)
		return true, stats, err
	}
	took := time.Since(started)
	if _, err = s.Pool.Exec(ctx, `WITH done AS (DELETE FROM voice_matching_requests WHERE id=ANY($1))
UPDATE voice_matching_status SET running_since=NULL,last_finished_at=now(),last_duration_ms=$2,last_requests=$3,last_failed_at=NULL WHERE id`, ids, took.Milliseconds(), len(ids)); err != nil {
		return true, stats, err
	}
	s.Log.Info("voice_matching_pass", "requests", len(ids), "full", full, "clips", len(clips), "duration_ms", took.Milliseconds(), "evaluated", stats.Evaluated)
	return true, stats, nil
}

// waitingMatching lists the waiting requests: their ids (deleted once matched), the distinct clips asked for, and
// whether one asks for the whole library. Only the requests read here are deleted by the pass: one committed
// later, even with a smaller id, waits for the next pass.
func waitingMatching(ctx context.Context, s *Server) (ids []int64, clips []int64, wholeLibrary bool, err error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,clip_id FROM voice_matching_requests ORDER BY id`)
	if err != nil {
		return nil, nil, false, err
	}
	defer rows.Close()
	seen := map[int64]bool{}
	for rows.Next() {
		var id int64
		var clip *int64
		if err = rows.Scan(&id, &clip); err != nil {
			return nil, nil, false, err
		}
		ids = append(ids, id)
		switch {
		case clip == nil:
			wholeLibrary = true
		case !seen[*clip]:
			seen[*clip] = true
			clips = append(clips, *clip)
		}
	}
	return ids, clips, wholeLibrary, rows.Err()
}

func (a *matchStats) add(b matchStats) {
	a.Evaluated += b.Evaluated
	a.Automatic += b.Automatic
	a.Suggestion += b.Suggestion
	a.Ambiguous += b.Ambiguous
	a.Unknown += b.Unknown
	a.Removed += b.Removed
}

// matchingStatus is what the interface shows: whether matching is in progress (requests waiting, or a pass
// running) and when the last pass ended, so a reader knows whether the suggestions shown are up to date.
type matchingStatus struct {
	Pending        bool       `json:"pending"`
	Running        bool       `json:"running"`
	RunningSince   *time.Time `json:"running_since"`
	LastFinishedAt *time.Time `json:"last_finished_at"`
	LastDurationMS *int64     `json:"last_duration_ms"`
	LastRequests   *int       `json:"last_requests"`
	LastFailedAt   *time.Time `json:"last_failed_at"`
}

func loadMatchingStatus(ctx context.Context, q pgQuerier) (matchingStatus, error) {
	var v matchingStatus
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM voice_matching_requests),running_since,last_finished_at,last_duration_ms,last_requests,last_failed_at FROM voice_matching_status WHERE id`).Scan(&v.Pending, &v.RunningSince, &v.LastFinishedAt, &v.LastDurationMS, &v.LastRequests, &v.LastFailedAt)
	v.Running = v.RunningSince != nil
	return v, err
}

func (s *Server) getVoiceMatchingStatus(w http.ResponseWriter, r *http.Request) {
	v, err := loadMatchingStatus(r.Context(), s.Pool)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, v)
}

// voiceMatchingStatusEvents streams the matching status to every signed-in user: a snapshot, then one per
// request or pass. The session is read again before every snapshot; a closed session ends the stream.
func (s *Server) voiceMatchingStatusEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(sessionKey{}).(session).User.ID
	authorize := func(ctx context.Context) error {
		ss, err := s.findSession(r.WithContext(ctx))
		switch {
		case errors.Is(err, pgx.ErrNoRows), errors.Is(err, http.ErrNoCookie):
			return events.ErrForbidden
		case err != nil:
			return err
		case ss.User.ID != userID:
			return events.ErrForbidden
		}
		return nil
	}
	s.Events.Stream(w, r, events.Topic{
		Name:      "voice_matching",
		Authorize: authorize,
		Snapshot:  func(ctx context.Context) (any, error) { return loadMatchingStatus(ctx, s.Pool) },
	})
}
