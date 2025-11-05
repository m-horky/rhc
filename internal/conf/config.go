package conf

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

func init() {
	sources := &ConfigSource{
		Path:      "/etc/rhc/config.toml",
		DropInDir: "/etc/rhc/config.toml.d/",
	}
	config, err := sources.Read()
	if err != nil {
		dto, parseErr := parseConfigDTO(defaultConfig)
		if parseErr != nil {
			panic(fmt.Sprintf("failed to parse embedded defaults: %v", parseErr))
		}
		config.Update(dto)
	}
	Configuration = config
}

// defaultConfig contains the embedded default configuration file.
// This file is compiled into the binary and serves as the base layer
// of configuration before /etc/rhc/config.toml and drop-in files are applied.
//
//go:embed ../../data/etc/config.toml
var defaultConfig string

// Configuration is the global immutable state.
var Configuration Config

// Config represents the immutable public configuration object.
type Config struct {
	CADir    string
	CertFile string
	KeyFile  string
	LogLevel slog.Level
}

// Update applies non-nil values from a configDTO.
func (c *Config) Update(dto configDTO) {
	if dto.CertFile != nil {
		c.CertFile = *dto.CertFile
	}
	if dto.KeyFile != nil {
		c.KeyFile = *dto.KeyFile
	}
	if dto.LogLevel != nil {
		switch *dto.LogLevel {
		case "DEBUG":
			c.LogLevel = slog.LevelDebug
		case "INFO":
			c.LogLevel = slog.LevelInfo
		case "WARN":
			c.LogLevel = slog.LevelWarn
		case "ERROR":
			c.LogLevel = slog.LevelError
		}
	}
	if dto.CADir != nil {
		c.CADir = *dto.CADir
	}
}

// ConfigSource orchestrates loading configuration from multiple sources.
// See the Read method.
type ConfigSource struct {
	Path      string
	DropInDir string
}

// Read loads and returns the complete Config by merging all layers:
// 1. Embedded defaults
// 2. Main configuration file
// 3. Drop-in files
func (cs *ConfigSource) Read() (Config, error) {
	resolved := Config{}

	// Start with embedded defaults
	dto, err := parseConfigDTO(defaultConfig)
	if err != nil {
		slog.Error("failed to parse embedded defaults", "error", err)
		return resolved, fmt.Errorf("failed to parse embedded defaults: %w", err)
	}
	resolved.Update(dto)

	// Load main configuration file
	data, err := os.ReadFile(cs.Path)
	if err != nil {
		if !os.IsNotExist(err) {
			// Existing but malformed file should result in failure (let's not hide
			// problems from the users).
			return resolved, fmt.Errorf("failed to load %s: %w", cs.Path, err)
		}
	} else {
		mainDTO, err := parseConfigDTO(string(data))
		if err != nil {
			// Existing but malformed file should result in failure (let's not hide
			// problems from the users).
			return resolved, fmt.Errorf("failed to parse %s: %w", cs.Path, err)
		}
		resolved.Update(mainDTO)
	}

	// Load drop-in files
	dropInDTOs, err := cs.parseDropInFiles()
	if err != nil {
		slog.Error("failed to load drop-in files", "error", err, "dir", cs.DropInDir)
		return resolved, err
	}

	// Apply each drop-in file in order
	for _, dropInDTO := range dropInDTOs {
		resolved.Update(dropInDTO)
	}

	return resolved, nil
}

type configDTO struct {
	CertFile *string `toml:"cert-file"`
	KeyFile  *string `toml:"key-file"`
	LogLevel *string `toml:"log-level"`
	CADir    *string `toml:"ca-dir"`
}

// parseConfigDTO parses a TOML string into a configDTO.
func parseConfigDTO(data string) (configDTO, error) {
	var dto configDTO

	if err := toml.Unmarshal([]byte(data), &dto); err != nil {
		return dto, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return dto, nil
}

// findDropInFiles finds and returns sorted paths to drop-in configuration files.
// Returns nil if the drop-in directory doesn't exist (not an error).
func (cs *ConfigSource) findDropInFiles() ([]string, error) {
	// Check if drop-in directory exists
	if _, err := os.Stat(cs.DropInDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Read directory contents
	entries, err := os.ReadDir(cs.DropInDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read drop-in directory %s: %w", cs.DropInDir, err)
	}

	// Collect .toml files
	var filenames []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".toml") {
			filenames = append(filenames, filepath.Join(cs.DropInDir, entry.Name()))
		}
	}

	// Sort lexicographically
	sort.Strings(filenames)

	return filenames, nil
}

// parseDropInFiles loads .toml files.
func (cs *ConfigSource) parseDropInFiles() ([]configDTO, error) {
	paths, err := cs.findDropInFiles()
	if err != nil {
		return nil, err
	}

	// Load each file
	var dtos []configDTO
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		dto, err := parseConfigDTO(string(data))
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		dtos = append(dtos, dto)
	}

	return dtos, nil
}
