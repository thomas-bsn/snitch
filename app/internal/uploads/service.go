package uploads

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"snitch/internal/library"
)

var (
	ErrInvalid      = errors.New("invalid upload")
	ErrNotFound     = errors.New("upload not found")
	ErrTooLarge     = errors.New("upload exceeds configured limit")
	ErrConflict     = errors.New("upload state conflict")
	ErrExpired      = errors.New("upload expired")
	ErrStorage      = errors.New("upload storage unavailable")
	ErrInvalidVideo = errors.New("uploaded video is not readable")
	ErrHashRead     = errors.New("uploaded content could not be hashed")
)

type Config struct {
	Root      string
	ProbePath string
	MaxBytes  int64
	ChunkSize int64
	TTL       time.Duration
}

type Service struct {
	pool   *pgxpool.Pool
	config Config

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

type Session struct {
	ID               string    `json:"id"`
	OriginalName     string    `json:"original_name"`
	SizeBytes        int64     `json:"size_bytes"`
	OffsetBytes      int64     `json:"offset_bytes"`
	ChunkBytes       int64     `json:"chunk_size_bytes"`
	Visibility       string    `json:"visibility"`
	Status           string    `json:"status"`
	ClipID           *int64    `json:"clip_id,omitempty"`
	ExpiresAt        time.Time `json:"expires_at"`
	SHA256           string    `json:"sha256,omitempty"`
	Destination      string    `json:"destination,omitempty"`
	Publication      string    `json:"publication,omitempty"`
	PublicationError string    `json:"publication_error,omitempty"`
}

type Metadata interface {
	ProbeFile(context.Context, string, string) (library.Metadata, error)
}

func New(pool *pgxpool.Pool, config Config) *Service {
	if config.Root == "" {
		config.Root = "/data/media"
	}
	if config.ProbePath == "" {
		config.ProbePath = "ffprobe"
	}
	if config.MaxBytes <= 0 {
		config.MaxBytes = 20 << 30
	}
	if config.ChunkSize <= 0 {
		config.ChunkSize = 4 << 20
	}
	if config.TTL <= 0 {
		config.TTL = 7 * 24 * time.Hour
	}
	config.Root = filepath.Clean(config.Root)
	return &Service{pool: pool, config: config, done: make(chan struct{})}
}

func (s *Service) MaxBytes() int64  { return s.config.MaxBytes }
func (s *Service) ChunkSize() int64 { return s.config.ChunkSize }

func (s *Service) Start(ctx context.Context) error {
	root, err := filepath.Abs(filepath.Clean(s.config.Root))
	if err != nil {
		return ErrStorage
	}
	if err = os.MkdirAll(root, 0o700); err != nil {
		return ErrStorage
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ErrStorage
	}
	s.config.Root = root
	if err = s.recoverProcessing(ctx); err != nil {
		return err
	}
	if err = s.purgeExpired(ctx); err != nil {
		return err
	}
	loopCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	go s.cleanupLoop(loopCtx)
	return nil
}

func (s *Service) Close() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		<-s.done
	}
}

func (s *Service) cleanupLoop(ctx context.Context) {
	defer close(s.done)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.purgeExpired(ctx)
		}
	}
}

func (s *Service) Create(ctx context.Context, ownerID int64, name, visibility string, size int64) (Session, error) {
	if ownerID < 1 || size < 1 || !validVisibility(visibility) {
		return Session{}, ErrInvalid
	}
	if size > s.config.MaxBytes {
		return Session{}, ErrTooLarge
	}
	name, ok := cleanName(name)
	if !ok {
		return Session{}, ErrInvalid
	}
	for attempt := 0; attempt < 3; attempt++ {
		id, err := newID()
		if err != nil {
			return Session{}, err
		}
		file, err := os.OpenFile(s.partPath(id), os.O_CREATE|os.O_EXCL|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return Session{}, ErrStorage
		}
		if err = file.Close(); err != nil {
			_ = os.Remove(s.partPath(id))
			return Session{}, ErrStorage
		}
		var result Session
		err = s.pool.QueryRow(ctx, `INSERT INTO uploads(id,owner_id,original_name,total_bytes,visibility,status,expires_at) VALUES($1,$2,$3,$4,$5,'uploading',now()+$6::interval) RETURNING id,original_name,total_bytes,offset_bytes,visibility,status,expires_at`, id, ownerID, name, size, visibility, s.config.TTL.String()).Scan(&result.ID, &result.OriginalName, &result.SizeBytes, &result.OffsetBytes, &result.Visibility, &result.Status, &result.ExpiresAt)
		if err != nil {
			_ = os.Remove(s.partPath(id))
			return Session{}, err
		}
		result.ChunkBytes = s.config.ChunkSize
		return result, nil
	}
	return Session{}, ErrStorage
}

