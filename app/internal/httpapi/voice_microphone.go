package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Dedicated microphone track (phase 5). An administrator designates,
// per source, a detected audio track (by stream index and/or exact title, never a
// built-in title) and its owner; the track is attributed to that Person directly,
// without diarization. A clip can replace the choice. Every outcome other than
// "designated" is reported explicitly: a designation that matches nothing, several
// tracks, or conflicting sources is never dropped silently.

// registerMicrophoneRoutes declares the microphone track routes.
func (s *Server) registerMicrophoneRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/admin/sources/{id}/microphone-track", s.protected(true, http.HandlerFunc(s.getSourceMicrophone)))
	mux.Handle("PUT /api/admin/sources/{id}/microphone-track", s.protected(true, http.HandlerFunc(s.putSourceMicrophone)))
	mux.Handle("DELETE /api/admin/sources/{id}/microphone-track", s.protected(true, http.HandlerFunc(s.deleteSourceMicrophone)))
	mux.Handle("GET /api/clips/{id}/microphone-track", s.protected(false, http.HandlerFunc(s.getClipMicrophone)))
	mux.Handle("PUT /api/admin/clips/{id}/microphone-track", s.protected(true, http.HandlerFunc(s.putClipMicrophone)))
	mux.Handle("DELETE /api/admin/clips/{id}/microphone-track", s.protected(true, http.HandlerFunc(s.deleteClipMicrophone)))
}

// querier is satisfied by the pool and by a transaction.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const (
	microphoneLabel     = "Piste micro"
	microphoneOrigin    = "microphone_track"
	microphoneNoTrack   = "none"
	microphoneDesigned  = "designated"
	microphoneDisabled  = "disabled"
	microphoneMissing   = "track_not_found"
	microphoneAmbiguous = "ambiguous_track"
	microphoneConflict  = "conflicting_sources"
	microphoneBlocked   = "blocked_by_validation"
	// The designated owner deleted their voice profile: nothing is attributed to them.
	microphoneOptedOut = "owner_profile_deleted"
)

type microphoneDesignation struct {
	Status     string
	Source     string
	Index      *int
	Person     *personRef
	Candidates []int
}

type microphoneRuleRow struct {
	Index  *int
	Title  *string
	Person personRef
}

func matchTracks(tracks []audioTrack, index *int, title *string) []int {
	matches := []int{}
	for _, t := range tracks {
		if t.Type != "audio" || (index != nil && t.Index != *index) || (title != nil && !strings.EqualFold(t.Title, *title)) {
			continue
		}
		matches = append(matches, t.Index)
	}
	return matches
}

