package httpapi

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"snitch/internal/library"
)

// Demandes de création de dossier (ADR 0033, #225). A member asks for a folder of the Racine vidéo
// and waits; an Administrator creates it at once, and approves or refuses a member's request. The
// requester is notified of the decision; nothing can be uploaded into a pending folder. Requests
// under a Dossier privé (ADR 0026) are reserved for the Super-administrator, as for the folder itself.

const kindFolderRequestDecided = "folder_request_decided"

type folderRequestView struct {
	ID          int64      `json:"id"`
	Parent      string     `json:"parent"`
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	Status      string     `json:"status"`
	Requester   string     `json:"requester"`
	CreatedAt   time.Time  `json:"created_at"`
	DecidedAt   *time.Time `json:"decided_at"`
	DecidedBy   string     `json:"decided_by"`
	RequesterID *int64     `json:"requester_id"`
}

// folderRelative normalizes a root-relative folder: "" is the Racine vidéo, no leading or trailing slash.
func folderRelative(folder string) string {
	return strings.Trim(filepath.ToSlash(filepath.Clean("/"+folder)), "/")
}

// validFolderName is the rule for a folder name, shared by direct creation and requests.
func validFolderName(name string) bool {
	return strings.TrimSpace(name) != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00") && !strings.HasPrefix(name, ".") && len(name) <= 255
}

// createFolderAt creates one folder under a parent of the root through an os.Root, which also confines
// the write if a local process turns a directory into a symlink.
func createFolderAt(rootPath, parent, name string) error {
	handle, err := os.OpenRoot(rootPath)
	if err != nil {
		return err
	}
	defer handle.Close()
	return handle.Mkdir(filepath.Join(parent, name), 0o750)
}