func (s *Service) List(ctx context.Context, ownerID int64) ([]Session, error) {
	if err := s.purgeExpired(ctx); err != nil {
		return nil, err
	}
	// Completed uploads whose named destination is not published stay visible:
	// otherwise a failed publication would be invisible and impossible to retry.
	rows, err := s.pool.Query(ctx, `SELECT u.id,u.original_name,u.total_bytes,u.offset_bytes,u.visibility,u.status,u.clip_id,u.expires_at,COALESCE(d.relative_path,''),COALESCE(d.published,false),COALESCE(d.last_error,'') FROM uploads u LEFT JOIN upload_destinations d ON d.upload_id=u.id WHERE u.owner_id=$1 AND (u.status IN ('uploading','processing','failed') OR (u.status='complete' AND d.upload_id IS NOT NULL AND NOT d.published)) ORDER BY u.created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Session, 0)
	for rows.Next() {
		var item Session
		var published bool
		if err = rows.Scan(&item.ID, &item.OriginalName, &item.SizeBytes, &item.OffsetBytes, &item.Visibility, &item.Status, &item.ClipID, &item.ExpiresAt, &item.Destination, &published, &item.PublicationError); err != nil {
			return nil, err
		}
		if item.Destination != "" {
			item.Publication = "pending"
			if published {
				item.Publication = "published"
			} else if item.PublicationError != "" {
				item.Publication = "failed"
			}
		}
		item.ChunkBytes = s.config.ChunkSize
		items = append(items, item)
	}
	return items, rows.Err()
}

// Chunk accepts the next contiguous byte range. A byte-identical retry is harmless.
func (s *Service) Chunk(ctx context.Context, ownerID int64, id string, start, end, total int64, data []byte) (int64, error) {
	if !validID(id) || ownerID < 1 || start < 0 || end < start || end == int64(^uint64(0)>>1) || total < 1 || int64(len(data)) != end-start+1 || int64(len(data)) > s.config.ChunkSize {
		return 0, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var session Session
	var expires time.Time
	err = tx.QueryRow(ctx, `SELECT id,original_name,total_bytes,offset_bytes,visibility,status,expires_at FROM uploads WHERE id=$1 AND owner_id=$2 FOR UPDATE`, id, ownerID).Scan(&session.ID, &session.OriginalName, &session.SizeBytes, &session.OffsetBytes, &session.Visibility, &session.Status, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if time.Now().After(expires) {
		return 0, ErrExpired
	}
	if session.Status != "uploading" || total != session.SizeBytes || end >= total {
		return 0, ErrConflict
	}
	if start < session.OffsetBytes {
		if end >= session.OffsetBytes {
			return 0, ErrConflict
		}
		file, openErr := s.openPart(id)
		if openErr != nil {
			return 0, ErrStorage
		}
		defer file.Close()
		previous := make([]byte, len(data))
		if _, openErr = file.ReadAt(previous, start); openErr != nil || !bytesEqual(previous, data) {
			return 0, ErrConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return 0, err
		}
		return session.OffsetBytes, nil
	}
	if start != session.OffsetBytes {
		return session.OffsetBytes, ErrConflict
	}
	file, err := s.openPart(id)
	if err != nil {
		return 0, ErrStorage
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return 0, ErrStorage
	}
	if info.Size() < session.OffsetBytes {
		return 0, ErrStorage
	}
	if info.Size() > session.OffsetBytes {
		if err = file.Truncate(session.OffsetBytes); err != nil {
			return 0, ErrStorage
		}
		if err = file.Sync(); err != nil {
			return 0, ErrStorage
		}
	}
	if _, err = file.WriteAt(data, session.OffsetBytes); err != nil {
		return 0, ErrStorage
	}
	if err = file.Sync(); err != nil {
		return 0, ErrStorage
	}
	newOffset := end + 1
	if _, err = tx.Exec(ctx, `UPDATE uploads SET offset_bytes=$2,updated_at=now() WHERE id=$1`, id, newOffset); err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return newOffset, nil
}

func (s *Service) Complete(ctx context.Context, ownerID int64, id string) (Session, error) {
	if !validID(id) || ownerID < 1 {
		return Session{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)
	var session Session
	var expires time.Time
	err = tx.QueryRow(ctx, `SELECT id,original_name,total_bytes,offset_bytes,visibility,status,clip_id,expires_at FROM uploads WHERE id=$1 AND owner_id=$2 FOR UPDATE`, id, ownerID).Scan(&session.ID, &session.OriginalName, &session.SizeBytes, &session.OffsetBytes, &session.Visibility, &session.Status, &session.ClipID, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	if session.Status == "complete" {
		var digest []byte
		if err = tx.QueryRow(ctx, `SELECT content_sha256 FROM clips WHERE id=$1`, *session.ClipID).Scan(&digest); err != nil {
			return Session{}, err
		}
		session.SHA256 = hex.EncodeToString(digest)
		session.ExpiresAt = expires
		session.ChunkBytes = s.config.ChunkSize
		if err = tx.Commit(ctx); err != nil {
			return Session{}, err
		}
		return session, nil
	}
	if time.Now().After(expires) {
		return Session{}, ErrExpired
	}
	if session.Status != "uploading" || session.OffsetBytes != session.SizeBytes {
		return Session{}, ErrConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE uploads SET status='processing',updated_at=now() WHERE id=$1`, id); err != nil {
		return Session{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	partPath, mediaPath := s.partPath(id), s.mediaPath(id)
	info, err := os.Stat(partPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() != session.SizeBytes {
		_ = s.failProcessing(ctx, id)
		return Session{}, ErrStorage
	}
	digest, err := library.HashFile(ctx, partPath)
	if err != nil {
		_ = s.failProcessing(ctx, id)
		return Session{}, ErrHashRead
	}
	metadata, err := library.ProbeFile(ctx, s.config.ProbePath, partPath)
	if err != nil {
		_ = s.failProcessing(ctx, id)
		_ = os.Remove(partPath)
		return Session{}, ErrInvalidVideo
	}
	if err = os.Rename(partPath, mediaPath); err != nil {
		_ = s.failProcessing(ctx, id)
		return Session{}, ErrStorage
	}
	tracks, err := jsonTracks(metadata.Tracks)
	if err != nil {
		_ = os.Rename(mediaPath, partPath)
		_ = s.resumeProcessing(ctx, id)
		return Session{}, err
	}
	finish, err := s.pool.Begin(ctx)
	if err != nil {
		_ = os.Rename(mediaPath, partPath)
		_ = s.resumeProcessing(ctx, id)
		return Session{}, err
	}
	defer finish.Rollback(ctx)
	var clipID, clipOwnerID int64
	var clipVisibility string
	err = finish.QueryRow(ctx, `INSERT INTO clips(owner_id,visibility,original_name,size_bytes,modified_at,recorded_at,recorded_at_estimated,duration_seconds,format_name,video_codec,audio_codec,tracks,content_sha256) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT(content_sha256) DO UPDATE SET content_sha256=EXCLUDED.content_sha256 RETURNING id,visibility,owner_id`, ownerID, session.Visibility, session.OriginalName, session.SizeBytes, info.ModTime(), metadata.RecordedAt, metadata.RecordedAtEstimated, metadata.Duration, metadata.FormatName, metadata.VideoCodec, metadata.AudioCodec, tracks, digest[:]).Scan(&clipID, &clipVisibility, &clipOwnerID)
	if err != nil {
		_ = os.Rename(mediaPath, partPath)
		_ = s.resumeProcessing(ctx, id)
		return Session{}, err
	}
	if (session.Visibility == "private" && (clipVisibility == "shared" || clipOwnerID != ownerID)) || (session.Visibility == "shared" && clipVisibility == "private") {
		_ = os.Rename(mediaPath, partPath)
		_ = s.resumeProcessing(ctx, id)
		return Session{}, ErrConflict
	}
	_, err = finish.Exec(ctx, `INSERT INTO uploaded_media(upload_id,clip_id,owner_id,storage_key,original_name) VALUES($1,$2,$3,$1,$4)`, id, clipID, ownerID, session.OriginalName)
	if err != nil {
		_ = os.Rename(mediaPath, partPath)
		_ = s.resumeProcessing(ctx, id)
		return Session{}, err
	}
	if _, err = finish.Exec(ctx, `UPDATE uploads SET status='complete',clip_id=$2,offset_bytes=total_bytes,updated_at=now() WHERE id=$1`, id, clipID); err != nil {
		_ = os.Rename(mediaPath, partPath)
		_ = s.resumeProcessing(ctx, id)
		return Session{}, err
	}
	if err = finish.Commit(ctx); err != nil {
		// If commit outcome is ambiguous, leave the durable media file in place; startup
		// recovery checks the committed status before deciding whether to resume it.
		return Session{}, err
	}
	var recordedHash []byte
	if err = s.pool.QueryRow(ctx, `SELECT content_sha256 FROM clips WHERE id=$1`, clipID).Scan(&recordedHash); err != nil {
		return Session{}, err
	}
	session.Status, session.ClipID, session.SHA256 = "complete", &clipID, hex.EncodeToString(recordedHash)
	session.ChunkBytes = s.config.ChunkSize
	session.ExpiresAt = expires
	return session, nil
}

func (s *Service) Cancel(ctx context.Context, ownerID int64, id string) error {
	if !validID(id) || ownerID < 1 {
		return ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM uploads WHERE id=$1 AND owner_id=$2 FOR UPDATE`, id, ownerID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status == "complete" {
		// A completed upload whose named destination never got published can be
		// abandoned: this is the only way to release its staging copy and intent.
		var published bool
		err = tx.QueryRow(ctx, `SELECT published FROM upload_destinations WHERE upload_id=$1 FOR UPDATE`, id).Scan(&published)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return err
		}
		if published {
			return ErrConflict
		}
		var clipID *int64
		if err = tx.QueryRow(ctx, `SELECT clip_id FROM uploads WHERE id=$1`, id).Scan(&clipID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM uploaded_media WHERE upload_id=$1`, id); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM upload_destinations WHERE upload_id=$1`, id); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE uploads SET status='cancelled',clip_id=NULL,updated_at=now() WHERE id=$1`, id); err != nil {
			return err
		}
		if clipID != nil {
			if _, err = tx.Exec(ctx, `DELETE FROM clips WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM source_files WHERE clip_id=$1) AND NOT EXISTS(SELECT 1 FROM uploaded_media WHERE clip_id=$1) AND NOT EXISTS(SELECT 1 FROM uploads WHERE clip_id=$1)`, *clipID); err != nil {
				return err
			}
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		_ = os.Remove(s.mediaPath(id))
		return nil
	}
	if status == "processing" {
		return ErrConflict
	}
	// An abandoned intent must not keep blocking a root change.
	if _, err = tx.Exec(ctx, `DELETE FROM upload_destinations WHERE upload_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE uploads SET status='cancelled',updated_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	_ = os.Remove(s.partPath(id))
	return nil
}

func (s *Service) failProcessing(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE uploads SET status='failed',updated_at=now() WHERE id=$1 AND status='processing'`, id)
	return err
}

func (s *Service) resumeProcessing(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE uploads SET status='uploading',updated_at=now() WHERE id=$1 AND status='processing'`, id)
	return err
}

func (s *Service) recoverProcessing(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT id FROM uploads WHERE status='processing'`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, id := range ids {
		part, media := s.partPath(id), s.mediaPath(id)
		if _, err = os.Stat(media); err == nil {
			if _, partErr := os.Stat(part); os.IsNotExist(partErr) {
				if err = os.Rename(media, part); err != nil {
					return ErrStorage
				}
			} else if partErr == nil {
				_ = os.Remove(media)
			} else {
				return ErrStorage
			}
		}
		info, statErr := os.Stat(part)
		if statErr != nil {
			_, err = s.pool.Exec(ctx, `UPDATE uploads SET status='failed',updated_at=now() WHERE id=$1 AND status='processing'`, id)
			if err != nil {
				return err
			}
			continue
		}
		_, err = s.pool.Exec(ctx, `UPDATE uploads SET status='uploading',offset_bytes=LEAST(total_bytes,$2),updated_at=now() WHERE id=$1 AND status='processing'`, id, info.Size())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) purgeExpired(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT id FROM uploads WHERE status='uploading' AND expires_at<=now()`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, id := range ids {
		tx, beginErr := s.pool.Begin(ctx)
		if beginErr != nil {
			return beginErr
		}
		var locked string
		beginErr = tx.QueryRow(ctx, `SELECT id FROM uploads WHERE id=$1 AND status='uploading' AND expires_at<=now() FOR UPDATE`, id).Scan(&locked)
		if errors.Is(beginErr, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			continue
		}
		if beginErr != nil {
			_ = tx.Rollback(ctx)
			return beginErr
		}
		if _, beginErr = tx.Exec(ctx, `UPDATE uploads SET status='expired',updated_at=now() WHERE id=$1`, id); beginErr != nil {
			_ = tx.Rollback(ctx)
			return beginErr
		}
		if _, beginErr = tx.Exec(ctx, `DELETE FROM upload_destinations WHERE upload_id=$1`, id); beginErr != nil {
			_ = tx.Rollback(ctx)
			return beginErr
		}
		if beginErr = tx.Commit(ctx); beginErr != nil {
			return beginErr
		}
		_ = os.Remove(s.partPath(id))
	}
	return nil
}

func (s *Service) openPart(id string) (*os.File, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	path := s.partPath(id)
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, ErrStorage
	}
	file, err := os.OpenFile(path, os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, ErrStorage
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) || !after.Mode().IsRegular() {
		_ = file.Close()
		return nil, ErrStorage
	}
	return file, nil
}

func (s *Service) partPath(id string) string  { return filepath.Join(s.config.Root, id+".part") }
func (s *Service) mediaPath(id string) string { return filepath.Join(s.config.Root, id+".media") }

func validVisibility(value string) bool { return value == "private" || value == "shared" }

func cleanName(name string) (string, bool) {
	if !utf8.ValidString(name) || len(name) == 0 || len(name) > 1024 {
		return "", false
	}
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(filepath.FromSlash(name))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", false
	}
	for len(name) > 255 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return name, true
}

func newID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func validID(id string) bool {
	if len(id) != 32 {
		return false
	}
	decoded, err := hex.DecodeString(id)
	return err == nil && len(decoded) == 16
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func ParseContentRange(value string) (start, end, total int64, err error) {
	if !strings.HasPrefix(value, "bytes ") {
		return 0, 0, 0, ErrInvalid
	}
	parts := strings.Split(strings.TrimPrefix(value, "bytes "), "/")
	if len(parts) != 2 {
		return 0, 0, 0, ErrInvalid
	}
	rangeParts := strings.Split(parts[0], "-")
	if len(rangeParts) != 2 {
		return 0, 0, 0, ErrInvalid
	}
	start, err = strconv.ParseInt(rangeParts[0], 10, 64)
	if err != nil {
		return 0, 0, 0, ErrInvalid
	}
	end, err = strconv.ParseInt(rangeParts[1], 10, 64)
	if err != nil {
		return 0, 0, 0, ErrInvalid
	}
	total, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil || start < 0 || end < start || total <= end {
		return 0, 0, 0, ErrInvalid
	}
	return start, end, total, nil
}

func jsonTracks(tracks []library.Track) ([]byte, error) {
	if tracks == nil {
		tracks = []library.Track{}
	}
	return json.Marshal(tracks)
}

// MediaRoot returns the configured root containing completed media. Callers must
// resolve storage keys from the database rather than accept client paths.
func (s *Service) MediaRoot() string { return s.config.Root }