// resolveMicrophone decides which track of the clip is the microphone, from the
// clip replacement first, then from the rules of the sources the clip belongs to.
func (s *Server) resolveMicrophone(ctx context.Context, q querier, clipID int64, tracks []audioTrack) (microphoneDesignation, error) {
	var disabled bool
	var personID *int64
	var personName *string
	var overrideIndex *int
	err := q.QueryRow(ctx, `SELECT o.disabled,o.audio_stream_index,o.person_id,p.display_name FROM clip_microphone_overrides o LEFT JOIN persons p ON p.id=o.person_id WHERE o.clip_id=$1`, clipID).Scan(&disabled, &overrideIndex, &personID, &personName)
	var d microphoneDesignation
	switch {
	case err == nil && disabled:
		return microphoneDesignation{Status: microphoneDisabled, Source: "clip_override", Candidates: []int{}}, nil
	case err == nil:
		d = microphoneDesignation{Status: microphoneDesigned, Source: "clip_override", Index: overrideIndex, Person: &personRef{ID: *personID, DisplayName: *personName}, Candidates: []int{}}
		if !hasAudioTrack(tracks, *overrideIndex) {
			d.Status = microphoneMissing
		}
	case errors.Is(err, pgx.ErrNoRows):
		rules, rulesErr := s.microphoneRules(ctx, q, clipID)
		if rulesErr != nil {
			return d, rulesErr
		}
		d = resolveMicrophoneRules(tracks, rules)
	default:
		return d, err
	}
	if d.Status == microphoneDesigned {
		var optedOut bool
		if err = q.QueryRow(ctx, `SELECT voice_profile_deleted_at IS NOT NULL FROM persons WHERE id=$1`, d.Person.ID).Scan(&optedOut); err != nil {
			return d, err
		}
		if optedOut {
			d.Status = microphoneOptedOut
			return d, nil
		}
		var blocked bool
		if err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM speaker_attributions a JOIN local_speakers ls ON ls.id=a.local_speaker_id WHERE ls.clip_id=$1 AND ls.audio_stream_index=$2 AND ls.kind='diarized' AND a.validation_state='validated')`, clipID, *d.Index).Scan(&blocked); err != nil {
			return d, err
		}
		if blocked {
			d.Status = microphoneBlocked
		}
	}
	return d, nil
}

func (s *Server) microphoneRules(ctx context.Context, q querier, clipID int64) ([]microphoneRuleRow, error) {
	rows, err := q.Query(ctx, `SELECT smt.audio_stream_index,smt.title,p.id,p.display_name FROM source_microphone_tracks smt JOIN persons p ON p.id=smt.person_id WHERE smt.source_id IN (SELECT source_id FROM source_files WHERE clip_id=$1 AND status='ready') ORDER BY smt.source_id`, clipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []microphoneRuleRow
	for rows.Next() {
		var r microphoneRuleRow
		if err = rows.Scan(&r.Index, &r.Title, &r.Person.ID, &r.Person.DisplayName); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func resolveMicrophoneRules(tracks []audioTrack, rules []microphoneRuleRow) microphoneDesignation {
	d := microphoneDesignation{Status: microphoneNoTrack, Candidates: []int{}}
	if len(rules) == 0 {
		return d
	}
	d.Source = "source_rule"
	var chosen *microphoneDesignation
	for _, rule := range rules {
		person := rule.Person
		matches := matchTracks(tracks, rule.Index, rule.Title)
		switch len(matches) {
		case 0:
			return microphoneDesignation{Status: microphoneMissing, Source: "source_rule", Index: rule.Index, Person: &person, Candidates: []int{}}
		case 1:
			index := matches[0]
			if chosen != nil && (*chosen.Index != index || chosen.Person.ID != person.ID) {
				return microphoneDesignation{Status: microphoneConflict, Source: "source_rule", Candidates: []int{}}
			}
			chosen = &microphoneDesignation{Status: microphoneDesigned, Source: "source_rule", Index: &index, Person: &person, Candidates: []int{}}
		default:
			return microphoneDesignation{Status: microphoneAmbiguous, Source: "source_rule", Person: &person, Candidates: matches}
		}
	}
	return *chosen
}

func clipTracks(ctx context.Context, q querier, clipID int64) ([]audioTrack, float64, error) {
	var raw []byte
	var duration float64
	if err := q.QueryRow(ctx, `SELECT COALESCE(tracks,'[]'::jsonb),COALESCE(duration_seconds,0) FROM clips WHERE id=$1`, clipID).Scan(&raw, &duration); err != nil {
		return nil, 0, err
	}
	var tracks []audioTrack
	if err := json.Unmarshal(raw, &tracks); err != nil {
		return nil, 0, err
	}
	return tracks, duration, nil
}

// applyMicrophone makes the clip's local speakers match its designation, in one
// transaction under the clip lock. It is idempotent: a designated track holds one
// local speaker without diarization job, carrying an automatic attribution from the
// microphone track. A human validation of that speaker is never removed.
func (s *Server) applyMicrophone(ctx context.Context, clipID int64, author authorRef) (microphoneDesignation, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return microphoneDesignation{}, err
	}
	defer tx.Rollback(ctx)
	var lockedID int64
	if err = tx.QueryRow(ctx, `SELECT id FROM clips WHERE id=$1 FOR UPDATE`, clipID).Scan(&lockedID); err != nil {
		return microphoneDesignation{}, err
	}
	tracks, duration, err := clipTracks(ctx, tx, clipID)
	if err != nil {
		return microphoneDesignation{}, err
	}
	d, err := s.resolveMicrophone(ctx, tx, clipID, tracks)
	if err != nil {
		return d, err
	}
	var speakerID int64
	existing := true
	if err = tx.QueryRow(ctx, `SELECT id FROM local_speakers WHERE clip_id=$1 AND kind='microphone'`, clipID).Scan(&speakerID); errors.Is(err, pgx.ErrNoRows) {
		existing, err = false, nil
	} else if err != nil {
		return d, err
	}
	var oldPersonID *int64
	var oldName string
	var oldLevel DecisionLevel
	var attributionID *int64
	var oldOrigin string
	if existing {
		var id int64
		var origin string
		err = tx.QueryRow(ctx, `SELECT a.id,a.person_id,COALESCE(p.display_name,''),a.decision_level,a.origin FROM speaker_attributions a LEFT JOIN persons p ON p.id=a.person_id WHERE a.local_speaker_id=$1 AND a.validation_state='unvalidated' FOR UPDATE OF a`, speakerID).Scan(&id, &oldPersonID, &oldName, &oldLevel, &origin)
		if err == nil {
			attributionID, oldOrigin = &id, origin
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return d, err
		}
	}
	if oldLevel == "" {
		oldLevel = LevelUnknown
	}
	event := func(action string, newPerson *personRef, newLevel DecisionLevel) error {
		var newID *int64
		newName := ""
		if newPerson != nil {
			newID, newName = &newPerson.ID, newPerson.DisplayName
		}
		_, e := tx.Exec(ctx, `INSERT INTO attribution_events(local_speaker_id,clip_id,action,old_person_id,old_person_name,old_decision_level,new_person_id,new_person_name,new_decision_level,author_id,author_username) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, speakerID, clipID, action, oldPersonID, oldName, oldLevel, newID, newName, newLevel, author.ID, author.Username)
		return e
	}
	if d.Status == microphoneDesigned {
		// The designated stream is not diarized: a previous, unvalidated diarization
		// of it is replaced by the designation.
		if err = dropSpeakerOverrides(ctx, tx, droppedByMicrophone, `ls.clip_id=$1 AND ls.audio_stream_index=$2 AND ls.kind='diarized'`, clipID, *d.Index); err != nil {
			return d, err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM local_speakers WHERE clip_id=$1 AND audio_stream_index=$2 AND kind='diarized'`, clipID, *d.Index); err != nil {
			return d, err
		}
		if existing {
			if _, err = tx.Exec(ctx, `UPDATE local_speakers SET audio_stream_index=$2,speaking_seconds=$3 WHERE id=$1`, speakerID, *d.Index, duration); err != nil {
				return d, err
			}
			if _, err = tx.Exec(ctx, `DELETE FROM speaker_turns WHERE local_speaker_id=$1`, speakerID); err != nil {
				return d, err
			}
		} else if err = tx.QueryRow(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,kind,label,ordinal,speaking_seconds,turn_count) VALUES($1,$2,'microphone',$3,0,$4,1) RETURNING id`, clipID, *d.Index, microphoneLabel, duration).Scan(&speakerID); err != nil {
			return d, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO speaker_turns(local_speaker_id,clip_id,turn_index,start_seconds,end_seconds) VALUES($1,$2,0,0,$3)`, speakerID, clipID, duration); err != nil {
			return d, err
		}
		switch {
		case attributionID == nil:
			if _, err = tx.Exec(ctx, `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,confidence,origin,author_id,author_username) VALUES($1,$2,'automatic','unvalidated',1,$3,$4,$5)`, speakerID, d.Person.ID, microphoneOrigin, author.ID, author.Username); err != nil {
				return d, err
			}
			err = event("microphone_assign", d.Person, LevelAutomatic)
		case oldPersonID == nil || *oldPersonID != d.Person.ID || oldOrigin != microphoneOrigin:
			if _, err = tx.Exec(ctx, `UPDATE speaker_attributions SET person_id=$2,decision_level='automatic',confidence=1,origin=$3,author_id=$4,author_username=$5,revision=revision+1,decided_at=now() WHERE id=$1`, *attributionID, d.Person.ID, microphoneOrigin, author.ID, author.Username); err != nil {
				return d, err
			}
			err = event("microphone_assign", d.Person, LevelAutomatic)
		}
		if err != nil {
			return d, err
		}
	} else if existing {
		if attributionID != nil {
			if _, err = tx.Exec(ctx, `DELETE FROM speaker_attributions WHERE id=$1`, *attributionID); err != nil {
				return d, err
			}
		}
		var validated bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM speaker_attributions WHERE local_speaker_id=$1 AND validation_state='validated')`, speakerID).Scan(&validated); err != nil {
			return d, err
		}
		if validated {
			// A human validation is never lost: the speaker stays, only the machine
			// attribution of the designation is withdrawn.
			if attributionID != nil {
				if err = event("microphone_unassign", nil, LevelUnknown); err != nil {
					return d, err
				}
			}
		} else {
			if err = dropSpeakerOverrides(ctx, tx, droppedByMicrophoneWithdrawal, `ls.id=$1`, speakerID); err != nil {
				return d, err
			}
			if _, err = tx.Exec(ctx, `DELETE FROM local_speakers WHERE id=$1`, speakerID); err != nil {
				return d, err
			}
		}
	}
	return d, tx.Commit(ctx)
}

