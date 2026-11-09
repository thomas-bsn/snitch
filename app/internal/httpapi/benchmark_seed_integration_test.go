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
	// A certified clip per corpus tells where its reference comes from (ADR 0061): the demonstration corpus relies on
	// the production pipeline, so its results carry the bias warning; the second one on an imported file.
	var owner, clip int64
	if err := pool.QueryRow(ctx, `INSERT INTO users(username,role,approved) VALUES('seed-owner','user',true) RETURNING id`).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO clips(owner_id,original_name,size_bytes,modified_at,recorded_at) VALUES($1,'seed-reference.mp4',1,now(),now()) RETURNING id`, owner).Scan(&clip); err != nil {
		t.Fatal(err)
	}
	var imported int64
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_datasets(name,author_name,stages) VALUES('Corpus importé','seed','{transcription}') RETURNING id`).Scan(&imported); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO benchmark_dataset_clips(dataset_id,clip_id,role,verified,reference_source) VALUES($1,$2,'test',true,'import')`, imported, clip)
	// The newest corpus is the one the screen offers first: the demonstration one, created last.
	var dataset int64
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_datasets(name,author_name,stages) VALUES('Corpus de démonstration','seed','{fingerprint}') RETURNING id`).Scan(&dataset); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO benchmark_dataset_clips(dataset_id,clip_id,role,verified,reference_source) VALUES($1,$2,'test',true,'production')`, dataset, clip)
	// A transcription run on the imported corpus: no production bias to announce. It is the oldest run, so the best
	// model of the stage stays the one of the demonstration corpus.
	var importedRun int64
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids,author_name,finished_at) VALUES($1,'Corpus importé','transcription','completed','{seed-asr-c,seed-asr-d}','seed',now()-interval '1 day') RETURNING id`, imported).Scan(&importedRun); err != nil {
		t.Fatal(err)
	}
	wordMetrics := func(wer, cer float64) string {
		return `{"clips":3,"reference_words":420,"word_errors":` + strconv.Itoa(int(wer*420)) + `,"reference_chars":2300,"character_errors":` + strconv.Itoa(int(cer*2300)) + `,"cer_skipped_clips":0,"wer":` + seedNum(wer) + `,"cer":` + seedNum(cer) + `}`
	}
	exec(`INSERT INTO benchmark_results(run_id,model_id,model,model_version,dimensions,state,items_total,items_scored,elapsed_seconds,metrics) VALUES
		($1,'seed-asr-c','seed-asr-c','v1',0,'complete',3,3,95,$2::jsonb),
		($1,'seed-asr-d','seed-asr-d','v1',0,'complete',3,3,140,$3::jsonb)`,
		importedRun, wordMetrics(0.3, 0.15), wordMetrics(0.25, 0.12))
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
	// A finished diarization run: the Sortformer model has the lower DER, so it differs from the production model.
	var diarization int64
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids,author_name,finished_at) VALUES($1,'Corpus de démonstration','diarization','completed','{pyannote-community-1,nvidia-sortformer-4spk-v2.1}','seed',now()) RETURNING id`, dataset).Scan(&diarization); err != nil {
		t.Fatal(err)
	}
	der := func(total, missed, falseAlarm, confusion float64) string {
		return `{"clips":3,"skipped_clips":0,"reference_speech_seconds":600,"missed_speech_seconds":` + seedNum(missed*600) + `,"false_alarm_seconds":` + seedNum(falseAlarm*600) + `,"confusion_seconds":` + seedNum(confusion*600) +
			`,"collar_seconds":0.25,"der":` + seedNum(total) + `,"missed_speech_rate":` + seedNum(missed) + `,"false_alarm_rate":` + seedNum(falseAlarm) + `,"confusion_rate":` + seedNum(confusion) + `}`
	}
	exec(`INSERT INTO benchmark_results(run_id,model_id,model,model_version,dimensions,state,items_total,items_scored,elapsed_seconds,metrics) VALUES
		($1,'pyannote-community-1','pyannote/speaker-diarization-community-1','3533c8c',0,'complete',3,3,80,$2::jsonb),
		($1,'nvidia-sortformer-4spk-v2.1','nvidia/diar_streaming_sortformer_4spk-v2.1','cd03eee',0,'complete',3,3,146,$3::jsonb)`,
		diarization, der(0.114, 0.04, 0.03, 0.044), der(0.07, 0.03, 0.02, 0.02))
	// A finished end-to-end run: two chains sharing a diarization model, the second one with the more accurate
	// transcription. The ids of the results (c1, c2) stand for the pairs stored with the run.
	var e2e int64
	chains := `{"chains":[{"id":"c1","diarization_model_id":"pyannote-community-1","transcription_model":"seed-asr-a"},{"id":"c2","diarization_model_id":"pyannote-community-1","transcription_model":"seed-asr-b"}]}`
	if err := pool.QueryRow(ctx, `INSERT INTO benchmark_runs(dataset_id,dataset_name,stage,status,model_ids,params,author_name,finished_at) VALUES($1,'Corpus de démonstration','e2e','completed','{c1,c2}',$2::jsonb,'seed',now()) RETURNING id`, dataset, chains).Scan(&e2e); err != nil {
		t.Fatal(err)
	}
	attributed := func(cpwer, wer float64) string {
		return `{"clips":3,"reference_words":420,"cp_word_errors":` + strconv.Itoa(int(cpwer*420)) + `,"word_errors":` + strconv.Itoa(int(wer*420)) + `,"cpwer":` + seedNum(cpwer) + `,"wer":` + seedNum(wer) +
			`,"reference_speakers":2,"model_speakers":2,"der":0.114}`
	}
	exec(`INSERT INTO benchmark_results(run_id,model_id,model,model_version,dimensions,state,items_total,items_scored,elapsed_seconds,metrics) VALUES
		($1,'c1','pyannote/speaker-diarization-community-1 + seed-asr-a','',0,'complete',3,3,175,$2::jsonb),
		($1,'c2','pyannote/speaker-diarization-community-1 + seed-asr-b','',0,'complete',3,3,220,$3::jsonb)`,
		e2e, attributed(0.19, 0.12), attributed(0.105, 0.081))
	// One Worker with the community diarization model ready, so the browser can compose chains and start a run.
	exec(`INSERT INTO workers(id,name,token_hash,max_concurrency,reported_concurrency,last_seen_at) VALUES('wk_seedbenchmarkworker0001','Seed benchmark worker',decode(repeat('ef',32),'hex'),1,1,now())`)
	exec(`INSERT INTO worker_voice_models(worker_id,model_id,state) VALUES('wk_seedbenchmarkworker0001','pyannote-community-1','ready')`)
}

func seedNum(x float64) string { return strconv.FormatFloat(x, 'f', -1, 64) }
