package httpapi

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Conservation of the dispositions and refusals at a new diarization (ADR 0031). The new
// diarization replaces the local speakers of an audio stream; each disposition (ignored or
// random) and each refusal of an old voice goes to the new voice whose turns cover the
// largest share of the old voice's speaking time, when that share is at least half and the
// choice is unambiguous. Two old voices that land on the same new voice with different
// dispositions (no disposition included) keep none: ignoring the merged voice would hide the
// other one. Refusals never conflict: they are merged. Everything else is dropped. Every
// carry and every drop is journaled by the system with its reason, before the old voices
// are deleted.

// systemAuthor writes what the system decides, with no author id.
var systemAuthor = authorRef{Username: "system"}

// carryMinShare is the share of an old voice's speaking time a new voice must cover.
const carryMinShare = 0.5

type carriedRefusal struct {
	PersonID       int64
	PersonName     string
	AuthorID       *int64
	AuthorUsername string
}

type oldVoice struct {
	ID          int64
	Label       string
	Disposition string
	Refusals    []carriedRefusal
	Target      int     // ordinal of the new voice, -1 when none
	Share       float64 // share of its speaking time covered by the best new voice
	Tied        bool    // two new voices cover the same best share
}

// voiceCarry is what a new diarization keeps of the voices it replaces, between the moment
// the old voices are read and the moment the new ones exist.
type voiceCarry struct {
	clipID int64
	voices []oldVoice
	// conflict marks the new voices on which old voices with different dispositions land.
	conflict map[int]bool
}

