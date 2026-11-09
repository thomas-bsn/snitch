//go:build integration

package httpapi

import (
	"context"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedBenchmarkRuns gives the browser suite finished runs to read: a corpus, a run of two models and a failed one.
// Rows are written directly: running the Workers is the job of the API tests, the browser tests read the screen.
func seedBenchmarkRuns(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	metrics := func(eer, threshold, top1, precision, recall float64, fp, fn int, genuine, impostor float64) string {
		return `{"references":10,"tests":8,"persons":4,"genuine_trials":8,"impostor_trials":24,"eer":` + seedNum(eer) + `,"eer_threshold":` + seedNum(threshold) +
			`,"top1_accuracy":` + seedNum(top1) + `,"true_positives":7,"false_positives":` + strconv.Itoa(fp) + `,"false_negatives":` + strconv.Itoa(fn) +
			`,"precision":` + seedNum(precision) + `,"recall":` + seedNum(recall) + `,"genuine_mean_score":` + seedNum(genuine) + `,"impostor_mean_score":` + seedNum(impostor) + `,"ignored_dimension":0}`
	}
	var dataset int64
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_datasets(name,author_name,stages) VALUES('Corpus de démonstration','seed','{fingerprint}') RETURNING id`).Scan(&dataset); err != nil {
		t.Fatal(err)
	}
	var finished, failed int64
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids,author_name,finished_at) VALUES($1,'Corpus de démonstration','fingerprint','completed','{pyannote-wespeaker-resnet34,seed-titanet}','seed',now()) RETURNING id`, dataset).Scan(&finished); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO benchmark_results(run_id,model_id,model,model_version,dimensions,state,items_total,items_scored,elapsed_seconds,metrics) VALUES
		($1,'pyannote-wespeaker-resnet34','pyannote/wespeaker-voxceleb-resnet34-LM','v1',256,'complete',18,18,192,$2::jsonb),
		($1,'seed-titanet','nvidia/titanet-large','v1',192,'partial',18,14,280,$3::jsonb)`,
		finished, metrics(0.05, 0.6, 0.91, 0.93, 0.88, 2, 4, 0.76, 0.22), metrics(0.031, 0.55, 0.95, 0.97, 0.92, 1, 2, 0.8, 0.2))
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids,author_name,finished_at) VALUES($1,'Corpus de démonstration','fingerprint','failed','{seed-titanet}','seed',now()) RETURNING id`, dataset).Scan(&failed); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO benchmark_results(run_id,model_id,model,model_version,dimensions,state,items_total,items_scored,metrics) VALUES($1,'seed-titanet','nvidia/titanet-large','',0,'failed',18,0,$2::jsonb)`,
		failed, `{"references":0,"tests":0,"persons":0,"genuine_trials":0,"impostor_trials":0,"true_positives":0,"false_positives":0,"false_negatives":0,"ignored_dimension":0}`)
	// A finished transcription run: model-b is more accurate, so the summary crowns it and it differs from production.
	var transcription int64
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids,author_name,finished_at) VALUES($1,'Corpus de démonstration','transcription','completed','{seed-asr-a,seed-asr-b}','seed',now()) RETURNING id`, dataset).Scan(&transcription); err != nil {
		t.Fatal(err)
	}
	words := func(wer, cer float64) string {
		return `{"clips":3,"reference_words":420,"word_errors":` + strconv.Itoa(int(wer*420)) + `,"reference_chars":2300,"character_errors":` + strconv.Itoa(int(cer*2300)) + `,"cer_skipped_clips":0,"wer":` + seedNum(wer) + `,"cer":` + seedNum(cer) + `}`
	}
	exec(`INSERT INTO benchmark_results(run_id,model_id,model,model_version,dimensions,state,items_total,items_scored,elapsed_seconds,metrics) VALUES
		($1,'seed-asr-a','seed-asr-a','v1',0,'complete',3,3,95,$2::jsonb),
		($1,'seed-asr-b','seed-asr-b','v1',0,'complete',3,3,140,$3::jsonb)`,
		transcription, words(0.12, 0.05), words(0.081, 0.032))
}

func seedNum(x float64) string { return strconv.FormatFloat(x, 'f', -1, 64) }
