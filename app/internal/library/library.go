package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidSource = errors.New("invalid source")
	ErrSourceExists  = errors.New("source already exists")
	ErrSourceBusy    = errors.New("source has pending uploads")
	ErrNotFound      = errors.New("source not found")
	ErrUnavailable   = errors.New("source unavailable")
)

// ManagedScanError reports, with a stable code, why a ready managed file could
// not be handed to the transcription scheduler.
type ManagedScanError struct {
	Code string
	Err  error
}

func (e *ManagedScanError) Error() string {
	if e.Err == nil {
		return e.Code
	}
	return e.Code + ": " + e.Err.Error()
}

func (e *ManagedScanError) Unwrap() error { return e.Err }

type Config struct {
	Roots          []string
	ProbePath      string
	Stability      time.Duration
	ReconcileEvery time.Duration
	Exclude        []string
}

type Service struct {
	pool   *pgxpool.Pool
	config Config
	logger interface{ Warn(string, ...any) }

	mu             sync.Mutex
	watcher        *fsnotify.Watcher
	watchCancel    context.CancelFunc
	ctx            context.Context
	watched        map[string]int64
	roots          map[int64]string
	timers         map[int64]*time.Timer
	closed         bool
	locks          sync.Map
	OnManagedReady func(context.Context, int64) error
}

type Source struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Path          string     `json:"path"`
	OwnerID       int64      `json:"-"`
	ManagedRoot   bool       `json:"managed_root"`
	Visibility    string     `json:"visibility"`
	Enabled       bool       `json:"enabled"`
	Health        string     `json:"health"`
	HealthMessage string     `json:"health_message,omitempty"`
	LastScan      *time.Time `json:"last_scan_at"`
}