// microphoneSnapshot is the immutable form of a designation kept in the history.
type microphoneSnapshot struct {
	AudioStreamIndex *int       `json:"audio_stream_index,omitempty"`
	Title            *string    `json:"title,omitempty"`
	Disabled         bool       `json:"disabled,omitempty"`
	Person           *personRef `json:"person,omitempty"`
}

func snapshotJSON(snap *microphoneSnapshot) any {
	if snap == nil {
		return nil
	}
	raw, _ := json.Marshal(snap)
	return raw
}

func (s *Server) authorOf(r *http.Request) authorRef {
	ss := r.Context().Value(sessionKey{}).(session)
	id := ss.User.ID
	return authorRef{ID: &id, Username: ss.User.Username}
}

func (s *Server) personName(ctx context.Context, q querier, id int64) (string, error) {
	var name string
	err := q.QueryRow(ctx, `SELECT display_name FROM persons WHERE id=$1`, id).Scan(&name)
	return name, err
}

type microphoneRuleInput struct {
	AudioStreamIndex *int    `json:"audio_stream_index"`
	Title            *string `json:"title"`
	PersonID         int64   `json:"person_id"`
}

type microphoneClipInput struct {
	Disabled         bool   `json:"disabled"`
	AudioStreamIndex *int   `json:"audio_stream_index"`
	PersonID         *int64 `json:"person_id"`
}

