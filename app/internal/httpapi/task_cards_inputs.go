package httpapi

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

// cardInputs holds what the derivation of a task card reads about a clip beyond
// its tasks: the audio tracks, the track selection, the microphone designation
// and the diarized speakers. It is read once for all the clips of the board, one
// query per table, and the existing rules (selectAudio, decideMicrophone) are
// applied to it, so the board and the single clip decide the same way.
type cardInputs struct {
	names         map[int64]string
	tracks        map[int64][]audioTrack
	overrides     map[int64]audioOverride
	sourceRules   map[int64][]audioSourceRule
	micOverrides  map[int64]microphoneOverrideRow
	micRules      map[int64][]microphoneRuleRow
	optedOut      map[int64]bool
	blocked       map[[2]int64]bool
	diarizedCount map[[2]int64]int
}

// selection applies the track selection rule to a clip.
func (in *cardInputs) selection(clipID int64) audioSelection {
	var override *audioOverride
	if o, ok := in.overrides[clipID]; ok {
		override = &o
	}
	return selectAudio(in.tracks[clipID], override, in.sourceRules[clipID])
}

// microphone applies the microphone designation rule to a clip.
func (in *cardInputs) microphone(clipID int64) (microphoneDesignation, error) {
	var override *microphoneOverrideRow
	if o, ok := in.micOverrides[clipID]; ok {
		override = &o
	}
	return decideMicrophone(in.tracks[clipID], override, func() ([]microphoneRuleRow, error) {
		return in.micRules[clipID], nil
	}, func(personID int64) (bool, error) {
		return in.optedOut[personID], nil
	}, func(index int) (bool, error) {
		return in.blocked[[2]int64{clipID, int64(index)}], nil
	})
}

// scanAll runs one query and hands every row to scan.
func (s *Server) scanAll(ctx context.Context, sql string, args []any, scan func(pgx.Rows) error) error {
	rows, err := s.Pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err = scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Server) loadCardInputs(ctx context.Context, clipIDs []int64) (*cardInputs, error) {
	in := &cardInputs{
		names:         map[int64]string{},
		tracks:        map[int64][]audioTrack{},
		overrides:     map[int64]audioOverride{},
		sourceRules:   map[int64][]audioSourceRule{},
		micOverrides:  map[int64]microphoneOverrideRow{},
		micRules:      map[int64][]microphoneRuleRow{},
		optedOut:      map[int64]bool{},
		blocked:       map[[2]int64]bool{},
		diarizedCount: map[[2]int64]int{},
	}
	if len(clipIDs) == 0 {
		return in, nil
	}
	ids := []any{clipIDs}
	steps := []struct {
		sql  string
		scan func(pgx.Rows) error
	}{
		{`SELECT id,original_name,tracks FROM clips WHERE id=ANY($1)`, func(r pgx.Rows) error {
			var id int64
			var name string
			var raw []byte
			if err := r.Scan(&id, &name, &raw); err != nil {
				return err
			}
			in.names[id] = name
			var tracks []audioTrack
			if json.Unmarshal(raw, &tracks) == nil {
				in.tracks[id] = tracks
			}
			return nil
		}},
		{`SELECT clip_id,audio_stream_index,version FROM clip_audio_track_overrides WHERE clip_id=ANY($1)`, func(r pgx.Rows) error {
			var id int64
			var o audioOverride
			if err := r.Scan(&id, &o.index, &o.version); err != nil {
				return err
			}
			in.overrides[id] = o
			return nil
		}},
		{`SELECT sf.clip_id,sar.rule,sar.version FROM source_audio_track_rules sar JOIN source_files sf ON sf.source_id=sar.source_id WHERE sf.clip_id=ANY($1) AND sf.status='ready' ORDER BY sf.clip_id,sar.source_id`, func(r pgx.Rows) error {
			var id int64
			var rule audioSourceRule
			if err := r.Scan(&id, &rule.raw, &rule.version); err != nil {
				return err
			}
			in.sourceRules[id] = append(in.sourceRules[id], rule)
			return nil
		}},
		{`SELECT o.clip_id,o.disabled,o.audio_stream_index,o.person_id,p.display_name FROM clip_microphone_overrides o LEFT JOIN persons p ON p.id=o.person_id WHERE o.clip_id=ANY($1)`, func(r pgx.Rows) error {
			var id int64
			var o microphoneOverrideRow
			if err := r.Scan(&id, &o.disabled, &o.index, &o.personID, &o.personName); err != nil {
				return err
			}
			in.micOverrides[id] = o
			return nil
		}},
		{`SELECT sf.clip_id,smt.audio_stream_index,smt.title,p.id,p.display_name FROM source_microphone_tracks smt JOIN persons p ON p.id=smt.person_id JOIN (SELECT DISTINCT clip_id,source_id FROM source_files WHERE clip_id=ANY($1) AND status='ready') sf ON sf.source_id=smt.source_id ORDER BY sf.clip_id,smt.source_id`, func(r pgx.Rows) error {
			var id int64
			var rule microphoneRuleRow
			if err := r.Scan(&id, &rule.Index, &rule.Title, &rule.Person.ID, &rule.Person.DisplayName); err != nil {
				return err
			}
			in.micRules[id] = append(in.micRules[id], rule)
			return nil
		}},
		// Opt-outs, validated speakers and diarized speaker counts in one read.
		{`SELECT 'optout',id,0,0 FROM persons WHERE voice_profile_deleted_at IS NOT NULL
		UNION ALL SELECT 'blocked',ls.clip_id,COALESCE(ls.audio_stream_index,-1),0 FROM local_speakers ls WHERE ls.clip_id=ANY($1) AND ls.kind='diarized' AND EXISTS (SELECT 1 FROM speaker_attributions a WHERE a.local_speaker_id=ls.id AND a.validation_state='validated')
		UNION ALL SELECT 'diarized',clip_id,COALESCE(audio_stream_index,-1),count(*) FROM local_speakers WHERE clip_id=ANY($1) AND kind='diarized' GROUP BY clip_id,audio_stream_index`, func(r pgx.Rows) error {
			var kind string
			var id, index, count int64
			if err := r.Scan(&kind, &id, &index, &count); err != nil {
				return err
			}
			switch kind {
			case "optout":
				in.optedOut[id] = true
			case "blocked":
				in.blocked[[2]int64{id, index}] = true
			default:
				in.diarizedCount[[2]int64{id, index}] = int(count)
			}
			return nil
		}},
	}
	for _, step := range steps {
		if err := s.scanAll(ctx, step.sql, ids, step.scan); err != nil {
			return nil, err
		}
	}
	return in, nil
}