type Track struct {
	Index         int    `json:"index"`
	Type          string `json:"type"`
	Codec         string `json:"codec"`
	Title         string `json:"title,omitempty"`
	Language      string `json:"language,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	ChannelLayout string `json:"channel_layout,omitempty"`
}

type File struct {
	Name                string     `json:"name"`
	RelativePath        string     `json:"relative_path"`
	Status              string     `json:"status"`
	ClipID              *int64     `json:"clip_id,omitempty"`
	ErrorCode           string     `json:"error_code,omitempty"`
	ScanRequested       bool       `json:"scan_requested"`
	SizeBytes           int64      `json:"size_bytes"`
	ModifiedAt          time.Time  `json:"modified_at"`
	DiscoveredAt        *time.Time `json:"discovered_at,omitempty"`
	RecordedAt          *time.Time `json:"recorded_at,omitempty"`
	RecordedAtEstimated bool       `json:"recorded_at_estimated"`
	DurationSeconds     *float64   `json:"duration_seconds,omitempty"`
	FormatName          string     `json:"format_name,omitempty"`
	VideoCodec          string     `json:"video_codec,omitempty"`
	AudioCodec          string     `json:"audio_codec,omitempty"`
	Tracks              []Track    `json:"tracks"`
}

type probeResult struct {
	Format struct {
		Duration   string            `json:"duration"`
		FormatName string            `json:"format_name"`
		Tags       map[string]string `json:"tags"`
	} `json:"format"`
	Streams []struct {
		Index         int               `json:"index"`
		Type          string            `json:"codec_type"`
		Codec         string            `json:"codec_name"`
		Width         int               `json:"width"`
		Height        int               `json:"height"`
		Channels      int               `json:"channels"`
		ChannelLayout string            `json:"channel_layout"`
		Tags          map[string]string `json:"tags"`
	} `json:"streams"`
}

type Metadata struct {
	RecordedAt          time.Time
	RecordedAtEstimated bool
	Duration            *float64
	FormatName          string
	VideoCodec          string
	AudioCodec          string
	Tracks              []Track
	ContainerTags       map[string]string
}

// ProbeFile checks an already-stored local video without changing it.
func ProbeFile(ctx context.Context, probePath, path string) (Metadata, error) {
	if probePath == "" {
		probePath = "ffprobe"
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return Metadata{}, errProbe
	}
	service := &Service{config: Config{ProbePath: probePath}}
	return service.probe(ctx, path, info.ModTime())
}

// HashFile returns the SHA-256 identity for a stable local clip.
func HashFile(ctx context.Context, path string) ([sha256.Size]byte, error) {
	return hashFile(ctx, path)
}

func New(pool *pgxpool.Pool, config Config, logger interface{ Warn(string, ...any) }) *Service {
	if config.ProbePath == "" {
		config.ProbePath = "ffprobe"
	}
	if config.Stability < 0 {
		config.Stability = 0
	}
	if config.ReconcileEvery <= 0 {
		config.ReconcileEvery = 5 * time.Minute
	}
	if config.Exclude == nil {
		config.Exclude = []string{".*", "*.tmp", "*.part", "*.partial", "*.crdownload", "*.download"}
	}
	return &Service{pool: pool, config: config, logger: logger, watched: map[string]int64{}, roots: map[int64]string{}, timers: map[int64]*time.Timer{}}
}

func (s *Service) AllowedRoots() []string {
	return append([]string{}, s.config.Roots...)
}

func (s *Service) CreateSource(ctx context.Context, ownerID int64, name, path string) (Source, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 || len(s.config.Roots) == 0 {
		return Source{}, ErrInvalidSource
	}
	absolute, err := s.authorizedPath(path)
	if err != nil {
		return Source{}, err
	}
	var source Source
	err = s.pool.QueryRow(ctx, `INSERT INTO clip_sources(name,root_path,owner_id) VALUES($1,$2,$3) RETURNING id,name,root_path,owner_id,visibility,enabled,health,health_message,last_scan_at,managed_root`, name, absolute, ownerID).Scan(&source.ID, &source.Name, &source.Path, &source.OwnerID, &source.Visibility, &source.Enabled, &source.Health, &source.HealthMessage, &source.LastScan, &source.ManagedRoot)
	if err != nil {
		if strings.Contains(err.Error(), "clip_sources_root_path_key") || strings.Contains(err.Error(), "clip_sources_name_key") {
			return Source{}, ErrSourceExists
		}
		return Source{}, err
	}
	s.WatchSource(source)
	s.ScheduleScan(source.ID, 0)
	return source, nil
}

func (s *Service) authorizedPath(path string) (string, error) {
	if len(s.config.Roots) == 0 || strings.TrimSpace(path) == "" {
		return "", ErrInvalidSource
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", ErrInvalidSource
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", ErrUnavailable
	}
	info, err := os.Stat(real)
	if err != nil || !info.IsDir() {
		return "", ErrUnavailable
	}
	for _, configured := range s.config.Roots {
		root, err := filepath.Abs(filepath.Clean(configured))
		if err != nil {
			continue
		}
		if resolved, resolveErr := filepath.EvalSymlinks(root); resolveErr == nil {
			root = resolved
		}
		if within(root, real) {
			return real, nil
		}
	}
	return "", ErrInvalidSource
}

func within(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func (s *Service) ListSources(ctx context.Context) ([]Source, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,name,root_path,owner_id,visibility,enabled,health,health_message,last_scan_at,managed_root FROM clip_sources ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sources := make([]Source, 0)
	for rows.Next() {
		var source Source
		if err = rows.Scan(&source.ID, &source.Name, &source.Path, &source.OwnerID, &source.Visibility, &source.Enabled, &source.Health, &source.HealthMessage, &source.LastScan, &source.ManagedRoot); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func (s *Service) GetSource(ctx context.Context, id int64) (Source, error) {
	var source Source
	err := s.pool.QueryRow(ctx, `SELECT id,name,root_path,owner_id,visibility,enabled,health,health_message,last_scan_at,managed_root FROM clip_sources WHERE id=$1`, id).Scan(&source.ID, &source.Name, &source.Path, &source.OwnerID, &source.Visibility, &source.Enabled, &source.Health, &source.HealthMessage, &source.LastScan, &source.ManagedRoot)
	if errors.Is(err, pgx.ErrNoRows) {
		return Source{}, ErrNotFound
	}
	return source, err
}

func (s *Service) DisableSource(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `UPDATE clip_sources SET enabled=false WHERE id=$1 AND enabled`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clip_sources WHERE id=$1)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
	}
	s.mu.Lock()
	for path, sourceID := range s.watched {
		if sourceID == id {
			if s.watcher != nil {
				_ = s.watcher.Remove(path)
			}
			delete(s.watched, path)
		}
	}
	delete(s.roots, id)
	if timer := s.timers[id]; timer != nil {
		timer.Stop()
		delete(s.timers, id)
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) ListFiles(ctx context.Context, sourceID int64) ([]File, error) {
	rows, err := s.pool.Query(ctx, `SELECT sf.relative_path,sf.status,sf.clip_id,COALESCE(sf.error_code,''),sf.scan_requested,sf.size_bytes,sf.modified_at,c.discovered_at,c.recorded_at,COALESCE(c.recorded_at_estimated,true),c.duration_seconds,COALESCE(c.format_name,''),COALESCE(c.video_codec,''),COALESCE(c.audio_codec,''),COALESCE(c.tracks,'[]'::jsonb) FROM source_files sf LEFT JOIN clips c ON c.id=sf.clip_id WHERE sf.source_id=$1 ORDER BY sf.relative_path`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	files := make([]File, 0)
	for rows.Next() {
		var file File
		var duration *float64
		var tracks []byte
		if err = rows.Scan(&file.RelativePath, &file.Status, &file.ClipID, &file.ErrorCode, &file.ScanRequested, &file.SizeBytes, &file.ModifiedAt, &file.DiscoveredAt, &file.RecordedAt, &file.RecordedAtEstimated, &duration, &file.FormatName, &file.VideoCodec, &file.AudioCodec, &tracks); err != nil {
			return nil, err
		}
		file.Name = filepath.Base(filepath.FromSlash(file.RelativePath))
		file.ModifiedAt = file.ModifiedAt.UTC()
		if file.DiscoveredAt != nil {
			utc := file.DiscoveredAt.UTC()
			file.DiscoveredAt = &utc
		}
		if file.RecordedAt != nil {
			utc := file.RecordedAt.UTC()
			file.RecordedAt = &utc
		}
		file.DurationSeconds = duration
		file.Tracks = []Track{}
		if err = json.Unmarshal(tracks, &file.Tracks); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *Service) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	watchCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.watcher = watcher
	s.watchCancel = cancel
	s.ctx = watchCtx
	s.mu.Unlock()
	sources, err := s.ListSources(ctx)
	if err != nil {
		cancel()
		_ = watcher.Close()
		return err
	}
	for _, source := range sources {
		if source.Enabled {
			s.WatchSource(source)
			s.ScheduleScan(source.ID, 0)
		}
	}
	go s.watchLoop(watchCtx, watcher)
	go s.reconcileLoop(watchCtx)
	return nil
}

func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.watchCancel != nil {
		s.watchCancel()
	}
	s.ctx = nil
	for _, timer := range s.timers {
		timer.Stop()
	}
	if s.watcher != nil {
		_ = s.watcher.Close()
	}
}

func (s *Service) WatchSource(source Source) {
	if source.ManagedRoot {
		return
	}
	s.mu.Lock()
	watcher := s.watcher
	if source.Enabled {
		s.roots[source.ID] = source.Path
	}
	s.mu.Unlock()
	if watcher == nil || !source.Enabled {
		return
	}
	err := filepath.WalkDir(source.Path, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if err = watcher.Add(path); err != nil {
			return err
		}
		s.mu.Lock()
		s.watched[path] = source.ID
		s.mu.Unlock()
		return nil
	})
	if err != nil && s.logger != nil {
		s.logger.Warn("source watch unavailable; periodic reconciliation remains enabled", "source_id", source.ID)
	}
}

func (s *Service) watchLoop(ctx context.Context, watcher *fsnotify.Watcher) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			id := s.sourceForPath(event.Name)
			if id == 0 {
				continue
			}
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if source, err := s.GetSource(ctx, id); err == nil {
						s.WatchSubtree(source, event.Name)
					}
				}
			}
			s.ScheduleScan(id, 250*time.Millisecond)
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func (s *Service) sourceForPath(path string) int64 {
	path, _ = filepath.Abs(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	var selected int64
	longest := -1
	for id, root := range s.roots {
		if within(root, path) && len(root) > longest {
			selected, longest = id, len(root)
		}
	}
	return selected
}

func (s *Service) WatchSubtree(source Source, path string) {
	s.mu.Lock()
	watcher := s.watcher
	s.mu.Unlock()
	if watcher == nil {
		return
	}
	_ = filepath.WalkDir(path, func(dir string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return err
		}
		s.mu.Lock()
		known := s.watched[dir] == source.ID
		s.mu.Unlock()
		if known {
			return nil
		}
		if err = watcher.Add(dir); err != nil {
			return err
		}
		s.mu.Lock()
		s.watched[dir] = source.ID
		s.mu.Unlock()
		return nil
	})
}

func (s *Service) ScheduleScan(sourceID int64, delay time.Duration) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	if old := s.timers[sourceID]; old != nil {
		old.Stop()
	}
	s.timers[sourceID] = time.AfterFunc(delay, func() {
		s.mu.Lock()
		delete(s.timers, sourceID)
		if s.closed {
			s.mu.Unlock()
			return
		}
		ctx := s.ctx
		s.mu.Unlock()
		if ctx == nil {
			ctx = context.Background()
		}
		_ = s.ScanSource(ctx, sourceID)
	})
	s.mu.Unlock()
}

func (s *Service) reconcileLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.ReconcileEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sources, err := s.ListSources(ctx)
			if err != nil {
				continue
			}
			for _, source := range sources {
				if source.Enabled {
					s.ScheduleScan(source.ID, 0)
				}
			}
		}
	}
}

func (s *Service) ScanSource(ctx context.Context, sourceID int64) error {
	lockValue, _ := s.locks.LoadOrStore(sourceID, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	source, err := s.GetSource(ctx, sourceID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !source.Enabled {
		return nil
	}
	if source.ManagedRoot {
		return s.scanManaged(ctx, source)
	}
	info, err := os.Stat(source.Path)
	if err != nil || !info.IsDir() {
		_, _ = s.pool.Exec(ctx, `UPDATE clip_sources SET health='unavailable',health_message='La source est inaccessible ; les états précédents sont conservés.' WHERE id=$1`, sourceID)
		return ErrUnavailable
	}
	var generation int64
	if err = s.pool.QueryRow(ctx, `UPDATE clip_sources SET scan_generation=scan_generation+1 WHERE id=$1 RETURNING scan_generation`, sourceID).Scan(&generation); err != nil {
		return err
	}
	pending := false
	var unreadable bool
	var incomplete bool
	err = filepath.WalkDir(source.Path, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			incomplete = true
			return nil
		}
		if path == source.Path {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !entry.Type().IsRegular() || s.excluded(entry.Name()) {
			return nil
		}
		rel, relErr := filepath.Rel(source.Path, path)
		if relErr != nil || !within(source.Path, path) {
			return fmt.Errorf("unsafe source path")
		}
		stat, statErr := os.Stat(path)
		if statErr != nil {
			incomplete = true
			return nil
		}
		if stat.Mode()&os.ModeSymlink != 0 || !stat.Mode().IsRegular() {
			return nil
		}
		isPending, isUnreadable, processErr := s.observe(ctx, source, filepath.ToSlash(rel), path, stat, generation)
		if processErr != nil {
			return processErr
		}
		pending = pending || isPending
		unreadable = unreadable || isUnreadable
		return nil
	})
	if err != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE clip_sources SET health='warning',health_message='La réconciliation est incomplète ; aucun chemin n’est marqué comme disparu.' WHERE id=$1`, sourceID)
		return err
	}
	if incomplete {
		_, _ = s.pool.Exec(ctx, `UPDATE clip_sources SET health='warning',health_message='La réconciliation est partielle ; les états précédents sont conservés.' WHERE id=$1`, sourceID)
		return nil
	}
	var totalFiles, missingFiles, unreadableFiles int
	err = s.pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE status='missing' OR last_seen_generation<$2),count(*) FILTER (WHERE status='unreadable' AND last_seen_generation >= $2) FROM source_files WHERE source_id=$1`, sourceID, generation).Scan(&totalFiles, &missingFiles, &unreadableFiles)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE clip_sources SET health='warning',health_message='La réconciliation est incomplète ; aucun chemin n’est marqué comme disparu.' WHERE id=$1`, sourceID)
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE source_files SET status='missing',error_code=NULL WHERE source_id=$1 AND last_seen_generation<$2 AND status<>'missing'`, sourceID, generation)
	if err != nil {
		_, _ = s.pool.Exec(ctx, `UPDATE clip_sources SET health='warning',health_message='La réconciliation est incomplète ; aucun chemin n’est marqué comme disparu.' WHERE id=$1`, sourceID)
		return err
	}
	health := "ready"
	healthMessage := ""
	if totalFiles > 0 && float64(missingFiles)/float64(totalFiles) >= 0.2 {
		health = "warning"
		healthMessage = fmt.Sprintf("%d fichiers introuvables sur %d lors de la réconciliation.", missingFiles, totalFiles)
	}
	if unreadableFiles > 0 || unreadable {
		health = "warning"
		if healthMessage == "" {
			healthMessage = fmt.Sprintf("%d fichier(s) illisible(s) lors de la réconciliation.", unreadableFiles)
		}
	}
	if _, err = s.pool.Exec(ctx, `UPDATE clip_sources SET health=$2,health_message=$3,last_scan_at=now() WHERE id=$1`, sourceID, health, healthMessage); err != nil {
		return err
	}
	if pending {
		s.ScheduleScan(sourceID, s.config.Stability/2+10*time.Millisecond)
	}
	return nil
}

func (s *Service) observe(ctx context.Context, source Source, relativePath, absolutePath string, stat os.FileInfo, generation int64) (pending, unreadable bool, err error) {
	// PostgreSQL timestamptz stores microsecond precision; normalize filesystem
	// timestamps before comparison so nanosecond filesystems don't reset stability.
	modified := stat.ModTime().UTC().Truncate(time.Microsecond)
	var clipID *int64
	var oldSize int64
	var oldModified, stableSince time.Time
	var status string
	err = s.pool.QueryRow(ctx, `SELECT clip_id,size_bytes,modified_at,stable_since,status FROM source_files WHERE source_id=$1 AND relative_path=$2 FOR UPDATE`, source.ID, relativePath).Scan(&clipID, &oldSize, &oldModified, &stableSince, &status)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, false, err
	}
	known := err == nil
	changed := !known || oldSize != stat.Size() || !oldModified.Equal(modified) || status == "missing"
	if changed {
		stableSince = time.Now().UTC()
		clipID = nil
		status = "waiting_stability"
		_, err = s.pool.Exec(ctx, `INSERT INTO source_files(source_id,relative_path,size_bytes,modified_at,stable_since,last_seen_generation,status) VALUES($1,$2,$3,$4,$5,$6,'waiting_stability') ON CONFLICT(source_id,relative_path) DO UPDATE SET clip_id=NULL,size_bytes=EXCLUDED.size_bytes,modified_at=EXCLUDED.modified_at,stable_since=EXCLUDED.stable_since,last_seen_generation=EXCLUDED.last_seen_generation,status='waiting_stability',error_code=NULL`, source.ID, relativePath, stat.Size(), modified, stableSince, generation)
		if err != nil {
			return false, false, err
		}
		return true, false, nil
	}
	if status == "ready" {
		_, err = s.pool.Exec(ctx, `UPDATE source_files SET last_seen_generation=$3 WHERE source_id=$1 AND relative_path=$2`, source.ID, relativePath, generation)
		return false, false, err
	}
	if status == "unreadable" {
		_, err = s.pool.Exec(ctx, `UPDATE source_files SET last_seen_generation=$3 WHERE source_id=$1 AND relative_path=$2`, source.ID, relativePath, generation)
		return false, true, err
	}
	if time.Since(stableSince) < s.config.Stability {
		_, err = s.pool.Exec(ctx, `UPDATE source_files SET last_seen_generation=$3 WHERE source_id=$1 AND relative_path=$2`, source.ID, relativePath, generation)
		return true, false, err
	}
	metadata, probeErr := s.probe(ctx, absolutePath, modified)
	if probeErr != nil {
		_, err = s.pool.Exec(ctx, `UPDATE source_files SET status='unreadable',error_code=$3,last_seen_generation=$4 WHERE source_id=$1 AND relative_path=$2`, source.ID, relativePath, errorCode(probeErr), generation)
		return false, true, err
	}
	latest, statErr := os.Stat(absolutePath)
	if statErr != nil || latest.Size() != stat.Size() || !latest.ModTime().UTC().Truncate(time.Microsecond).Equal(modified) {
		_, err = s.pool.Exec(ctx, `UPDATE source_files SET status='waiting_stability',stable_since=now(),last_seen_generation=$3 WHERE source_id=$1 AND relative_path=$2`, source.ID, relativePath, generation)
		return true, false, err
	}
	contentHash, err := hashFile(ctx, absolutePath)
	if err != nil {
		_, updateErr := s.pool.Exec(ctx, `UPDATE source_files SET status='unreadable',error_code='video_unreadable',last_seen_generation=$3 WHERE source_id=$1 AND relative_path=$2`, source.ID, relativePath, generation)
		return false, true, updateErr
	}
	latest, statErr = os.Stat(absolutePath)
	if statErr != nil || latest.Size() != stat.Size() || !latest.ModTime().UTC().Truncate(time.Microsecond).Equal(modified) {
		_, err = s.pool.Exec(ctx, `UPDATE source_files SET status='waiting_stability',stable_since=now(),last_seen_generation=$3 WHERE source_id=$1 AND relative_path=$2`, source.ID, relativePath, generation)
		return true, false, err
	}
	tracks, err := json.Marshal(metadata.Tracks)
	if err != nil {
		return false, false, err
	}
	game, gameOrigin, captureTime, captureOrigin, err := CaptureMetadata(ctx, s.pool, filepath.Base(absolutePath), metadata.ContainerTags)
	if err != nil {
		return false, false, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, false, err
	}
	defer tx.Rollback(ctx)
	var newClipID int64
	err = tx.QueryRow(ctx, `INSERT INTO clips(owner_id,visibility,original_name,size_bytes,modified_at,recorded_at,recorded_at_estimated,duration_seconds,format_name,video_codec,audio_codec,tracks,content_sha256,game,game_origin,capture_time,capture_time_origin) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NULLIF($14,''),NULLIF($15,''),$16,NULLIF($17,'')) ON CONFLICT(content_sha256) DO UPDATE SET content_sha256=EXCLUDED.content_sha256 RETURNING id`, sourceOwnerID(source), source.Visibility, filepath.Base(absolutePath), stat.Size(), modified, metadata.RecordedAt, metadata.RecordedAtEstimated, metadata.Duration, metadata.FormatName, metadata.VideoCodec, metadata.AudioCodec, tracks, contentHash[:], game, gameOrigin, captureTime, captureOrigin).Scan(&newClipID)
	if err != nil {
		return false, false, err
	}
	_, err = tx.Exec(ctx, `UPDATE source_files SET clip_id=$3,status='ready',error_code=NULL,last_seen_generation=$4 WHERE source_id=$1 AND relative_path=$2`, source.ID, relativePath, newClipID, generation)
	if err != nil {
		return false, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, false, err
	}
	return false, false, nil
}

// Source owner IDs are looked up separately so the exported source shape does not leak it.
func sourceOwnerID(source Source) int64 { return source.OwnerID }

func (s *Service) excluded(name string) bool {
	for _, pattern := range s.config.Exclude {
		if match, _ := filepath.Match(pattern, name); match {
			return true
		}
	}
	return false
}

func (s *Service) probe(ctx context.Context, path string, modified time.Time) (Metadata, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, s.config.ProbePath,
		"-v", "error", "-protocol_whitelist", "file,pipe",
		"-show_entries", "format=duration,format_name:format_tags=creation_time,game:stream=index,codec_type,codec_name,width,height,channels,channel_layout:stream_tags=title,language",
		"-of", "json", path,
	)
	output := &cappedBuffer{max: 1 << 20}
	cmd.Stdout = output
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return Metadata{}, errProbe
	}
	var result probeResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		return Metadata{}, errProbe
	}
	metadata := Metadata{RecordedAt: modified, RecordedAtEstimated: true, FormatName: result.Format.FormatName, Tracks: []Track{}, ContainerTags: result.Format.Tags}
	if creation := tag(result.Format.Tags, "creation_time"); creation != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, creation); err == nil {
			metadata.RecordedAt = parsed.UTC()
			metadata.RecordedAtEstimated = false
		}
	}
	if duration, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil && duration > 0 {
		metadata.Duration = &duration
	}
	for _, stream := range result.Streams {
		track := Track{Index: stream.Index, Type: stream.Type, Codec: stream.Codec, Title: tag(stream.Tags, "title"), Language: tag(stream.Tags, "language"), Width: stream.Width, Height: stream.Height, Channels: stream.Channels, ChannelLayout: stream.ChannelLayout}
		metadata.Tracks = append(metadata.Tracks, track)
		switch stream.Type {
		case "video":
			if metadata.VideoCodec == "" {
				metadata.VideoCodec = stream.Codec
			}
		case "audio":
			if metadata.AudioCodec == "" {
				metadata.AudioCodec = stream.Codec
			}
		}
	}
	if metadata.VideoCodec == "" {
		return Metadata{}, errNoVideo
	}
	return metadata, nil
}

func hashFile(ctx context.Context, path string) ([sha256.Size]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 1024*1024)
	for {
		if err := ctx.Err(); err != nil {
			return [sha256.Size]byte{}, err
		}
		count, readErr := file.Read(buffer)
		if count > 0 {
			_, _ = hash.Write(buffer[:count])
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return [sha256.Size]byte{}, readErr
		}
	}
	var sum [sha256.Size]byte
	copy(sum[:], hash.Sum(nil))
	return sum, nil
}

var (
	errProbe   = errors.New("probe failed")
	errNoVideo = errors.New("video stream missing")
)

func errorCode(err error) string {
	if errors.Is(err, errNoVideo) {
		return "video_stream_missing"
	}
	return "video_unreadable"
}

func tag(tags map[string]string, name string) string {
	for key, value := range tags {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

type cappedBuffer struct {
	bytes.Buffer
	max int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.max {
		return 0, fmt.Errorf("probe output too large")
	}
	return b.Buffer.Write(p)
}

func (s *Service) WatchStatus(ctx context.Context, id int64) error {
	_, err := s.GetSource(ctx, id)
	return err
}