type detectedTrack struct {
	AudioStreamIndex int    `json:"audio_stream_index"`
	Title            string `json:"title"`
	Language         string `json:"language"`
	Channels         int    `json:"channels"`
	ChannelLayout    string `json:"channel_layout"`
	ClipCount        int    `json:"clip_count"`
}

type microphoneSourceRule struct {
	AudioStreamIndex *int      `json:"audio_stream_index"`
	Title            *string   `json:"title"`
	Person           personRef `json:"person"`
	Version          int64     `json:"version"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (s *Server) sourceExists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clip_sources WHERE id=$1)`, id).Scan(&exists)
	return exists, err
}

// sourceMicrophoneView lists the rule of a source and the audio tracks really
// detected on its clips (index, title, language, layout and how many clips).
func (s *Server) sourceMicrophoneView(ctx context.Context, id int64, statusCounts map[string]int) (map[string]any, error) {
	var rule *microphoneSourceRule
	var r microphoneSourceRule
	err := s.Pool.QueryRow(ctx, `SELECT smt.audio_stream_index,smt.title,p.id,p.display_name,smt.version,smt.updated_at FROM source_microphone_tracks smt JOIN persons p ON p.id=smt.person_id WHERE smt.source_id=$1`, id).Scan(&r.AudioStreamIndex, &r.Title, &r.Person.ID, &r.Person.DisplayName, &r.Version, &r.UpdatedAt)
	if err == nil {
		rule = &r
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT (t->>'index')::int,COALESCE(t->>'title',''),COALESCE(t->>'language',''),COALESCE((t->>'channels')::int,0),COALESCE(t->>'channel_layout',''),count(DISTINCT c.id) FROM source_files sf JOIN clips c ON c.id=sf.clip_id CROSS JOIN LATERAL jsonb_array_elements(COALESCE(c.tracks,'[]'::jsonb)) t WHERE sf.source_id=$1 AND sf.status='ready' AND t->>'type'='audio' GROUP BY 1,2,3,4,5 ORDER BY 1,2,3`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	detected := []detectedTrack{}
	for rows.Next() {
		var t detectedTrack
		if err = rows.Scan(&t.AudioStreamIndex, &t.Title, &t.Language, &t.Channels, &t.ChannelLayout, &t.ClipCount); err != nil {
			return nil, err
		}
		detected = append(detected, t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if statusCounts == nil {
		statusCounts = map[string]int{}
	}
	return map[string]any{"source_id": id, "rule": rule, "detected_tracks": detected, "status_counts": statusCounts}, nil
}

// applySource re-applies the designation to every ready clip of the source and
// counts the outcomes, so nothing that failed to match goes unnoticed.
func (s *Server) applySource(ctx context.Context, sourceID int64, author authorRef) (map[string]int, error) {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT clip_id FROM source_files WHERE source_id=$1 AND clip_id IS NOT NULL AND status='ready' ORDER BY clip_id`, sourceID)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, id := range ids {
		d, applyErr := s.applyMicrophone(ctx, id, author)
		if applyErr != nil {
			return nil, applyErr
		}
		counts[d.Status]++
	}
	return counts, nil
}

