package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"snitch/internal/library"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"snitch/internal/config"
	"snitch/internal/httpapi"
	"snitch/internal/logging"
	"snitch/internal/secrets"
	"snitch/internal/setup"
	"snitch/migrations"
)

func main() {
	if err := run(); err != nil {
		// Every error returned by run is a fixed sentence written for the operator
		// (never a wrapped driver or provider error), so the message stays readable.
		slog.Error("command failed", append(logging.Failure(err), "message", err.Error())...)
		os.Exit(1)
	}
}
func run() error {
	slog.SetDefault(logging.New(os.Stdout, "snitch", os.Getenv("LOG_LEVEL")))
	if len(os.Args) == 4 && os.Args[1] == "probe-smoke" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		valid, err := library.ProbeFile(ctx, "ffprobe", os.Args[2])
		if err != nil || valid.VideoCodec == "" {
			return fmt.Errorf("synthetic video probe failed")
		}
		if _, err = library.ProbeFile(ctx, "ffprobe", os.Args[3]); err == nil {
			return fmt.Errorf("no-video fixture was accepted")
		}
		return nil
	}
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: snitch serve|migrate|init-secrets|healthcheck|probe-smoke <video> <no-video>")
	}
	if os.Args[1] == "healthcheck" {
		client := http.Client{Timeout: 3 * time.Second}
		address := os.Getenv("LISTEN_ADDR")
		if address == "" {
			address = ":8080"
		}
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("invalid LISTEN_ADDR")
		}
		if host == "" || host == "0.0.0.0" || host == "::" {
			host = "127.0.0.1"
		}
		resp, err := client.Get("http://" + net.JoinHostPort(host, port) + "/api/health/ready")
		if err != nil {
			return fmt.Errorf("API unavailable")
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("API not ready")
		}
		return nil
	}
	if os.Args[1] == "init-secrets" {
		// Runs before the database starts: produces (or keeps) the password the database and the application share.
		dir := os.Getenv("SECRETS_DIR")
		if dir == "" {
			dir = "/data/secrets"
		}
		return secrets.EnsurePostgresPassword(dir, os.Getenv("POSTGRES_PASSWORD"))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if os.Args[1] != "migrate" && os.Args[1] != "serve" {
		return fmt.Errorf("unknown command")
	}
	// The deployment keeps the database password in a file; without it, PGPASSWORD (explicit setups) applies.
	if file := os.Getenv("POSTGRES_PASSWORD_FILE"); file != "" {
		password, err := secrets.ReadPostgresPassword(file)
		if err != nil {
			return err
		}
		os.Setenv("PGPASSWORD", password)
	}
	if os.Args[1] == "migrate" {
		// This command deliberately needs only DATABASE_URL, not the encryption key.
		db, err := goose.OpenDBWithDriver("pgx", os.Getenv("DATABASE_URL"))
		if err != nil {
			return fmt.Errorf("cannot open migration database")
		}
		defer db.Close()
		provider, err := goose.NewProvider(database.DialectPostgres, db, migrations.Files)
		if err != nil {
			return fmt.Errorf("cannot initialize migrations")
		}
		_, err = provider.Up(ctx)
		if err != nil {
			return fmt.Errorf("migration failed")
		}
		return nil
	}
	if os.Args[1] != "serve" {
		return fmt.Errorf("unknown command")
	}
	c, err := config.Load()
	if err != nil {
		return err
	}
	pc, err := pgxpool.ParseConfig(c.DatabaseURL)
	if err != nil {
		return fmt.Errorf("invalid DATABASE_URL")
	}
	pc.MaxConns = 6
	pc.MinConns = 0
	pc.MaxConnIdleTime = time.Minute * 5
	p, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return fmt.Errorf("cannot initialize database pool")
	}
	defer p.Close()
	probe, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := checkSchema(probe, p); err != nil {
		return fmt.Errorf("database unavailable or migrations missing: run snitch migrate before serve")
	}
	code, err := setup.Issue(probe, p)
	if err != nil {
		return fmt.Errorf("database unavailable or migrations missing")
	}
	if code != "" {
		// One readable line to copy from: the address to open and the code to enter.
		address := c.PublicURL
		if address == "" {
			_, port, _ := net.SplitHostPort(c.Address)
			address = "http://localhost:" + port
		}
		logging.InstallationCode(slog.Default(), address, code)
	}
	// Expired transient rows do not accumulate during normal service operation.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup, cancel := context.WithTimeout(ctx, 5*time.Second)
				_, _ = p.Exec(cleanup, `DELETE FROM sessions WHERE expires_at<now()`)
				_, _ = p.Exec(cleanup, `DELETE FROM oauth_states WHERE expires_at<now()`)
				cancel()
			}
		}
	}()
	api := httpapi.New(p, c, slog.Default())
	if err = api.Start(ctx); err != nil {
		return fmt.Errorf("cannot start source monitoring")
	}
	defer api.Close()
	server := &http.Server{Addr: c.Address, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	errs := make(chan error, 1)
	go func() { errs <- server.ListenAndServe() }()
	slog.Info("server started", "address", c.Address)
	select {
	case err = <-errs:
		if err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server failed")
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = server.Shutdown(shutdown); err != nil {
			return fmt.Errorf("graceful shutdown timed out")
		}
	}
	return nil
}

// Check every bundled version without creating goose tables or applying migrations.
func checkSchema(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := migrations.Files.ReadDir(".")
	if err != nil {
		return err
	}
	expected := map[int64]bool{}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		prefix, _, _ := strings.Cut(file.Name(), "_")
		version, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return err
		}
		expected[version] = true
	}
	rows, err := pool.Query(ctx, `SELECT version_id,is_applied FROM goose_db_version ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	applied := map[int64]bool{}
	for rows.Next() {
		var version int64
		var active bool
		if err := rows.Scan(&version, &active); err != nil {
			return err
		}
		if version != 0 {
			if active {
				applied[version] = true
			} else {
				delete(applied, version)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(applied) != len(expected) {
		return fmt.Errorf("schema version mismatch")
	}
	for version := range expected {
		if !applied[version] {
			return fmt.Errorf("schema version mismatch")
		}
	}
	return nil
}
