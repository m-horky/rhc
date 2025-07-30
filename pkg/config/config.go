package config

import (
	"log/slog"
	"os"
	"path/filepath"
)

const (
	CliLogLevel  = "log-level"
	CliCertFile  = "cert-file"
	CliKeyFile   = "key-file"
	CliAPIServer = "base-url"
)

type Config struct {
	CertFile string
	KeyFile  string
	LogLevel slog.Level
	CADir    string
}

// Global config instance
var Global = Config{}

// ConfigPath returns the default configuration file path
func ConfigPath() (string, error) {
	// default config file path in `/etc/rhc/config.toml`
	filePath := filepath.Join("/etc", LongName, "config.toml")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", nil
	}

	return filePath, nil
}