func (s *Server) getSourceMicrophone(w http.ResponseWriter, r *http.Request) {
	id, ok := sourceID(w, r)
	if !ok {
		return
	}
	if exists, err := s.sourceExists(r.Context(), id); err != nil {
		s.internal(w, err)
		return
	} else if !exists {
		fail(w, http.StatusNotFound, "source_not_found")
		return
	}
	view, err := s.sourceMicrophoneView(r.Context(), id, nil)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

func (s *Server) putSourceMicrophone(w http.ResponseWriter, r *http.Request) {
	id, ok := sourceID(w, r)
	if !ok {
		return
	}
	var input microphoneRuleInput
	if !decode(w, r, &input) {
		return
	}
	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" || len(title) > 200 {
			fail(w, http.StatusBadRequest, "invalid_microphone_rule")
			return
		}
		input.Title = &title
	}
	if (input.AudioStreamIndex == nil && input.Title == nil) || (input.AudioStreamIndex != nil && *input.AudioStreamIndex < 0) || input.PersonID < 1 {
		fail(w, http.StatusBadRequest, "invalid_microphone_rule")
		return
	}
	if exists, err := s.sourceExists(r.Context(), id); err != nil {
		s.internal(w, err)
		return
	} else if !exists {
		fail(w, http.StatusNotFound, "source_not_found")
		return
	}
	name, err := s.personName(r.Context(), s.Pool, input.PersonID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "person_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	author := s.authorOf(r)
	ctx := r.Context()
	if err = s.replaceSourceRule(ctx, id, &input, name, author); err != nil {
		s.internal(w, err)
		return
	}
	counts, err := s.applySource(ctx, id, author)
	if err != nil {
		s.internal(w, err)
		return
	}
	view, err := s.sourceMicrophoneView(ctx, id, counts)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

// replaceSourceRule stores the rule (nil removes it) and its history entry in one
// transaction. Saving an identical rule changes nothing and records nothing.
func (s *Server) replaceSourceRule(ctx context.Context, id int64, input *microphoneRuleInput, name string, author authorRef) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var old microphoneSnapshot
	var oldPerson personRef
	var hadOld bool
	err = tx.QueryRow(ctx, `SELECT smt.audio_stream_index,smt.title,p.id,p.display_name FROM source_microphone_tracks smt JOIN persons p ON p.id=smt.person_id WHERE smt.source_id=$1 FOR UPDATE OF smt`, id).Scan(&old.AudioStreamIndex, &old.Title, &oldPerson.ID, &oldPerson.DisplayName)
	if err == nil {
		hadOld, old.Person = true, &oldPerson
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var oldSnap *microphoneSnapshot
	if hadOld {
		oldSnap = &old
	}
	if input == nil {
		if !hadOld {
			return nil
		}
		if _, err = tx.Exec(ctx, `DELETE FROM source_microphone_tracks WHERE source_id=$1`, id); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO microphone_track_events(scope,source_id,action,old_rule,new_rule,author_id,author_username) VALUES('source',$1,'remove',$2,NULL,$3,$4)`, id, snapshotJSON(oldSnap), author.ID, author.Username); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	next := &microphoneSnapshot{AudioStreamIndex: input.AudioStreamIndex, Title: input.Title, Person: &personRef{ID: input.PersonID, DisplayName: name}}
	if hadOld && sameSnapshot(oldSnap, next) {
		return nil
	}
	if _, err = tx.Exec(ctx, `INSERT INTO source_microphone_tracks(source_id,audio_stream_index,title,person_id,updated_by) VALUES($1,$2,$3,$4,$5) ON CONFLICT(source_id) DO UPDATE SET audio_stream_index=EXCLUDED.audio_stream_index,title=EXCLUDED.title,person_id=EXCLUDED.person_id,version=source_microphone_tracks.version+1,updated_by=EXCLUDED.updated_by,updated_at=now()`, id, input.AudioStreamIndex, input.Title, input.PersonID, author.ID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO microphone_track_events(scope,source_id,action,old_rule,new_rule,author_id,author_username) VALUES('source',$1,'set',$2,$3,$4,$5)`, id, snapshotJSON(oldSnap), snapshotJSON(next), author.ID, author.Username); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func sameSnapshot(a, b *microphoneSnapshot) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return string(ja) == string(jb)
}

func (s *Server) deleteSourceMicrophone(w http.ResponseWriter, r *http.Request) {
	id, ok := sourceID(w, r)
	if !ok {
		return
	}
	if exists, err := s.sourceExists(r.Context(), id); err != nil {
		s.internal(w, err)
		return
	} else if !exists {
		fail(w, http.StatusNotFound, "source_not_found")
		return
	}
	author := s.authorOf(r)
	if err := s.replaceSourceRule(r.Context(), id, nil, "", author); err != nil {
		s.internal(w, err)
		return
	}
	counts, err := s.applySource(r.Context(), id, author)
	if err != nil {
		s.internal(w, err)
		return
	}
	view, err := s.sourceMicrophoneView(r.Context(), id, counts)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

type microphoneEventView struct {
	ID        int64           `json:"id"`
	Scope     string          `json:"scope"`
	SourceID  *int64          `json:"source_id"`
	Action    string          `json:"action"`
	OldRule   json.RawMessage `json:"old_rule"`
	NewRule   json.RawMessage `json:"new_rule"`
	Author    *authorRef      `json:"author"`
	CreatedAt time.Time       `json:"created_at"`
}

type microphoneTrackItem struct {
	Index         int    `json:"index"`
	Title         string `json:"title"`
	Language      string `json:"language"`
	Channels      int    `json:"channels"`
	ChannelLayout string `json:"channel_layout"`
	Role          string `json:"role"`
}

// clipMicrophoneView reports how the clip's audio tracks are treated: which one is
// the microphone (if any), why a designation did not apply, and the history.
func (s *Server) clipMicrophoneView(ctx context.Context, clipID int64) (map[string]any, error) {
	tracks, _, err := clipTracks(ctx, s.Pool, clipID)
	if err != nil {
		return nil, err
	}
	d, err := s.resolveMicrophone(ctx, s.Pool, clipID, tracks)
	if err != nil {
		return nil, err
	}
	var speakerID *int64
	var speakerIndex *int
	err = s.Pool.QueryRow(ctx, `SELECT id,audio_stream_index FROM local_speakers WHERE clip_id=$1 AND kind='microphone'`, clipID).Scan(&speakerID, &speakerIndex)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	applied := d.Status == microphoneDesigned && speakerID != nil && speakerIndex != nil && *speakerIndex == *d.Index
	items := []microphoneTrackItem{}
	for _, t := range tracks {
		if t.Type != "audio" {
			continue
		}
		role := "other"
		if (d.Status == microphoneDesigned || d.Status == microphoneBlocked || d.Status == microphoneOptedOut) && d.Index != nil && *d.Index == t.Index {
			role = "microphone"
		}
		for _, c := range d.Candidates {
			if c == t.Index {
				role = "candidate"
			}
		}
		items = append(items, microphoneTrackItem{Index: t.Index, Title: t.Title, Language: t.Language, Channels: t.Channels, ChannelLayout: t.ChannelLayout, Role: role})
	}
	history := []microphoneEventView{}
	rows, err := s.Pool.Query(ctx, `SELECT id,scope,source_id,action,old_rule,new_rule,author_id,author_username,created_at FROM microphone_track_events WHERE clip_id=$1 OR (scope='source' AND source_id IN (SELECT source_id FROM source_files WHERE clip_id=$1)) ORDER BY id DESC LIMIT 50`, clipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e microphoneEventView
		var authorID *int64
		var username string
		if err = rows.Scan(&e.ID, &e.Scope, &e.SourceID, &e.Action, &e.OldRule, &e.NewRule, &authorID, &username, &e.CreatedAt); err != nil {
			return nil, err
		}
		if authorID != nil || username != "" {
			e.Author = &authorRef{ID: authorID, Username: username}
		}
		history = append(history, e)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	var source any
	if d.Source != "" {
		source = d.Source
	}
	var index any
	if d.Index != nil {
		index = *d.Index
	}
	var person any
	if d.Person != nil {
		person = d.Person
	}
	return map[string]any{"clip_id": clipID, "status": d.Status, "selection_source": source, "audio_stream_index": index, "person": person, "candidates": d.Candidates, "local_speaker_id": speakerID, "applied": applied, "tracks": items, "history": history}, nil
}

func (s *Server) getClipMicrophone(w http.ResponseWriter, r *http.Request) {
	clip, ok := s.authorizedClip(w, r)
	if !ok {
		return
	}
	view, err := s.clipMicrophoneView(r.Context(), clip.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

func clipPathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := pathID(w, r, "invalid_clip_id")
	if !ok {
		return 0, false
	}
	return id, true
}

// putClipMicrophone replaces the source rule for one clip: another track and owner,
// or "no microphone track". The change is recorded in the history.
func (s *Server) putClipMicrophone(w http.ResponseWriter, r *http.Request) {
	id, ok := clipPathID(w, r)
	if !ok {
		return
	}
	var input microphoneClipInput
	if !decode(w, r, &input) {
		return
	}
	hasTrack := input.AudioStreamIndex != nil || input.PersonID != nil
	if (input.Disabled && hasTrack) || (!input.Disabled && (input.AudioStreamIndex == nil || input.PersonID == nil || *input.AudioStreamIndex < 0 || *input.PersonID < 1)) {
		fail(w, http.StatusBadRequest, "invalid_microphone_override")
		return
	}
	ctx := r.Context()
	tracks, _, err := clipTracks(ctx, s.Pool, id)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "clip_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	next := &microphoneSnapshot{Disabled: input.Disabled}
	if !input.Disabled {
		if !hasAudioTrack(tracks, *input.AudioStreamIndex) {
			fail(w, http.StatusBadRequest, "audio_stream_not_found")
			return
		}
		name, nameErr := s.personName(ctx, s.Pool, *input.PersonID)
		if errors.Is(nameErr, pgx.ErrNoRows) {
			fail(w, http.StatusNotFound, "person_not_found")
			return
		}
		if nameErr != nil {
			s.internal(w, nameErr)
			return
		}
		next.AudioStreamIndex, next.Person = input.AudioStreamIndex, &personRef{ID: *input.PersonID, DisplayName: name}
	}
	author := s.authorOf(r)
	if err = s.replaceClipOverride(ctx, id, next, author); err != nil {
		s.internal(w, err)
		return
	}
	s.finishClipMicrophone(w, r, id, author)
}

func (s *Server) deleteClipMicrophone(w http.ResponseWriter, r *http.Request) {
	id, ok := clipPathID(w, r)
	if !ok {
		return
	}
	if _, _, err := clipTracks(r.Context(), s.Pool, id); errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "clip_not_found")
		return
	} else if err != nil {
		s.internal(w, err)
		return
	}
	author := s.authorOf(r)
	if err := s.replaceClipOverride(r.Context(), id, nil, author); err != nil {
		s.internal(w, err)
		return
	}
	s.finishClipMicrophone(w, r, id, author)
}

func (s *Server) finishClipMicrophone(w http.ResponseWriter, r *http.Request, id int64, author authorRef) {
	if _, err := s.applyMicrophone(r.Context(), id, author); err != nil {
		s.internal(w, err)
		return
	}
	view, err := s.clipMicrophoneView(r.Context(), id)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

// replaceClipOverride stores the clip replacement (nil removes it) and its history
// entry in one transaction.
func (s *Server) replaceClipOverride(ctx context.Context, id int64, next *microphoneSnapshot, author authorRef) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var old microphoneSnapshot
	var oldPerson personRef
	var oldPersonID *int64
	var oldName *string
	hadOld := false
	err = tx.QueryRow(ctx, `SELECT o.disabled,o.audio_stream_index,o.person_id,p.display_name FROM clip_microphone_overrides o LEFT JOIN persons p ON p.id=o.person_id WHERE o.clip_id=$1 FOR UPDATE OF o`, id).Scan(&old.Disabled, &old.AudioStreamIndex, &oldPersonID, &oldName)
	if err == nil {
		hadOld = true
		if oldPersonID != nil && oldName != nil {
			oldPerson = personRef{ID: *oldPersonID, DisplayName: *oldName}
			old.Person = &oldPerson
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var oldSnap *microphoneSnapshot
	if hadOld {
		oldSnap = &old
	}
	if next == nil {
		if !hadOld {
			return nil
		}
		if _, err = tx.Exec(ctx, `DELETE FROM clip_microphone_overrides WHERE clip_id=$1`, id); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO microphone_track_events(scope,clip_id,action,old_rule,new_rule,author_id,author_username) VALUES('clip',$1,'remove',$2,NULL,$3,$4)`, id, snapshotJSON(oldSnap), author.ID, author.Username); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if hadOld && sameSnapshot(oldSnap, next) {
		return nil
	}
	var personID *int64
	if next.Person != nil {
		personID = &next.Person.ID
	}
	if _, err = tx.Exec(ctx, `INSERT INTO clip_microphone_overrides(clip_id,disabled,audio_stream_index,person_id,updated_by) VALUES($1,$2,$3,$4,$5) ON CONFLICT(clip_id) DO UPDATE SET disabled=EXCLUDED.disabled,audio_stream_index=EXCLUDED.audio_stream_index,person_id=EXCLUDED.person_id,version=clip_microphone_overrides.version+1,updated_by=EXCLUDED.updated_by,updated_at=now()`, id, next.Disabled, next.AudioStreamIndex, personID, author.ID); err != nil {
		return err
	}
	action := "set"
	if next.Disabled {
		action = "disable"
	}
	if _, err = tx.Exec(ctx, `INSERT INTO microphone_track_events(scope,clip_id,action,old_rule,new_rule,author_id,author_username) VALUES('clip',$1,$2,$3,$4,$5,$6)`, id, action, snapshotJSON(oldSnap), snapshotJSON(next), author.ID, author.Username); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// microphoneSpeakerViews lists the clip's microphone local speaker (on its own
// stream, whatever the diarized stream is) with its effective attribution.
func (s *Server) microphoneSpeakerViews(ctx context.Context, clipID int64, attributions map[int64]*attributionView) ([]localSpeakerView, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,label,ordinal,speaking_seconds,turn_count,kind,audio_stream_index FROM local_speakers WHERE clip_id=$1 AND kind='microphone' ORDER BY ordinal`, clipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []localSpeakerView{}
	for rows.Next() {
		sp := localSpeakerView{Fingerprints: []voiceFingerprintView{}}
		if err = rows.Scan(&sp.ID, &sp.Label, &sp.Ordinal, &sp.SpeakingSeconds, &sp.TurnCount, &sp.Kind, &sp.AudioStreamIndex); err != nil {
			return nil, err
		}
		sp.applyAttribution(attributions[sp.ID])
		out = append(out, sp)
	}
	return out, rows.Err()
}
