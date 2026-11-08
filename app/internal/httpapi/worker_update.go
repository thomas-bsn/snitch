package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	// Written only inside Do and read only inside Do: no unsynchronised access.
	s.workerCatalogOnce.Do(func() {
		if s.workerCatalogOverride != nil { // injected by tests, never set in production
			s.workerCatalogBuilt = s.workerCatalogOverride
			return
		}
		registry, repo, err := workerupdate.RegistryOf(s.workerImage())
		if err != nil {
			s.Log.Error("worker update catalogue disabled: invalid worker image")
			return
		}
		s.workerCatalogBuilt = &workerupdate.Catalog{Client: &http.Client{Timeout: 10 * time.Second}, Registry: registry, Repo: repo}
	})
	return s.workerCatalogBuilt
}

func (s *Server) workerImage() string {
	if s.Config.WorkerImage == "" {
		return config.DefaultWorkerImage
	}
	return s.Config.WorkerImage
}

// updateTarget is the stored strategy: the mode and the effective version (empty when none).
type updateTarget struct {
	Mode    string
	Version string
}

func (s *Server) readUpdateTarget(ctx context.Context) (updateTarget, error) {
	var version *string
	t := updateTarget{Mode: "disabled"}
	err := s.Pool.QueryRow(ctx, `SELECT mode,version FROM worker_update_target WHERE id=true`).Scan(&t.Mode, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return updateTarget{Mode: "disabled"}, nil
	}
	if err != nil {
		return t, err
	}
	if version != nil {
		t.Version = *version
	}
	return t, nil
}

// effectiveTarget returns the strategy with its version brought up to date. In latest mode the newest published
// release is read from the cached catalogue (no extra registry call) and stored only when it is strictly higher than
// the stored one: the database is untouched by polls that find nothing new, a registry outage keeps the last valid
// target, and the version can never go down. The conditional UPDATE makes concurrent resolutions idempotent.
func (s *Server) effectiveTarget(ctx context.Context) (updateTarget, error) {
	t, err := s.readUpdateTarget(ctx)
	if err != nil || t.Mode != "latest" {
		return t, err
	}
	c := s.workerCatalogFor()
	if c == nil {
		return t, nil
	}
	versions, _ := c.Versions(ctx)
	newest := workerupdate.Newest(versions)
	if newest == "" || (t.Version != "" && !workerupdate.Less(t.Version, newest)) {
		return t, nil
	}
	var old *string
	if t.Version != "" {
		old = &t.Version
	}
	if _, err = s.Pool.Exec(ctx, `UPDATE worker_update_target SET version=$1,updated_at=now() WHERE id=true AND mode='latest' AND version IS NOT DISTINCT FROM $2`, newest, old); err != nil {
		return t, err
	}
	// A concurrent change (another resolution, or the administrator) may have won: report what is stored now.
	return s.readUpdateTarget(ctx)
}

func (s *Server) getWorkerUpdates(w http.ResponseWriter, r *http.Request) {
	current, err := s.effectiveTarget(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	target := current.Version
	image := ""
	if target != "" {
		if image, err = workerupdate.ImageFor(s.workerImage(), target); err != nil {
			s.internal(w, err)
			return
		}
	}
	catalog := []string{}
	var checkedAt *time.Time
	checkFailed := false
	var catalogErr error = workerupdate.ErrCatalog
	if c := s.workerCatalogFor(); c != nil {
		catalog, catalogErr = c.Versions(r.Context())
		if at, failed := c.Status(); !at.IsZero() {
			checkedAt, checkFailed = &at, failed
		}
	}
	if catalog == nil {
		// The contract promises an array: an empty catalogue must not reach the screen as null.
		catalog = []string{}
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
		"mode":              current.Mode,
		"target_version":    nilIfEmpty(target),
		"target_image":      nilIfEmpty(image),
		"latest_version":    nilIfEmpty(workerupdate.Newest(catalog)),
		"checked_at":        checkedAt,
		"check_failed":      checkFailed,
		"image_repository":  repositoryOf(s.workerImage()),
		"catalog":           catalog,
		"catalog_available": catalogErr == nil,
		"workers":           items,
	})
}

func (s *Server) putWorkerUpdateTarget(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Mode    *string `json:"mode"`
		Version *string `json:"version"`
	}
	if !decode(w, r, &input) {
		return
	}
	version := ""
	if input.Version != nil {
		version = strings.TrimSpace(*input.Version)
	}
	mode := "disabled"
	if version != "" {
		mode = "pinned"
	}
	if input.Mode != nil {
		mode = *input.Mode
	}
	var stored *string
	switch mode {
	case "disabled":
		version = ""
	case "pinned":
		if version == "" {
			fail(w, http.StatusBadRequest, "invalid_worker_version")
			return
		}
	case "latest":
		// The version field is ignored: the target follows the registry. Keep the current one as a floor so that
		// switching to latest never lowers what the fleet is told to run.
		current, err := s.readUpdateTarget(r.Context())
		if err != nil {
			s.internal(w, err)
			return
		}
		version = current.Version
	default:
		fail(w, http.StatusBadRequest, "invalid_worker_update_mode")
		return
	}
	if version != "" {
		// Format is enforced here, and the image is always built from the configured repository.
		if _, err := workerupdate.ImageFor(s.workerImage(), version); err != nil {
			fail(w, http.StatusBadRequest, "invalid_worker_version")
			return
		}
		stored = &version
	}
	if _, err := s.Pool.Exec(r.Context(), `UPDATE worker_update_target SET mode=$1,version=$2,updated_at=now() WHERE id=true`, mode, stored); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"mode": mode, "target_version": nilIfEmpty(version)})
}

// getWorkerUpdate tells a Worker's local updater which image it should run, and whether it is idle (no job leased),
// so that the updater can wait for a quiet moment. In latest mode it resolves the target from the cached catalogue.
func (s *Server) getWorkerUpdate(w http.ResponseWriter, r *http.Request) {
	current, err := s.effectiveTarget(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	target := current.Version
	image := ""
	if target != "" {
		if image, err = workerupdate.ImageFor(s.workerImage(), target); err != nil {
			s.internal(w, err)
			return
		}
	}
	var busy bool
	if err = s.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM jobs WHERE lease_owner=$1 AND status='leased' AND lease_expires_at>now())`, currentWorker(r).ID).Scan(&busy); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"target_version": nilIfEmpty(target), "target_image": nilIfEmpty(image), "idle": !busy, "updater_sha256": workerUpdaterSHA256})
}

// workerUpdaterSHA256 identifies the updater script embedded in this build. Workers compare it with their own copy
// and fetch the script from getWorkerUpdater when it differs, so a backend upgrade also upgrades installed updaters.
var workerUpdaterSHA256 = func() string {
	sum := sha256.Sum256([]byte(workerUpdaterScript))
	return hex.EncodeToString(sum[:])
}()

// getWorkerUpdater serves the embedded updater script to an authenticated Worker. It holds no secret.
func (s *Server) getWorkerUpdater(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(workerUpdaterScript))
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
