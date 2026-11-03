package httpapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The fine Hugging Face causes (#203) must be accepted by both cause columns; unknown codes stay refused.
func testHFFineCauseSchema(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	for i, code := range []string{"hf_token_invalid", "hf_gated_denied", "hf_repo_invalid"} {
		// One worker per cause: worker_voice_models has (worker_id, model_id) as primary key.
		worker := fmt.Sprintf("wk_syntheticfinecause%02d", i)
		if _, err := pool.Exec(ctx, `INSERT INTO workers(id,name,token_hash) VALUES($1,'Synthetic worker',decode(repeat($2,32),'hex'))`, worker, fmt.Sprintf("%02x", i+1)); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO worker_install_events(worker_id,phase,status,cause_code,occurred_at) VALUES($1,'voice-models','failed',$2,now())`, worker, code); err != nil {
			t.Errorf("install event refused %s: %v", code, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO worker_voice_models(worker_id,model_id,state,cause_code) VALUES($1,'synthetic-model','failed',$2)`, worker, code); err != nil {
			t.Errorf("voice model state refused %s: %v", code, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO worker_install_events(worker_id,phase,status,cause_code,occurred_at) VALUES('wk_syntheticfinecause00','voice-models','failed','hf_made_up',now())`); err == nil {
		t.Error("an unknown install cause must stay refused")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO worker_voice_models(worker_id,model_id,state,cause_code) VALUES('wk_syntheticfinecause00','synthetic-model','failed','hf_made_up')`); err == nil {
		t.Error("an unknown voice model cause must stay refused")
	}
}
