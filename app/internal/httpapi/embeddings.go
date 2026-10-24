package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type embeddingBatch struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}
type embeddingResponse struct {
	Data []embeddingData `json:"data"`
}
type embeddingData struct {
	Index     *int      `json:"index"`
	Embedding []float32 `json:"embedding"`
}
type indexSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
	Index int     `json:"-"`
}
type indexResult struct {
	Segments []indexSegment `json:"segments"`
}

func (s *Server) ensureEmbeddingProfile(ctx context.Context) error {
	if s.Config.EmbeddingsURL == "" || s.Config.EmbeddingsModel == "" {
		return nil
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var active *int64
	if err := tx.QueryRow(ctx, `SELECT active_profile_id FROM embedding_index_state WHERE id=1 FOR UPDATE`).Scan(&active); err != nil {
		return err
	}
	if active != nil {
		return tx.Commit(ctx)
	}
	var id int64
	if err := tx.QueryRow(ctx, `INSERT INTO embedding_profiles(model,state) VALUES($1,'active') RETURNING id`, s.Config.EmbeddingsModel).Scan(&id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE embedding_index_state SET active_profile_id=$1,updated_at=now() WHERE id=1 AND active_profile_id IS NULL`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) embeddingLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		started := time.Now()
		err := s.indexNext(ctx)
		delay := 50*time.Millisecond - time.Since(started)
		if err != nil {
			delay = time.Second
			if s.Log != nil {
				s.Log.Warn("transcript indexing", "error", err)
			}
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}
}

func (s *Server) indexNext(ctx context.Context) error {
	if s.Config.EmbeddingsURL == "" {
		return nil
	}
	var taskID, profileID, jobID, clipID int64
	var model string
	var raw []byte
	err := s.Pool.QueryRow(ctx, `WITH candidate AS (SELECT t.id FROM embedding_tasks t JOIN embedding_index_state st ON st.id=1 WHERE (t.state='pending' OR (t.state='leased' AND t.lease_expires_at<now())) AND t.profile_id IN (st.active_profile_id,st.candidate_profile_id) ORDER BY t.id FOR UPDATE OF t SKIP LOCKED LIMIT 1) UPDATE embedding_tasks t SET state='leased',lease_expires_at=now()+interval '2 minutes',attempts=t.attempts+1,updated_at=now() FROM candidate c JOIN jobs j ON j.id=(SELECT transcription_job_id FROM embedding_tasks WHERE id=c.id) JOIN embedding_profiles p ON p.id=(SELECT profile_id FROM embedding_tasks WHERE id=c.id) WHERE t.id=c.id RETURNING t.id,t.profile_id,t.transcription_job_id,j.clip_id,j.result,p.model`).Scan(&taskID, &profileID, &jobID, &clipID, &raw, &model)
	if err == pgx.ErrNoRows {
		return s.activateCompleteCandidate(ctx)
	}
	if err != nil {
		return err
	}
	var transcript indexResult
	if err = json.Unmarshal(raw, &transcript); err != nil {
		return s.embeddingTaskFailed(ctx, taskID, err)
	}
	segments := make([]indexSegment, 0, len(transcript.Segments))
	for index, seg := range transcript.Segments {
		if strings.TrimSpace(seg.Text) != "" {
			seg.Index = index
			segments = append(segments, seg)
		}
	}
	if len(segments) == 0 {
		_, err = s.Pool.Exec(ctx, `UPDATE embedding_tasks SET state='done',lease_expires_at=NULL,error=NULL,updated_at=now() WHERE id=$1`, taskID)
		return err
	}
	inputs := make([]string, len(segments))
	for i, v := range segments {
		inputs[i] = v.Text
	}
	body, _ := json.Marshal(embeddingBatch{Model: model, Input: inputs})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Config.EmbeddingsURL, strings.NewReader(string(body)))
	if err != nil {
		return s.embeddingTaskFailed(ctx, taskID, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		return s.embeddingTaskFailed(ctx, taskID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return s.embeddingTaskFailed(ctx, taskID, fmt.Errorf("provider status %d", resp.StatusCode))
	}
	var result embeddingResponse
	if err = json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&result); err != nil {
		return s.embeddingTaskFailed(ctx, taskID, err)
	}
	if err = orderEmbeddingData(result.Data); err != nil {
		return s.embeddingTaskFailed(ctx, taskID, err)
	}
	if len(result.Data) != len(segments) {
		return s.embeddingTaskFailed(ctx, taskID, fmt.Errorf("provider returned %d vectors for %d segments", len(result.Data), len(segments)))
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var dimensions int
	if err = tx.QueryRow(ctx, `SELECT COALESCE(dimensions,0) FROM embedding_profiles WHERE id=$1 FOR UPDATE`, profileID).Scan(&dimensions); err != nil {
		return err
	}
	for _, item := range result.Data {
		if len(item.Embedding) == 0 || len(item.Embedding) > 4000 || (dimensions > 0 && dimensions != len(item.Embedding)) {
			return s.embeddingTaskFailed(ctx, taskID, fmt.Errorf("invalid embedding dimension"))
		}
		norm := float64(0)
		for _, value := range item.Embedding {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return s.embeddingTaskFailed(ctx, taskID, fmt.Errorf("invalid embedding value"))
			}
			norm += float64(value) * float64(value)
		}
		if norm == 0 {
			return s.embeddingTaskFailed(ctx, taskID, fmt.Errorf("zero embedding vector"))
		}
		dimensions = len(item.Embedding)
	}
	if _, err = tx.Exec(ctx, `UPDATE embedding_profiles SET dimensions=$2,updated_at=now() WHERE id=$1`, profileID, dimensions); err != nil {
		return err
	}
	for i, seg := range segments {
		vals := make([]string, len(result.Data[i].Embedding))
		for j, v := range result.Data[i].Embedding {
			vals[j] = strconv.FormatFloat(float64(v), 'g', -1, 32)
		}
		vector := "[" + strings.Join(vals, ",") + "]"
		if _, err = tx.Exec(ctx, `INSERT INTO transcript_embeddings(profile_id,transcription_job_id,clip_id,segment_index,segment_start,segment_end,text,embedding) VALUES($1,$2,$3,$4,$5,$6,$7,$8::vector) ON CONFLICT(profile_id,transcription_job_id,segment_index) DO UPDATE SET segment_start=EXCLUDED.segment_start,segment_end=EXCLUDED.segment_end,text=EXCLUDED.text,embedding=EXCLUDED.embedding`, profileID, jobID, clipID, seg.Index, seg.Start, seg.End, seg.Text, vector); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE embedding_tasks SET state='done',lease_expires_at=NULL,error=NULL,updated_at=now() WHERE id=$1`, taskID); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	indexName := fmt.Sprintf("transcript_embeddings_p%d_hnsw", profileID)
	_, err = s.Pool.Exec(ctx, fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON transcript_embeddings USING hnsw ((embedding::halfvec(%d)) halfvec_cosine_ops) WHERE profile_id=%d`, indexName, dimensions, profileID))
	if err != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE embedding_profiles SET state='failed',error=left($2,240),updated_at=now() WHERE id=$1`, profileID, err.Error())
		return err
	}
	return s.activateCompleteCandidate(ctx)
}