// pendingFolderRequestAt reports whether the folder is, or lies under, a folder that waits for a decision.
func (s *Server) pendingFolderRequestAt(ctx context.Context, folder string) (bool, error) {
	rows, err := s.Pool.Query(ctx, `SELECT parent,name FROM folder_requests WHERE status='pending'`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var parent, name string
		if err := rows.Scan(&parent, &name); err != nil {
			return false, err
		}
		target := folderRelative(parent + "/" + name)
		if folder == target || strings.HasPrefix(folder, target+"/") {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *Server) createFolderRequest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Parent string `json:"parent"`
		Name   string `json:"name"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !validFolderName(input.Name) {
		fail(w, http.StatusBadRequest, "invalid_folder_name")
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	root, err := s.Library.ManagedRoot(r.Context())
	if err != nil {
		fail(w, http.StatusConflict, "video_root_unconfigured")
		return
	}
	parent := folderRelative(input.Parent)
	absolute, err := library.ManagedPath(root.Path, parent)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, "video_path_unavailable")
		return
	}
	if st, e := os.Stat(absolute); e != nil || !st.IsDir() {
		fail(w, http.StatusUnprocessableEntity, "video_path_unavailable")
		return
	}
	superAdmin := isSuperAdministrator(ss.User.Role)
	private, err := s.privateTopFolders(r, root.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !superAdmin && private[firstSegment(parent)] {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	target := folderRelative(parent + "/" + input.Name)
	if _, err := os.Lstat(filepath.Join(root.Path, target)); err == nil {
		fail(w, http.StatusConflict, "folder_exists")
		return
	} else if !os.IsNotExist(err) {
		s.internal(w, err)
		return
	}
	if isAdministrator(ss.User.Role) {
		if err := createFolderAt(root.Path, parent, input.Name); err != nil {
			fail(w, http.StatusConflict, "folder_not_created")
			return
		}
		reply(w, http.StatusCreated, map[string]string{"status": "created", "path": target})
		return
	}
	ctx := r.Context()
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO folder_requests(parent,name,requested_by,requester_username) VALUES($1,$2,$3,$4) RETURNING id`, parent, input.Name, ss.User.ID, ss.User.Username).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail(w, http.StatusConflict, "folder_request_pending")
			return
		}
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO folder_request_events(action,parent,name,requester_id,requester_username,actor_id,actor_username) VALUES('request',$1,$2,$3,$4,$3,$4)`, parent, input.Name, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusAccepted, folderRequestView{ID: id, Parent: parent, Name: input.Name, Path: target, Status: "pending", Requester: ss.User.Username, CreatedAt: time.Now().UTC(), RequesterID: &ss.User.ID})
}

// listFolderRequests returns the pending requests the caller may see: an Administrator sees all of
// them outside private folders, anyone else only their own.
func (s *Server) listFolderRequests(w http.ResponseWriter, r *http.Request) {
	ss := r.Context().Value(sessionKey{}).(session)
	root, err := s.Library.ManagedRoot(r.Context())
	if errors.Is(err, pgx.ErrNoRows) {
		reply(w, http.StatusOK, map[string]any{"items": []folderRequestView{}})
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	private, err := s.privateTopFolders(r, root.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	superAdmin := isSuperAdministrator(ss.User.Role)
	query := `SELECT id,parent,name,status,requested_by,requester_username,created_at,decided_at,decided_by_username FROM folder_requests WHERE status='pending'`
	args := []any{}
	if !isAdministrator(ss.User.Role) {
		query += ` AND requested_by=$1`
		args = append(args, ss.User.ID)
	}
	rows, err := s.Pool.Query(r.Context(), query+` ORDER BY created_at,id LIMIT 500`, args...)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer rows.Close()
	items := []folderRequestView{}
	for rows.Next() {
		var v folderRequestView
		if err := rows.Scan(&v.ID, &v.Parent, &v.Name, &v.Status, &v.RequesterID, &v.Requester, &v.CreatedAt, &v.DecidedAt, &v.DecidedBy); err != nil {
			s.internal(w, err)
			return
		}
		if !superAdmin && private[firstSegment(v.Parent)] {
			continue
		}
		v.Path = folderRelative(v.Parent + "/" + v.Name)
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) approveFolderRequest(w http.ResponseWriter, r *http.Request) {
	s.decideFolderRequest(w, r, true)
}

func (s *Server) refuseFolderRequest(w http.ResponseWriter, r *http.Request) {
	s.decideFolderRequest(w, r, false)
}

// decideFolderRequest records the Administrator's decision, creates the folder when approved, and tells
// the requester. The folder and the decision are committed together.
func (s *Server) decideFolderRequest(w http.ResponseWriter, r *http.Request, approve bool) {
	id, ok := pathID(w, r, "invalid_id")
	if !ok {
		return
	}
	ss := r.Context().Value(sessionKey{}).(session)
	ctx := r.Context()
	root, err := s.Library.ManagedRoot(ctx)
	if err != nil {
		fail(w, http.StatusConflict, "video_root_unconfigured")
		return
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(ctx)
	var parent, name, status, requester string
	var requesterID *int64
	err = tx.QueryRow(ctx, `SELECT parent,name,status,requested_by,requester_username FROM folder_requests WHERE id=$1 FOR UPDATE`, id).Scan(&parent, &name, &status, &requesterID, &requester)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, http.StatusNotFound, "folder_request_not_found")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	if status != "pending" {
		fail(w, http.StatusConflict, "request_not_pending")
		return
	}
	private, err := s.privateTopFolders(r, root.ID)
	if err != nil {
		s.internal(w, err)
		return
	}
	if !isSuperAdministrator(ss.User.Role) && private[firstSegment(parent)] {
		fail(w, http.StatusForbidden, "forbidden")
		return
	}
	target := folderRelative(parent + "/" + name)
	outcome, action := "refused", "refuse"
	if approve {
		outcome, action = "approved", "approve"
		if _, err := os.Lstat(filepath.Join(root.Path, target)); err == nil {
			fail(w, http.StatusConflict, "folder_exists")
			return
		}
		if err := createFolderAt(root.Path, parent, name); err != nil {
			fail(w, http.StatusConflict, "folder_not_created")
			return
		}
	}
	var createdAt, decidedAt time.Time
	if err = tx.QueryRow(ctx, `UPDATE folder_requests SET status=$2,decided_at=now(),decided_by=$3,decided_by_username=$4 WHERE id=$1 RETURNING created_at,decided_at`, id, outcome, ss.User.ID, ss.User.Username).Scan(&createdAt, &decidedAt); err != nil {
		s.internal(w, err)
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO folder_request_events(action,parent,name,requester_id,requester_username,actor_id,actor_username) VALUES($1,$2,$3,$4,$5,$6,$7)`, action, parent, name, requesterID, requester, ss.User.ID, ss.User.Username); err != nil {
		s.internal(w, err)
		return
	}
	if requesterID != nil {
		var notification int64
		if err = tx.QueryRow(ctx, `INSERT INTO notifications(kind,state,resolved_at,folder_path,folder_outcome) VALUES($1,'resolved',now(),$2,$3) RETURNING id`, kindFolderRequestDecided, target, outcome).Scan(&notification); err != nil {
			s.internal(w, err)
			return
		}
		if _, err = tx.Exec(ctx, `INSERT INTO notification_recipients(notification_id,user_id) VALUES($1,$2)`, notification, *requesterID); err != nil {
			s.internal(w, err)
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		s.internal(w, err)
		return
	}
	reply(w, http.StatusOK, folderRequestView{ID: id, Parent: parent, Name: name, Path: target, Status: outcome, Requester: requester, RequesterID: requesterID, DecidedBy: ss.User.Username, CreatedAt: createdAt, DecidedAt: &decidedAt})
}
