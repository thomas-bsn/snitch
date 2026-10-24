package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"receipts/internal/config"
	"receipts/internal/httpapi"
	"receipts/internal/setup"
	"receipts/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("command failed", "reason", err.Error())
		os.Exit(1)
	}
}
func run() error {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: receipts serve|migrate|healthcheck")
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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
	code, err := setup.Issue(probe, p)
	if err != nil {
		return fmt.Errorf("database unavailable or migrations missing")
	}
	if code != "" {
		slog.Warn("one-time installation code", "code", code, "expires_in_minutes", 15)
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