func (s *Server) activateCompleteCandidate(ctx context.Context) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var candidate *int64
	err = tx.QueryRow(ctx, `SELECT candidate_profile_id FROM embedding_index_state WHERE id=1 FOR UPDATE`).Scan(&candidate)
	if err != nil || candidate == nil {
		return err
	}
	var candidateState string
	if err = tx.QueryRow(ctx, `SELECT state FROM embedding_profiles WHERE id=$1`, *candidate).Scan(&candidateState); err != nil {
		return err
	}
	if candidateState == "failed" {
		return tx.Commit(ctx)
	}
	var pending, failed int
	if err = tx.QueryRow(ctx, `SELECT count(*) FILTER(WHERE state IN ('pending','leased')),count(*) FILTER(WHERE state='failed') FROM embedding_tasks WHERE profile_id=$1`, *candidate).Scan(&pending, &failed); err != nil {
		return err
	}
	if failed > 0 {
		_, err = tx.Exec(ctx, `UPDATE embedding_profiles SET state='failed',error='Une ou plusieurs transcriptions n’ont pas pu être indexées.',updated_at=now() WHERE id=$1`, *candidate)
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if pending > 0 {
		return nil
	}
	if _, err = tx.Exec(ctx, `UPDATE embedding_profiles SET state='retired',updated_at=now() WHERE id=(SELECT active_profile_id FROM embedding_index_state WHERE id=1)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE embedding_profiles SET state='active',error=NULL,updated_at=now() WHERE id=$1`, *candidate); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE embedding_index_state SET active_profile_id=candidate_profile_id,candidate_profile_id=NULL,updated_at=now() WHERE id=1`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) embeddingTaskFailed(ctx context.Context, id int64, cause error) error {
	_, err := s.Pool.Exec(ctx, `UPDATE embedding_tasks SET state='failed',lease_expires_at=NULL,error=left($2,240),updated_at=now() WHERE id=$1`, id, cause.Error())
	if err != nil {
		return err
	}
	return cause
}

func (s *Server) vectorFor(ctx context.Context, input string) (string, error) {
	var model string
	if err := s.Pool.QueryRow(ctx, `SELECT p.model FROM embedding_index_state s JOIN embedding_profiles p ON p.id=s.active_profile_id WHERE s.id=1`).Scan(&model); err != nil {
		return "", err
	}
	body, _ := json.Marshal(embeddingBatch{Model: model, Input: []string{input}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Config.EmbeddingsURL, strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("provider status %d", resp.StatusCode)
	}
	var result embeddingResponse
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return "", err
	}
	if err = orderEmbeddingData(result.Data); err != nil {
		return "", err
	}
	if len(result.Data) != 1 || len(result.Data[0].Embedding) == 0 || len(result.Data[0].Embedding) > 4000 {
		return "", fmt.Errorf("invalid embedding response")
	}
	norm := float64(0)
	for _, value := range result.Data[0].Embedding {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return "", fmt.Errorf("invalid embedding value")
		}
		norm += float64(value) * float64(value)
	}
	if norm == 0 {
		return "", fmt.Errorf("zero embedding vector")
	}
	vals := make([]string, len(result.Data[0].Embedding))
	for i, v := range result.Data[0].Embedding {
		vals[i] = strconv.FormatFloat(float64(v), 'g', -1, 32)
	}
	return "[" + strings.Join(vals, ",") + "]", nil
}

func orderEmbeddingData(data []embeddingData) error {
	indexed := 0
	for _, item := range data {
		if item.Index != nil {
			indexed++
		}
	}
	if indexed == 0 {
		return nil
	}
	if indexed != len(data) {
		return fmt.Errorf("embedding response has incomplete indexes")
	}
	ordered := make([]embeddingData, len(data))
	seen := make([]bool, len(data))
	for _, item := range data {
		index := *item.Index
		if index < 0 || index >= len(data) || seen[index] {
			return fmt.Errorf("embedding response has invalid indexes")
		}
		seen[index] = true
		ordered[index] = item
	}
	copy(data, ordered)
	return nil
}

func (s *Server) embeddingStatus(w http.ResponseWriter, r *http.Request) {
	var out struct {
		Configured       bool   `json:"configured"`
		Provider         string `json:"provider"`
		Model            string `json:"model"`
		Dimensions       int    `json:"dimensions"`
		State            string `json:"state"`
		Error            string `json:"error,omitempty"`
		Pending          int    `json:"pending"`
		Done             int    `json:"done"`
		Failed           int    `json:"failed"`
		CandidateModel   string `json:"candidate_model,omitempty"`
		CandidateState   string `json:"candidate_state,omitempty"`
		CandidateError   string `json:"candidate_error,omitempty"`
		CandidateTotal   int    `json:"candidate_total,omitempty"`
		CandidateDone    int    `json:"candidate_done,omitempty"`
		CandidatePending int    `json:"candidate_pending,omitempty"`
		CandidateFailed  int    `json:"candidate_failed,omitempty"`
	}
	out.Configured = s.Config.EmbeddingsURL != ""
	out.Provider = "openai-compatible"
	out.State = "unconfigured"
	err := s.Pool.QueryRow(r.Context(), `SELECT COALESCE(p.dimensions,0),COALESCE(p.state,'unconfigured'),count(*) FILTER(WHERE t.state IN ('pending','leased')),count(*) FILTER(WHERE t.state='done'),count(*) FILTER(WHERE t.state='failed'),COALESCE((SELECT error FROM embedding_tasks et WHERE et.profile_id=p.id AND et.state='failed' ORDER BY et.updated_at DESC LIMIT 1),p.error,'') FROM embedding_index_state s LEFT JOIN embedding_profiles p ON p.id=s.active_profile_id LEFT JOIN embedding_tasks t ON t.profile_id=p.id WHERE s.id=1 GROUP BY p.dimensions,p.state,p.error`).Scan(&out.Dimensions, &out.State, &out.Pending, &out.Done, &out.Failed, &out.Error)
	if err != nil && err != pgx.ErrNoRows {
		s.internal(w, err)
		return
	}
	if err = s.Pool.QueryRow(r.Context(), `SELECT model FROM embedding_profiles p JOIN embedding_index_state st ON st.active_profile_id=p.id WHERE st.id=1`).Scan(&out.Model); err != nil && err != pgx.ErrNoRows {
		s.internal(w, err)
		return
	}
	if err = s.Pool.QueryRow(r.Context(), `SELECT p.model,p.state,count(t.id),count(*) FILTER(WHERE t.state='done'),count(*) FILTER(WHERE t.state IN ('pending','leased')),count(*) FILTER(WHERE t.state='failed'),COALESCE((SELECT error FROM embedding_tasks et WHERE et.profile_id=p.id AND et.state='failed' ORDER BY et.updated_at DESC LIMIT 1),p.error,'') FROM embedding_index_state st JOIN embedding_profiles p ON p.id=st.candidate_profile_id LEFT JOIN embedding_tasks t ON t.profile_id=p.id WHERE st.id=1 GROUP BY p.model,p.state,p.error`).Scan(&out.CandidateModel, &out.CandidateState, &out.CandidateTotal, &out.CandidateDone, &out.CandidatePending, &out.CandidateFailed, &out.CandidateError); err != nil && err != pgx.ErrNoRows {
		s.internal(w, err)
		return
	}
	reply(w, 200, out)
}

func (s *Server) backfillEmbeddings(w http.ResponseWriter, r *http.Request) {
	tag, err := s.Pool.Exec(r.Context(), `WITH latest AS (SELECT DISTINCT ON (clip_id) id FROM jobs WHERE status='done' AND step LIKE 'transcription.audio.%' ORDER BY clip_id,updated_at DESC,id DESC) INSERT INTO embedding_tasks(profile_id,transcription_job_id) SELECT p.id,j.id FROM embedding_profiles p JOIN embedding_index_state st ON st.active_profile_id=p.id CROSS JOIN latest j ON CONFLICT(profile_id,transcription_job_id) DO UPDATE SET state='pending',lease_expires_at=NULL,error=NULL,updated_at=now() WHERE embedding_tasks.state='failed'`)
	if err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 202, map[string]int64{"queued": tag.RowsAffected()})
}

func (s *Server) rebuildEmbeddings(w http.ResponseWriter, r *http.Request) {
	if s.Config.EmbeddingsURL == "" {
		fail(w, 503, "embeddings_unconfigured")
		return
	}
	var input struct {
		Model string `json:"model"`
	}
	if !decode(w, r, &input) {
		return
	}
	input.Model = strings.TrimSpace(input.Model)
	if input.Model == "" || len(input.Model) > 128 {
		fail(w, 400, "invalid_embedding_model")
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var candidate *int64
	if err = tx.QueryRow(r.Context(), `SELECT candidate_profile_id FROM embedding_index_state WHERE id=1 FOR UPDATE`).Scan(&candidate); err != nil {
		s.internal(w, err)
		return
	}
	if candidate != nil {
		var candidateState string
		if err = tx.QueryRow(r.Context(), `SELECT state FROM embedding_profiles WHERE id=$1`, *candidate).Scan(&candidateState); err != nil {
			s.internal(w, err)
			return
		}
		if candidateState != "failed" {
			fail(w, 409, "embedding_rebuild_in_progress")
			return
		}
		if _, err = tx.Exec(r.Context(), fmt.Sprintf(`DROP INDEX IF EXISTS transcript_embeddings_p%d_hnsw`, *candidate)); err != nil {
			s.internal(w, err)
			return
		}
		if _, err = tx.Exec(r.Context(), `DELETE FROM embedding_profiles WHERE id=$1`, *candidate); err != nil {
			s.internal(w, err)
			return
		}
		if _, err = tx.Exec(r.Context(), `UPDATE embedding_index_state SET candidate_profile_id=NULL WHERE id=1`); err != nil {
			s.internal(w, err)
			return
		}
	}
	var id int64
	if err = tx.QueryRow(r.Context(), `INSERT INTO embedding_profiles(model,state) VALUES($1,'building') RETURNING id`, input.Model).Scan(&id); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(r.Context(), `WITH latest AS (SELECT DISTINCT ON (clip_id) id FROM jobs WHERE status='done' AND step LIKE 'transcription.audio.%' ORDER BY clip_id,updated_at DESC,id DESC) INSERT INTO embedding_tasks(profile_id,transcription_job_id) SELECT $1,j.id FROM latest j ON CONFLICT(profile_id,transcription_job_id) DO NOTHING`, id); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE embedding_index_state SET candidate_profile_id=$1,updated_at=now() WHERE id=1`, id); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, 202, map[string]any{"candidate_profile_id": id, "model": input.Model})
}
