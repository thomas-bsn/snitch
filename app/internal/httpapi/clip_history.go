package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"snitch/internal/jobs"
)

// Clip history (ADR 0024, #136): per clip, the timeline of the events of every
// task of the clip, kept after the clip leaves the board. It is read from the
// task events, never grouped by run (that notion does not exist before #130).
// Voice trials are left out, as on the board.

const (
	clipHistoryDefaultSize = 10
	clipHistoryMaxSize     = 50
)

type clipHistoryStep struct {
	JobID       int64        `json:"job_id"`
	Step        string       `json:"step"`
	Status      string       `json:"status"`
	Attempts    int          `json:"attempts"`
	MaxAttempts int          `json:"max_attempts"`
	Cycle       int          `json:"cycle"`
	Worker      *string      `json:"worker"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Events      []jobs.Event `json:"events"`
}

type clipHistoryItem struct {
	ClipID       int64             `json:"clip_id"`
	OriginalName string            `json:"original_name"`
	LastActivity time.Time         `json:"last_activity_at"`
	Steps        []clipHistoryStep `json:"steps"`
}

type clipHistoryPage struct {
	Items    []clipHistoryItem `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// likePattern escapes the wildcards of a user search so it matches literally.
func likePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

func (s *Server) listClipHistory(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, size := 1, clipHistoryDefaultSize
	if raw := query.Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100000 {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
		page = n
	}
	if raw := query.Get("page_size"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > clipHistoryMaxSize {
			fail(w, http.StatusBadRequest, "invalid_page")
			return
		}
		size = n
	}
	search := strings.TrimSpace(query.Get("q"))
	if len(search) > 200 {
		fail(w, http.StatusBadRequest, "invalid_page")
		return
	}
	ctx := r.Context()
	out := clipHistoryPage{Items: []clipHistoryItem{}, Page: page, PageSize: size}
	rows, err := s.Pool.Query(ctx, `SELECT c.id,c.original_name,max(j.updated_at),count(*) OVER ()::int
FROM clips c JOIN jobs j ON j.clip_id=c.id AND j.step NOT LIKE 'voicetrial.%'
WHERE ($1='' OR c.original_name ILIKE $2 OR c.id::text=$1)
GROUP BY c.id ORDER BY max(j.updated_at) DESC,c.id DESC LIMIT $3 OFFSET $4`, search, likePattern(search), size, (page-1)*size)
	if err != nil {
		s.internal(w, err)
		return
	}
	index := map[int64]*clipHistoryItem{}
	var clipIDs []int64
	for rows.Next() {
		var item clipHistoryItem
		if err = rows.Scan(&item.ClipID, &item.OriginalName, &item.LastActivity, &out.Total); err != nil {
			rows.Close()
			s.internal(w, err)
			return
		}
		item.Steps = []clipHistoryStep{}
		out.Items = append(out.Items, item)
		clipIDs = append(clipIDs, item.ClipID)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	if len(out.Items) == 0 {
		reply(w, http.StatusOK, out)
		return
	}
	for i := range out.Items {
		index[out.Items[i].ClipID] = &out.Items[i]
	}
	jobRows, err := s.Pool.Query(ctx, `SELECT id,clip_id,step,status,attempts,max_attempts,cycle,
COALESCE(lease_owner,(SELECT e.actor_id FROM job_events e WHERE e.job_id=jobs.id AND e.actor='worker' AND e.actor_id<>'' ORDER BY e.id DESC LIMIT 1)),created_at,updated_at
FROM jobs WHERE clip_id=ANY($1) AND step NOT LIKE 'voicetrial.%' ORDER BY clip_id,id`, clipIDs)
	if err != nil {
		s.internal(w, err)
		return
	}
	stepOf := map[int64]*clipHistoryStep{}
	var jobIDs []int64
	for jobRows.Next() {
		var st clipHistoryStep
		var clipID int64
		if err = jobRows.Scan(&st.JobID, &clipID, &st.Step, &st.Status, &st.Attempts, &st.MaxAttempts, &st.Cycle, &st.Worker, &st.CreatedAt, &st.UpdatedAt); err != nil {
			jobRows.Close()
			s.internal(w, err)
			return
		}
		st.Events = []jobs.Event{}
		jobIDs = append(jobIDs, st.JobID)
		item := index[clipID]
		item.Steps = append(item.Steps, st)
	}
	jobRows.Close()
	if err = jobRows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	for _, item := range out.Items {
		for i := range item.Steps {
			stepOf[item.Steps[i].JobID] = &index[item.ClipID].Steps[i]
		}
	}
	eventRows, err := s.Pool.Query(ctx, `SELECT job_id,id,event,cycle,attempt,actor,actor_id,error_code,created_at,step,provider,model,revision,http_status,duration_ms FROM job_events WHERE job_id=ANY($1) ORDER BY job_id,id`, jobIDs)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer eventRows.Close()
	for eventRows.Next() {
		var jobID int64
		var e jobs.Event
		if err = eventRows.Scan(&jobID, &e.ID, &e.Event, &e.Cycle, &e.Attempt, &e.Actor, &e.ActorID, &e.ErrorCode, &e.CreatedAt, &e.Step, &e.Provider, &e.Model, &e.Revision, &e.HTTPStatus, &e.DurationMS); err != nil {
			s.internal(w, err)
			return
		}
		e.RecommendedAction = jobs.RecommendedAction(e.ErrorCode)
		st := stepOf[jobID]
		st.Events = append(st.Events, e)
	}
	if err = eventRows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, out)
}
