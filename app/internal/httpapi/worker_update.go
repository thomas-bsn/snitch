package httpapi

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"snitch/internal/config"
	"snitch/internal/workerupdate"

	"github.com/jackc/pgx/v5"
)

// Automatic Worker updates (ADR 0036). Administrators choose the target version; each Worker's
// local updater reads it with its durable token and reports what it did.

var (
	workerUpdateStates = map[string]bool{"idle": true, "pulling": true, "restarting": true, "updated": true, "rolled_back": true, "failed": true}
	workerCauseCode    = regexp.MustCompile(`^[a-z_]{0,40}$`)
	workerDigest       = regexp.MustCompile(`^(sha256:[a-f0-9]{64})?$`)
)

// workerCatalogFor returns the registry catalogue of the configured Worker image, built once.
func (s *Server) workerCatalogFor() *workerupdate.Catalog {
	if s.workerCatalog != nil {
		return s.workerCatalog // injected by tests, never set in production
	}
	s.workerCatalogOnce.Do(func() {
		registry, repo, err := workerupdate.RegistryOf(s.workerImage())
		if err != nil {
			s.Log.Error("worker update catalogue disabled: invalid worker image")
			return
		}
		s.workerCatalog = &workerupdate.Catalog{Client: &http.Client{Timeout: 10 * time.Second}, Registry: registry, Repo: repo}
	})
	return s.workerCatalog
}

func (s *Server) workerImage() string {
	if s.Config.WorkerImage == "" {
		return config.DefaultWorkerImage
	}
	return s.Config.WorkerImage
}

// targetVersion reads the administrator's choice; an empty string means no automatic update.
func (s *Server) targetVersion(ctx context.Context) (string, error) {
	var version *string
	err := s.Pool.QueryRow(ctx, `SELECT version FROM worker_update_target WHERE id=true`).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if version == nil {
		return "", nil
	}
	return *version, nil
}

func (s *Server) getWorkerUpdates(w http.ResponseWriter, r *http.Request) {
	target, err := s.targetVersion(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	image := ""
	if target != "" {
		if image, err = workerupdate.ImageFor(s.workerImage(), target); err != nil {
			s.internal(w, err)
			return
		}
	}
	catalog := []string{}
	var catalogErr error = workerupdate.ErrCatalog
	if c := s.workerCatalogFor(); c != nil {
		catalog, catalogErr = c.Versions(r.Context())
	}
	rows, err := s.Pool.Query(r.Context(), `SELECT w.id,w.name,COALESCE(s.state,''),COALESCE(s.version,''),COALESCE(s.digest,''),COALESCE(s.cause_code,''),s.reported_at
		FROM workers w LEFT JOIN worker_update_status s ON s.worker_id=w.id WHERE w.revoked_at IS NULL ORDER BY w.name,w.id`)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	type workerState struct {
		ID         string     `json:"id"`
		Name       string     `json:"name"`
		State      string     `json:"state"`
		Version    string     `json:"version"`
		Digest     string     `json:"digest"`
		CauseCode  string     `json:"cause_code"`
		ReportedAt *time.Time `json:"reported_at"`
	}
	items := []workerState{}
	for rows.Next() {
		var item workerState
		if err = rows.Scan(&item.ID, &item.Name, &item.State, &item.Version, &item.Digest, &item.CauseCode, &item.ReportedAt); err != nil {
			s.internal(w, err)
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{
		"target_version":    nilIfEmpty(target),
		"target_image":      nilIfEmpty(image),
		"image_repository":  repositoryOf(s.workerImage()),
		"catalog":           catalog,
		"catalog_available": catalogErr == nil,
		"workers":           items,
	})
}

func (s *Server) putWorkerUpdateTarget(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Version *string `json:"version"`
	}
	if !decode(w, r, &input) {
		return
	}
	version := ""
	if input.Version != nil {
		version = strings.TrimSpace(*input.Version)
	}
	var stored *string
	if version != "" {
		// Format is enforced here, and the image is always built from the configured repository.
		if _, err := workerupdate.ImageFor(s.workerImage(), version); err != nil {
			fail(w, http.StatusBadRequest, "invalid_worker_version")
			return
		}
		stored = &version
	}
	if _, err := s.Pool.Exec(r.Context(), `UPDATE worker_update_target SET version=$1,updated_at=now() WHERE id=true`, stored); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"target_version": nilIfEmpty(version)})
}

// getWorkerUpdate tells a Worker's local updater which image it should run. It changes nothing.
func (s *Server) getWorkerUpdate(w http.ResponseWriter, r *http.Request) {
	target, err := s.targetVersion(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	image := ""
	if target != "" {
		if image, err = workerupdate.ImageFor(s.workerImage(), target); err != nil {
			s.internal(w, err)
			return
		}
	}
	reply(w, http.StatusOK, map[string]any{"target_version": nilIfEmpty(target), "target_image": nilIfEmpty(image)})
}

func (s *Server) postWorkerUpdateStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		State     string `json:"state"`
		Version   string `json:"version"`
		Digest    string `json:"digest"`
		CauseCode string `json:"cause_code"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !workerUpdateStates[input.State] || !workerupdate.ValidVersion(input.Version) || !workerDigest.MatchString(input.Digest) || !workerCauseCode.MatchString(input.CauseCode) {
		fail(w, http.StatusBadRequest, "invalid_worker_update_status")
		return
	}
	worker := currentWorker(r)
	_, err := s.Pool.Exec(r.Context(), `INSERT INTO worker_update_status(worker_id,state,version,digest,cause_code,reported_at) VALUES($1,$2,$3,$4,$5,now())
		ON CONFLICT(worker_id) DO UPDATE SET state=EXCLUDED.state,version=EXCLUDED.version,digest=EXCLUDED.digest,cause_code=EXCLUDED.cause_code,reported_at=now()`,
		worker.ID, input.State, input.Version, input.Digest, input.CauseCode)
	if err != nil {
		s.internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func repositoryOf(image string) string {
	if i := strings.LastIndex(image, ":"); i > strings.LastIndex(image, "/") {
		return image[:i]
	}
	return image
}
