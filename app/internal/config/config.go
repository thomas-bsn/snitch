package config

import (
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL, PublicURL, Address, DisplayName, EmbeddingsURL, EmbeddingsModel, WorkerImage string
	ClipRoots                                                                                 []string
	ProbePath, FFmpegPath                                                                     string
	StabilityWindow                                                                           time.Duration
	ReconcileInterval                                                                         time.Duration
	ClipExcludePatterns                                                                       []string
	UploadRoot                                                                                string
	UploadMaxBytes, UploadChunkBytes                                                          int64
	UploadSessionTTL                                                                          time.Duration
	MasterKey                                                                                 []byte
	Secure                                                                                    bool
	DiscordAuthorizeURL, DiscordTokenURL, DiscordUserURL                                      string
}

const DefaultWorkerImage = "ghcr.io/thomas-bsn/snitch-worker:latest"

var workerImagePattern = regexp.MustCompile(`^[a-z0-9][A-Za-z0-9._/:@-]*$`)

func Load() (Config, error) {
	c := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"), PublicURL: env("PUBLIC_URL", "http://localhost:8080"),
		Address: env("LISTEN_ADDR", ":8080"), DisplayName: env("DISPLAY_NAME", "Snitch"),
		// The application carries no release version, so the default image tag is "latest";
		// pin WORKER_IMAGE to a published tag or digest to control worker upgrades.
		WorkerImage:   env("WORKER_IMAGE", DefaultWorkerImage),
		EmbeddingsURL: os.Getenv("EMBEDDINGS_URL"), EmbeddingsModel: os.Getenv("EMBEDDINGS_MODEL"),
		ClipRoots: splitRoots(os.Getenv("CLIPS_ROOTS")), ProbePath: env("FFPROBE_PATH", "ffprobe"), FFmpegPath: env("FFMPEG_PATH", "ffmpeg"),
		ClipExcludePatterns: splitPatterns(env("CLIP_EXCLUDE_PATTERNS", ".*,*.tmp,*.part,*.partial,*.crdownload,*.download")),
		UploadRoot:          env("UPLOAD_ROOT", "/data/media"),
		DiscordAuthorizeURL: "https://discord.com/oauth2/authorize", DiscordTokenURL: "https://discord.com/api/oauth2/token", DiscordUserURL: "https://discord.com/api/users/@me",
	}
	var err error
	c.StabilityWindow, err = time.ParseDuration(env("CLIP_STABILITY_WINDOW", "3s"))
	if err != nil || c.StabilityWindow < 0 || c.StabilityWindow > 24*time.Hour {
		return c, errors.New("CLIP_STABILITY_WINDOW must be between 0 and 24 hours")
	}
	c.ReconcileInterval, err = time.ParseDuration(env("SOURCE_RECONCILE_INTERVAL", "5m"))
	if err != nil || c.ReconcileInterval < time.Second || c.ReconcileInterval > 24*time.Hour {
		return c, errors.New("SOURCE_RECONCILE_INTERVAL must be between 1 second and 24 hours")
	}
	c.UploadRoot, err = filepath.Abs(filepath.Clean(c.UploadRoot))
	if err != nil || c.UploadRoot == "." {
		return c, errors.New("UPLOAD_ROOT must be a valid path")
	}
	c.UploadMaxBytes, err = int64Env("MAX_UPLOAD_BYTES", 20<<30)
	if err != nil || c.UploadMaxBytes < 1 || c.UploadMaxBytes > 1<<50 {
		return c, errors.New("MAX_UPLOAD_BYTES must be between 1 byte and 1 PiB")
	}
	c.UploadChunkBytes, err = int64Env("UPLOAD_CHUNK_BYTES", 4<<20)
	if err != nil || c.UploadChunkBytes < 16<<10 || c.UploadChunkBytes > 16<<20 {
		return c, errors.New("UPLOAD_CHUNK_BYTES must be between 16 KiB and 16 MiB")
	}
	c.UploadSessionTTL, err = time.ParseDuration(env("UPLOAD_SESSION_TTL", "168h"))
	if err != nil || c.UploadSessionTTL < time.Hour || c.UploadSessionTTL > 30*24*time.Hour {
		return c, errors.New("UPLOAD_SESSION_TTL must be between 1 hour and 30 days")
	}
	c.MasterKey, err = base64.StdEncoding.DecodeString(os.Getenv("MASTER_KEY"))
	if err != nil || len(c.MasterKey) != 32 {
		return c, errors.New("MASTER_KEY must encode 32 bytes in base64")
	}
	c.Secure, err = strconv.ParseBool(env("COOKIE_SECURE", "true"))
	if err != nil {
		return c, errors.New("invalid COOKIE_SECURE")
	}
	u, e := url.Parse(c.PublicURL)
	if e != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return c, errors.New("PUBLIC_URL must be an HTTP(S) origin")
	}
	c.PublicURL = strings.TrimRight(c.PublicURL, "/")
	if c.Secure && u.Scheme != "https" {
		return c, errors.New("secure cookies require HTTPS PUBLIC_URL")
	}
	if !workerImagePattern.MatchString(c.WorkerImage) {
		return c, errors.New("WORKER_IMAGE must be a container image reference")
	}
	db, err := url.Parse(c.DatabaseURL)
	if err != nil || db.Host == "" || (db.Scheme != "postgres" && db.Scheme != "postgresql") {
		return c, errors.New("DATABASE_URL must be a PostgreSQL URL")
	}
	if strings.TrimSpace(c.DisplayName) == "" {
		return c, errors.New("DISPLAY_NAME must not be blank")
	}
	_, port, err := net.SplitHostPort(c.Address)
	n, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || n < 1 || n > 65535 {
		return c, errors.New("LISTEN_ADDR must contain a host and port from 1 to 65535")
	}
	if c.EmbeddingsURL != "" {
		endpoint, err := url.Parse(c.EmbeddingsURL)
		if err != nil || endpoint.Host == "" || endpoint.User != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.RawQuery != "" || endpoint.Fragment != "" {
			return c, errors.New("EMBEDDINGS_URL must be an HTTP(S) URL without credentials, query or fragment")
		}
	}
	return c, nil
}

func splitRoots(value string) []string {
	var roots []string
	for _, root := range filepath.SplitList(value) {
		if strings.TrimSpace(root) == "" {
			continue
		}
		absolute, err := filepath.Abs(filepath.Clean(root))
		if err == nil {
			roots = append(roots, absolute)
		}
	}
	return roots
}

func splitPatterns(value string) []string {
	var patterns []string
	for _, pattern := range strings.Split(value, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			patterns = append(patterns, pattern)
		}
	}
	return patterns
}

func int64Env(key string, fallback int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
