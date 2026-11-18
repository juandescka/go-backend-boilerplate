package migration

import (
	"net"
	"net/url"
	"path/filepath"
	"strconv"

	"hris-core-service/config"
)

const DefaultMigrationsPath = "migrations/postgres"

type Config struct {
	SourceURL   string
	DatabaseURL string
}

func NewConfig() Config {
	return Config{SourceURL: migrationSourceURL(DefaultMigrationsPath), DatabaseURL: migrationDatabaseURL(config.AppConfig.Database)}
}

func migrationSourceURL(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}).String()
}

func migrationDatabaseURL(cfg config.DatabaseConfig) string {
	u := url.URL{Scheme: "postgres", User: url.UserPassword(cfg.Username, cfg.Password), Host: net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)), Path: "/" + cfg.Name, RawPath: "/" + url.PathEscape(cfg.Name)}
	u.RawQuery = url.Values{
		"sslmode":              {cfg.SSLMode},
		"connect_timeout":      {strconv.Itoa(cfg.QueryTimeoutSeconds)},
		"application_name":     {"hris-core-service-migrate"},
		"timezone":             {"UTC"},
		"client_encoding":      {"UTF8"},
		"target_session_attrs": {"any"},
	}.Encode()
	return u.String()
}
