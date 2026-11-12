//go:build integration

package httpapi

import (
	"cmp"
	"context"
	"encoding/json"
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/segmentparts"
)

// The Person filter reads the word rule in SQL (segment_part_speakers), and corrections the segment rule
// (dominant_turn_speaker); the clip page and the benchmark read both in Go (segmentparts.Attribute). Both must name the
// same voices, in the same order, for the same words and turns: worked cases first, then generated ones on a coarse
// time grid so ties, uncovered words and single words abound.
func testSegmentPartsParity(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	h := newVoiceHarness(t, pool)
	const stream, otherStream = 4, 5
	var job int64
	if err := pool.QueryRow(ctx, `INSERT INTO jobs(clip_id,step,input_version,status) VALUES($1,'voice.diarization.4','parity','done') RETURNING id`, h.clipID).Scan(&job); err != nil {
		t.Fatal(err)
	}
	speaker := func(track int, label string, ordinal int) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO local_speakers(clip_id,audio_stream_index,diarization_job_id,label,ordinal,speaking_seconds,turn_count) VALUES($1,$2,$3,$4,$5,0,0) RETURNING id`, h.clipID, track, job, label, ordinal).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	voices := []int64{speaker(stream, "A", 0), speaker(stream, "B", 1), speaker(stream, "C", 2)}
	// A voice of another track speaks all the time: it must never get a word of this track (ADR 0052).
	stranger := speaker(otherStream, "X", 0)
	if _, err := pool.Exec(ctx, `INSERT INTO speaker_turns(local_speaker_id,clip_id,turn_index,start_seconds,end_seconds) VALUES($1,$2,0,0,1000)`, stranger, h.clipID); err != nil {
		t.Fatal(err)
	}

	type scenario struct {
		name    string
		segment segmentparts.Span
		words   []segmentparts.Word
		turns   []segmentparts.Turn[int64] // Speaker is an index in voices
	}
	w := func(bounds ...float64) []segmentparts.Word {
		out := []segmentparts.Word{}
		for i := 0; i+1 < len(bounds); i += 2 {
			out = append(out, segmentparts.Word{Start: bounds[i], End: bounds[i+1]})
		}
		return out
	}
	turn := func(voice int64, start, end float64) segmentparts.Turn[int64] {
		return segmentparts.Turn[int64]{Speaker: voice, Start: start, End: end}
	}
	scenarios := []scenario{
		{"two voices answer each other", segmentparts.Span{Start: 0, End: 4}, w(0, 1, 1, 2, 2, 3, 3, 4), []segmentparts.Turn[int64]{turn(0, 0, 2.2), turn(1, 2.2, 4)}},
		{"a single word between two words of another voice", segmentparts.Span{Start: 0, End: 3}, w(0, 1, 1, 2, 2, 3), []segmentparts.Turn[int64]{turn(0, 0, 1), turn(1, 1, 2), turn(0, 2, 3)}},
		{"words without turn", segmentparts.Span{Start: 0, End: 5}, w(0, 1, 1, 2, 2, 3, 3, 4, 4, 5), []segmentparts.Turn[int64]{turn(0, 1, 2), turn(1, 3, 5)}},
		{"no turn at all", segmentparts.Span{Start: 0, End: 2}, w(0, 1, 1, 2), nil},
		{"a tie keeps the previous voice", segmentparts.Span{Start: 0, End: 4}, w(0, 1, 1, 2, 2, 3, 3, 4), []segmentparts.Turn[int64]{turn(0, 1, 2), turn(0, 2, 4), turn(1, 0, 2)}},
		{"turns between the words only", segmentparts.Span{Start: 0, End: 3}, w(0, 1, 2, 3), []segmentparts.Turn[int64]{turn(1, 1, 2)}},
		{"no word", segmentparts.Span{Start: 0, End: 3}, nil, []segmentparts.Turn[int64]{turn(2, 1, 2), turn(0, 0, 0.5)}},
		{"words without time", segmentparts.Span{Start: 0, End: 3}, w(0, 0, 0, 0), []segmentparts.Turn[int64]{turn(2, 1, 2)}},
		// Listed out of ordinal order: production reads turns by ordinal, so the tie goes to A on both sides.
		{"a segment tie goes to the lowest ordinal", segmentparts.Span{Start: 0, End: 2}, nil, []segmentparts.Turn[int64]{turn(1, 0, 1), turn(0, 1, 2)}},
		{"a segment tie under words goes to the lowest ordinal", segmentparts.Span{Start: 0, End: 2}, w(0, 2), []segmentparts.Turn[int64]{turn(2, 0, 1), turn(1, 1, 2)}},
	}
	random := rand.New(rand.NewSource(398))
	grid := func(max int) float64 { return float64(random.Intn(max)) / 2 }
	for i := 0; i < 150; i++ {
		sc := scenario{name: "generated", segment: segmentparts.Span{Start: 0, End: 10}}
		at := 0.0
		for n := random.Intn(9); n > 0; n-- {
			start := at + grid(3)
			end := start + grid(4)
			sc.words = append(sc.words, segmentparts.Word{Start: start, End: end})
			at = end
		}
		// Turns grouped by voice in ordinal order, as the clip page reads them.
		for voice := range voices {
			at = 0
			for n := random.Intn(4); n > 0; n-- {
				start := at + grid(6)
				end := start + grid(6)
				sc.turns = append(sc.turns, turn(int64(voice), start, end))
				at = end
			}
		}
		scenarios = append(scenarios, sc)
	}

	several := 0
	for i, sc := range scenarios {
		if _, err := pool.Exec(ctx, `DELETE FROM speaker_turns WHERE local_speaker_id=ANY($1)`, voices); err != nil {
			t.Fatal(err)
		}
		turns := make([]segmentparts.Turn[int64], len(sc.turns))
		for k, tr := range sc.turns {
			turns[k] = turn(voices[tr.Speaker], tr.Start, tr.End)
			if _, err := pool.Exec(ctx, `INSERT INTO speaker_turns(local_speaker_id,clip_id,turn_index,start_seconds,end_seconds) VALUES($1,$2,$3,$4,$5)`, turns[k].Speaker, h.clipID, k, tr.Start, tr.End); err != nil {
				t.Fatal(err)
			}
		}
		// The order the Go callers read turns in (ORDER BY ls.ordinal,st.turn_index), which decides ties: voices are
		// created in ordinal order.
		slices.SortStableFunc(turns, func(a, b segmentparts.Turn[int64]) int { return cmp.Compare(a.Speaker, b.Speaker) })
		// The segment rule: the voice of a segment for corrections and the search under the segment rule.
		whole := segmentparts.Attribute(segmentparts.RuleSegment, sc.segment, sc.words, turns)[0]
		deduced, _, found, err := deducedSpeaker(ctx, pool, h.clipID, stream, sc.segment.Start, sc.segment.End)
		if err != nil {
			t.Fatal(err)
		}
		if found != whole.Known || deduced != whole.Speaker {
			t.Errorf("scenario %d (%s): SQL segment rule names %d (%v), Go %d (%v)\nturns %v", i, sc.name, deduced, found, whole.Speaker, whole.Known, turns)
		}
		// What the production code names: the voice of each part, once, in order of first part.
		want := []*int64{}
		seen := map[int64]bool{}
		for _, part := range segmentparts.Attribute(segmentparts.RuleWord, sc.segment, sc.words, turns) {
			switch {
			case !part.Known:
				want = append(want, nil)
			case !seen[part.Speaker]:
				id := part.Speaker
				seen[id] = true
				want = append(want, &id)
			}
		}
		if len(want) > 1 {
			several++
		}
		words := []map[string]any{}
		for _, word := range sc.words {
			if word.End == 0 {
				words = append(words, map[string]any{"word": "x"})
				continue
			}
			words = append(words, map[string]any{"word": "x", "start": word.Start, "end": word.End})
		}
		raw, _ := json.Marshal(words)
		rows, err := pool.Query(ctx, `SELECT id FROM segment_part_speakers($1,$2,$3,$4,NULL,NULL,$5::jsonb) ORDER BY part_order`, h.clipID, stream, sc.segment.Start, sc.segment.End, string(raw))
		if err != nil {
			t.Fatal(err)
		}
		got := []*int64{}
		for rows.Next() {
			var id *int64
			if err = rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			got = append(got, id)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("scenario %d (%s): SQL names %v, Go names %v\nwords %v\nturns %v", i, sc.name, deref(got), deref(want), sc.words, turns)
		}
		for _, id := range got {
			if id != nil && *id == stranger {
				t.Errorf("scenario %d: a word took a voice of another track", i)
			}
		}
	}
	if several < 20 {
		t.Fatalf("the generated scenarios must exercise segments with several parts, got %d", several)
	}
}

func deref(ids []*int64) []int64 {
	out := []int64{}
	for _, id := range ids {
		if id == nil {
			out = append(out, 0)
			continue
		}
		out = append(out, *id)
	}
	return out
}
