package config

import (
	"encoding/base64"
	"errors"
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
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return c, errors.New("PUBLIC_URL must be an HTTP(S) origin")
	}
	c.PublicURL = strings.TrimRight(c.PublicURL, "/")
	if c.Secure && u.Scheme != "https" {
		return c, errors.New("secure cookies require HTTPS PUBLIC_URL")
	}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL required")
	}
	return c, nil
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