// planVoiceCarry reads the old voices of the stream that the diarization deletes (diarized,
// without human validation: validated ones are archived with their own state) and maps each
// one to a new voice by the overlap of their turns.
func planVoiceCarry(ctx context.Context, tx pgx.Tx, clipID int64, index int, result diarizationResult) (*voiceCarry, error) {
	plan := &voiceCarry{clipID: clipID, conflict: map[int]bool{}}
	rows, err := tx.Query(ctx, `SELECT ls.id,ls.label,COALESCE(d.disposition,'') FROM local_speakers ls LEFT JOIN local_speaker_dispositions d ON d.local_speaker_id=ls.id
WHERE ls.clip_id=$1 AND ls.audio_stream_index=$2 AND ls.kind='diarized'
  AND NOT EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND a.validation_state='validated')
ORDER BY ls.ordinal,ls.id`, clipID, index)
	if err != nil {
		return nil, err
	}
	byID := map[int64]int{}
	for rows.Next() {
		v := oldVoice{Target: -1}
		if err = rows.Scan(&v.ID, &v.Label, &v.Disposition); err != nil {
			rows.Close()
			return nil, err
		}
		byID[v.ID] = len(plan.voices)
		plan.voices = append(plan.voices, v)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(plan.voices) == 0 {
		return plan, nil
	}
	ids := make([]int64, len(plan.voices))
	for i, v := range plan.voices {
		ids[i] = v.ID
	}
	refusals, err := tx.Query(ctx, `SELECT vr.local_speaker_id,vr.person_id,COALESCE(p.display_name,''),vr.author_id,vr.author_username FROM voice_refusals vr LEFT JOIN persons p ON p.id=vr.person_id WHERE vr.local_speaker_id=ANY($1) ORDER BY vr.local_speaker_id,vr.person_id`, ids)
	if err != nil {
		return nil, err
	}
	for refusals.Next() {
		var speakerID int64
		var r carriedRefusal
		if err = refusals.Scan(&speakerID, &r.PersonID, &r.PersonName, &r.AuthorID, &r.AuthorUsername); err != nil {
			refusals.Close()
			return nil, err
		}
		at := byID[speakerID]
		plan.voices[at].Refusals = append(plan.voices[at].Refusals, r)
	}
	refusals.Close()
	if err = refusals.Err(); err != nil {
		return nil, err
	}
	turns, err := tx.Query(ctx, `SELECT local_speaker_id,start_seconds,end_seconds FROM speaker_turns WHERE local_speaker_id=ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	oldTurns := map[int64][][2]float64{}
	for turns.Next() {
		var speakerID int64
		var start, end float64
		if err = turns.Scan(&speakerID, &start, &end); err != nil {
			turns.Close()
			return nil, err
		}
		oldTurns[speakerID] = append(oldTurns[speakerID], [2]float64{start, end})
	}
	turns.Close()
	if err = turns.Err(); err != nil {
		return nil, err
	}
	var newTurns [][][2]float64
	if result.Speakers != nil {
		for _, sp := range *result.Speakers {
			turnsOfNew := make([][2]float64, 0, len(*sp.Turns))
			for _, t := range *sp.Turns {
				turnsOfNew = append(turnsOfNew, [2]float64{*t.Start, *t.End})
			}
			newTurns = append(newTurns, turnsOfNew)
		}
	}
	for i := range plan.voices {
		v := &plan.voices[i]
		v.Target, v.Share, v.Tied = carryTarget(oldTurns[v.ID], newTurns)
	}
	plan.conflict = carryConflicts(plan.voices)
	return plan, nil
}

// carryTarget maps an old voice, by its turns, to the new voice (by ordinal) whose turns cover
// the largest share of its speaking time; target is -1 when that share is below carryMinShare
// or when two new voices cover the same best share (tied).
func carryTarget(old [][2]float64, news [][][2]float64) (target int, share float64, tied bool) {
	total := 0.0
	for _, t := range old {
		total += t[1] - t[0]
	}
	if total <= 0 {
		return -1, 0, false
	}
	best, second, bestOrdinal := 0.0, 0.0, -1
	for ordinal, turnsOfNew := range news {
		covered := 0.0
		for _, o := range old {
			for _, n := range turnsOfNew {
				if overlap := min(o[1], n[1]) - max(o[0], n[0]); overlap > 0 {
					covered += overlap
				}
			}
		}
		if covered > best {
			best, second, bestOrdinal = covered, best, ordinal
		} else if covered > second {
			second = covered
		}
	}
	share, tied = best/total, best > 0 && second == best
	if share >= carryMinShare && !tied {
		return bestOrdinal, share, false
	}
	return -1, share, tied
}

// carryConflicts marks the new voices on which old voices with different dispositions (no
// disposition included) land: none of those dispositions is carried.
func carryConflicts(voices []oldVoice) map[int]bool {
	dispositions := map[int]map[string]bool{}
	for _, v := range voices {
		if v.Target < 0 {
			continue
		}
		if dispositions[v.Target] == nil {
			dispositions[v.Target] = map[string]bool{}
		}
		dispositions[v.Target][v.Disposition] = true
	}
	conflict := map[int]bool{}
	for target, seen := range dispositions {
		conflict[target] = len(seen) > 1
	}
	return conflict
}

// carriedReason is the journal sentence of a carry: what was done ("Reportée sur", "Reprise
// de", "Repris de"), the voice and its diarization, and the share of speaking time.
func carriedReason(done, label, diarization string, share float64) string {
	return fmt.Sprintf("%s la voix %s de la %s (%s de son temps de parole)", done, label, diarization, percent(share))
}

func percent(share float64) string {
	return fmt.Sprintf("%.0f %%", share*100)
}

// dropReason says why a voice keeps nothing; "" when it is carried.
func (c *voiceCarry) dropReason(v oldVoice, conflicts bool, newLabels []string) string {
	switch {
	case v.Tied && v.Share >= carryMinShare:
		return fmt.Sprintf("deux voix de la nouvelle diarisation couvrent autant son temps de parole (%s chacune)", percent(v.Share))
	case v.Target < 0:
		return fmt.Sprintf("aucune voix de la nouvelle diarisation n'en couvre plus de %s, moins de la moitié de son temps de parole", percent(v.Share))
	case conflicts && c.conflict[v.Target]:
		return fmt.Sprintf("la voix %s de la nouvelle diarisation couvre aussi une voix de disposition différente (conflit)", newLabels[v.Target])
	}
	return ""
}

// release journals, on the old voices, what happens to their dispositions and refusals and
// removes them, so that the deletion of the old voices drops nothing silently.
func (c *voiceCarry) release(ctx context.Context, tx pgx.Tx, newLabels []string) error {
	for _, v := range c.voices {
		if v.Disposition != "" {
			reason := "Abandonnée à la nouvelle diarisation : " + c.dropReason(v, true, newLabels)
			if v.Target >= 0 && !c.conflict[v.Target] {
				reason = carriedReason("Reportée sur", newLabels[v.Target], "nouvelle diarisation", v.Share)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO voice_disposition_events(local_speaker_id,clip_id,old_disposition,new_disposition,author_id,author_username,reason) VALUES($1,$2,$3,'',NULL,'system',$4)`, v.ID, c.clipID, v.Disposition, reason); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM local_speaker_dispositions WHERE local_speaker_id=$1`, v.ID); err != nil {
				return err
			}
		}
		if why := c.dropReason(v, false, newLabels); why != "" {
			for _, r := range v.Refusals {
				if err := journalRefusal(ctx, tx, v.ID, c.clipID, r.PersonID, r.PersonName, "drop", systemAuthor, "Abandonné à la nouvelle diarisation : "+why); err != nil {
					return err
				}
			}
		}
		if _, err := tx.Exec(ctx, `DELETE FROM voice_refusals WHERE local_speaker_id=$1`, v.ID); err != nil {
			return err
		}
	}
	return nil
}

// apply gives the carried dispositions and refusals to the new voices (ids by ordinal).
func (c *voiceCarry) apply(ctx context.Context, tx pgx.Tx, newIDs []int64) error {
	for _, v := range c.voices {
		if v.Target < 0 {
			continue
		}
		target := newIDs[v.Target]
		if v.Disposition != "" && !c.conflict[v.Target] {
			if _, err := tx.Exec(ctx, `INSERT INTO local_speaker_dispositions(local_speaker_id,disposition,author_id,author_username) VALUES($1,$2,NULL,'system') ON CONFLICT (local_speaker_id) DO NOTHING`, target, v.Disposition); err != nil {
				return err
			}
			reason := carriedReason("Reprise de", v.Label, "diarisation précédente", v.Share)
			if _, err := tx.Exec(ctx, `INSERT INTO voice_disposition_events(local_speaker_id,clip_id,old_disposition,new_disposition,author_id,author_username,reason)
SELECT $1,$2,'',$3,NULL,'system',$4 WHERE NOT EXISTS (SELECT 1 FROM voice_disposition_events WHERE local_speaker_id=$1)`, target, c.clipID, v.Disposition, reason); err != nil {
				return err
			}
		}
		for _, r := range v.Refusals {
			if _, err := tx.Exec(ctx, `INSERT INTO voice_refusals(local_speaker_id,person_id,author_id,author_username) VALUES($1,$2,$3,$4) ON CONFLICT (local_speaker_id,person_id) DO NOTHING`, target, r.PersonID, r.AuthorID, r.AuthorUsername); err != nil {
				return err
			}
			reason := carriedReason("Repris de", v.Label, "diarisation précédente", v.Share)
			if err := journalRefusal(ctx, tx, target, c.clipID, r.PersonID, r.PersonName, "carry", systemAuthor, reason); err != nil {
				return err
			}
		}
	}
	return nil
}
