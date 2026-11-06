//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// watchUICompareSeed serves the visual comparison (#251): once the browser suite has created
// the synthetic clip through the API, it writes its id in <dir>/seed-request. The attributions
// of the prototype's five voices are then written straight into the database (the
// fingerprints step of this clip stays failed), and <dir>/seed-done is written ("ok" or the error).
func watchUICompareSeed(ctx context.Context, pool *pgxpool.Pool, dir string, stop <-chan struct{}) {
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Millisecond):
			}
			raw, err := os.ReadFile(filepath.Join(dir, "seed-request"))
			if err != nil {
				continue
			}
			result := "ok"
			clipID, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
			if err == nil {
				err = seedUICompareVoices(ctx, pool, clipID)
			}
			if err != nil {
				result = err.Error()
			}
			os.WriteFile(filepath.Join(dir, "seed-done"), []byte(result), 0o600)
			return
		}
	}()
}

// SPEAKER_00 validated as Marc Lefèvre, SPEAKER_01 a suggestion for Julie Martin (with Claire
// Dubois as second candidate), SPEAKER_02 ambiguous between Thomas Roux and Hugo Bernard,
// SPEAKER_03 and SPEAKER_04 unknown (no attribution).
func seedUICompareVoices(ctx context.Context, pool *pgxpool.Pool, clipID int64) error {
	person := func(name string) (id int64, err error) {
		err = pool.QueryRow(ctx, `SELECT id FROM persons WHERE display_name=$1`, name).Scan(&id)
		return
	}
	speaker := func(label string) (id int64, err error) {
		err = pool.QueryRow(ctx, `SELECT id FROM local_speakers WHERE clip_id=$1 AND label=$2`, clipID, label).Scan(&id)
		return
	}
	attribute := func(label, level, state string, who string, confidence *float64, origin string, candidates map[string]float64) error {
		sid, err := speaker(label)
		if err != nil {
			return fmt.Errorf("speaker %s: %w", label, err)
		}
		var pid *int64
		if who != "" {
			p, err := person(who)
			if err != nil {
				return err
			}
			pid = &p
		}
		var aid int64
		if err = pool.QueryRow(ctx, `INSERT INTO speaker_attributions(local_speaker_id,person_id,decision_level,validation_state,confidence,origin) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, sid, pid, level, state, confidence, origin).Scan(&aid); err != nil {
			return err
		}
		rank := 0
		for _, name := range []string{"Julie Martin", "Claire Dubois", "Thomas Roux", "Hugo Bernard"} {
			score, ok := candidates[name]
			if !ok {
				continue
			}
			cid, err := person(name)
			if err != nil {
				return err
			}
			rank++
			if _, err = pool.Exec(ctx, `INSERT INTO speaker_attribution_candidates(attribution_id,person_id,rank,score) VALUES($1,$2,$3,$4)`, aid, cid, rank, score); err != nil {
				return err
			}
		}
		return nil
	}
	f := func(v float64) *float64 { return &v }
	if err := attribute("SPEAKER_00", "human", "validated", "Marc Lefèvre", nil, "human", nil); err != nil {
		return err
	}
	if err := attribute("SPEAKER_01", "suggestion", "unvalidated", "Julie Martin", f(0.91), matchingOrigin, map[string]float64{"Julie Martin": 0.91, "Claire Dubois": 0.8}); err != nil {
		return err
	}
	if err := attribute("SPEAKER_02", "ambiguous", "unvalidated", "", f(0.83), matchingOrigin, map[string]float64{"Thomas Roux": 0.83, "Hugo Bernard": 0.81}); err != nil {
		return err
	}
	return seedUICompareFolders(ctx, pool)
}

// The folders shown under the titles of the "To process" list, as in the prototype: one source
// file per clip (the clips themselves were uploaded, so they have none).
func seedUICompareFolders(ctx context.Context, pool *pgxpool.Pool) error {
	// Display titles without the file extension, like the prototype's titles.
	if _, err := pool.Exec(ctx, `UPDATE clips SET display_title=regexp_replace(original_name,'\.mp4$','') WHERE display_title IS NULL`); err != nil {
		return err
	}
	var sourceID int64
	if err := pool.QueryRow(ctx, `INSERT INTO clip_sources(name,root_path,owner_id) SELECT 'ui-compare-folders','/nonexistent/ui-compare-folders',id FROM users WHERE username='administrator' RETURNING id`).Scan(&sourceID); err != nil {
		return err
	}
	for title, folder := range map[string]string{
		"Réunion de préparation — tournage S12": "2026/novembre",
		"Rushes parking — prise 3":              "2026/octobre/parking",
		"Interview Claire — making-of":          "2026/octobre",
		"Brief équipe lumière":                  "2026/octobre",
		"Visio production — budget":             "2026/septembre",
		"Repérages Studio B":                    "2026/septembre",
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,clip_id,size_bytes,modified_at,last_seen_generation,status) SELECT $1,$2||'/'||original_name,id,10,now(),0,'ready' FROM clips WHERE original_name=$3`, sourceID, folder, title+".mp4"); err != nil {
			return err
		}
	}
	return nil
}
