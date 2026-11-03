package httpapi

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The fine Hugging Face causes (#203) must be accepted by both cause columns; unknown codes stay refused.
func testHFFineCauseSchema(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO workers(id,name,token_hash) VALUES('wk_syntheticfinecause01','Synthetic worker',decode(repeat('ab',32),'hex'))`); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"hf_token_invalid", "hf_gated_denied", "hf_repo_invalid"} {
		if _, err := pool.Exec(ctx, `INSERT INTO worker_install_events(worker_id,phase,status,cause_code,occurred_at) VALUES('wk_syntheticfinecause01','voice-models','failed',$1,now())`, code); err != nil {
			t.Errorf("install event refused %s: %v", code, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO worker_voice_models(worker_id,model_id,state,cause_code) VALUES('wk_syntheticfinecause01','synthetic-model','failed',$1)`, code); err != nil {
			t.Errorf("voice model state refused %s: %v", code, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO worker_install_events(worker_id,phase,status,cause_code,occurred_at) VALUES('wk_syntheticfinecause01','voice-models','failed','hf_made_up',now())`); err == nil {
		t.Error("an unknown install cause must stay refused")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO worker_voice_models(worker_id,model_id,state,cause_code) VALUES('wk_syntheticfinecause01','synthetic-model','failed','hf_made_up')`); err == nil {
		t.Error("an unknown voice model cause must stay refused")
	}
}
