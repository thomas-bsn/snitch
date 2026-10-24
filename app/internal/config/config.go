package config

import (
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL, PublicURL, Address, DisplayName, EmbeddingsURL, EmbeddingsModel string
	MasterKey                                                                    []byte
	Secure                                                                       bool
	DiscordAuthorizeURL, DiscordTokenURL, DiscordUserURL                         string
}

func Load() (Config, error) {
	c := Config{DatabaseURL: os.Getenv("DATABASE_URL"), PublicURL: env("PUBLIC_URL", "http://localhost:8080"), Address: env("LISTEN_ADDR", ":8080"), DisplayName: env("DISPLAY_NAME", "Receipts"), EmbeddingsURL: os.Getenv("EMBEDDINGS_URL"), EmbeddingsModel: os.Getenv("EMBEDDINGS_MODEL"), DiscordAuthorizeURL: "https://discord.com/oauth2/authorize", DiscordTokenURL: "https://discord.com/api/oauth2/token", DiscordUserURL: "https://discord.com/api/users/@me"}
	var err error
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
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
