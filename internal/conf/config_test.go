package conf

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Helper functions for creating pointer values in DTO tests
func stringPtr(s string) *string { return &s }

func TestConfig_Update(t *testing.T) {
	tests := []struct {
		name     string
		base     Config
		overlay  configDTO
		expected Config
	}{
		{
			name: "overlay replaces values",
			base: Config{
				CertFile: "/etc/rhc/base.pem",
				LogLevel: slog.LevelInfo,
			},
			overlay: configDTO{
				CertFile: stringPtr("/etc/rhc/overlay.pem"),
				LogLevel: stringPtr("DEBUG"),
			},
			expected: Config{
				CertFile: "/etc/rhc/overlay.pem",
				LogLevel: slog.LevelDebug,
			},
		},
		{
			name: "overlay partial update",
			base: Config{
				CertFile: "/etc/rhc/cert.pem",
				KeyFile:  "/etc/rhc/key.pem",
				LogLevel: slog.LevelInfo,
			},
			overlay: configDTO{
				LogLevel: stringPtr("DEBUG"),
			},
			expected: Config{
				CertFile: "/etc/rhc/cert.pem",
				KeyFile:  "/etc/rhc/key.pem",
				LogLevel: slog.LevelDebug,
			},
		},
		{
			name: "empty overlay does nothing",
			base: Config{
				CertFile: "/etc/rhc/cert.pem",
				LogLevel: slog.LevelInfo,
			},
			overlay: configDTO{},
			expected: Config{
				CertFile: "/etc/rhc/cert.pem",
				LogLevel: slog.LevelInfo,
			},
		},
		{
			name: "overlay can set empty strings",
			base: Config{
				CertFile: "/etc/rhc/cert.pem",
				CADir:    "/etc/pki/tls/certs",
			},
			overlay: configDTO{
				CertFile: stringPtr(""),
				CADir:    stringPtr(""),
			},
			expected: Config{
				CertFile: "",
				CADir:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.base
			result.Update(tt.overlay)
			if diff := cmp.Diff(tt.expected, result); diff != "" {
				t.Errorf("Update() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConfigSource_ReadFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		fileContent string
		setupFile   bool
		expectError bool
		expected    Config
	}{
		{
			name: "valid config file",
			fileContent: `cert-file = "/etc/rhc/test.pem"
log-level = "DEBUG"
ca-dir = "/test/certs"
`,
			setupFile:   true,
			expectError: false,
			expected: Config{
				CertFile: "/etc/rhc/test.pem",
				LogLevel: slog.LevelDebug,
				CADir:    "/test/certs",
			},
		},
		{
			name:        "missing file uses defaults",
			setupFile:   false,
			expectError: false,
			expected: Config{
				LogLevel: slog.LevelInfo, // from defaults
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test-"+tt.name+".toml")

			if tt.setupFile {
				if err := os.WriteFile(testFile, []byte(tt.fileContent), 0644); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
			}

			source := &ConfigSource{Path: testFile, DropInDir: filepath.Join(tmpDir, "nonexistent")}
			result, err := source.Read()

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError {
				if diff := cmp.Diff(tt.expected, result); diff != "" {
					t.Errorf("Read() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestParseConfigDTO(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    configDTO
	}{
		{
			name: "valid TOML string",
			input: `
cert-file = "/etc/rhc/cert.pem"
key-file = "/etc/rhc/key.pem"
`,
			expectError: false,
			expected: configDTO{
				CertFile: stringPtr("/etc/rhc/cert.pem"),
				KeyFile:  stringPtr("/etc/rhc/key.pem"),
			},
		},
		{
			name:        "empty string",
			input:       "",
			expectError: false,
			expected:    configDTO{},
		},
		{
			name:        "invalid TOML",
			input:       "not valid toml ===",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseConfigDTO(tt.input)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError {
				if diff := cmp.Diff(tt.expected, result); diff != "" {
					t.Errorf("parseConfigDTO() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestConfigSource_FullStack(t *testing.T) {
	// Create temporary directory structure for testing
	tmpDir := t.TempDir()
	mainConfigPath := filepath.Join(tmpDir, "config.toml")
	dropinDir := filepath.Join(tmpDir, "config.toml.d")

	// Create drop-in directory
	if err := os.Mkdir(dropinDir, 0755); err != nil {
		t.Fatalf("failed to create drop-in directory: %v", err)
	}

	// Test case: main config + drop-ins with proper ordering
	t.Run("full configuration stack", func(t *testing.T) {
		// Write main config
		mainConfig := `
cert-file = "/etc/rhc/main.pem"
log-level = "INFO"
ca-dir = "/etc/pki/tls/certs"
`
		if err := os.WriteFile(mainConfigPath, []byte(mainConfig), 0644); err != nil {
			t.Fatalf("failed to write main config: %v", err)
		}

		// Write drop-in files (should be loaded in lexicographic order)
		dropinFiles := map[string]string{
			"10-key.toml":   `key-file = "/etc/rhc/dropin.key"`,
			"20-debug.toml": `log-level = "DEBUG"`,
			"30-cadir.toml": `ca-dir = "/custom/certs"`,
		}

		for filename, content := range dropinFiles {
			path := filepath.Join(dropinDir, filename)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write drop-in file %s: %v", filename, err)
			}
		}

		// Load configuration
		cs := &ConfigSource{Path: mainConfigPath, DropInDir: dropinDir}
		config, err := cs.Read()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify final configuration
		// Defaults < Main < Drop-ins (in order)
		if config.CertFile != "/etc/rhc/main.pem" {
			t.Errorf("expected CertFile=/etc/rhc/main.pem, got %s", config.CertFile)
		}
		if config.KeyFile != "/etc/rhc/dropin.key" {
			t.Errorf("expected KeyFile=/etc/rhc/dropin.key, got %s", config.KeyFile)
		}
		if config.LogLevel != slog.LevelDebug {
			t.Errorf("expected LogLevel=DEBUG, got %v", config.LogLevel)
		}
		if config.CADir != "/custom/certs" {
			t.Errorf("expected CADir=/custom/certs, got %s", config.CADir)
		}
	})

	t.Run("drop-in shadowing", func(t *testing.T) {
		// Test that later drop-ins override earlier ones
		tmpDir2 := t.TempDir()
		mainPath2 := filepath.Join(tmpDir2, "config.toml")
		dropinDir2 := filepath.Join(tmpDir2, "config.toml.d")
		os.Mkdir(dropinDir2, 0755)

		// Main config sets log level
		os.WriteFile(mainPath2, []byte(`log-level = "INFO"`), 0644)

		// Drop-in files that override each other
		os.WriteFile(filepath.Join(dropinDir2, "10-first.toml"), []byte(`log-level = "WARN"`), 0644)
		os.WriteFile(filepath.Join(dropinDir2, "20-second.toml"), []byte(`log-level = "DEBUG"`), 0644)

		cs := &ConfigSource{Path: mainPath2, DropInDir: dropinDir2}
		config, err := cs.Read()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// The last drop-in (20-second.toml) should win
		if config.LogLevel != slog.LevelDebug {
			t.Errorf("expected LogLevel=DEBUG, got %v", config.LogLevel)
		}
	})
}

func TestConfigSource_MissingDropinDir(t *testing.T) {
	tmpDir := t.TempDir()
	mainConfigPath := filepath.Join(tmpDir, "config.toml")
	dropinDir := filepath.Join(tmpDir, "config.toml.d") // doesn't exist

	// Write main config
	mainConfig := `log-level = "INFO"`
	if err := os.WriteFile(mainConfigPath, []byte(mainConfig), 0644); err != nil {
		t.Fatalf("failed to write main config: %v", err)
	}

	// Should not error when drop-in directory is missing
	cs := &ConfigSource{Path: mainConfigPath, DropInDir: dropinDir}
	config, err := cs.Read()
	if err != nil {
		t.Fatalf("unexpected error when drop-in dir missing: %v", err)
	}

	if config.LogLevel != slog.LevelInfo {
		t.Errorf("expected LogLevel=INFO, got %v", config.LogLevel)
	}
}

func TestEmbeddedDefault(t *testing.T) {
	// Test that the embedded default config is valid TOML
	dto, err := parseConfigDTO(defaultConfig)
	if err != nil {
		t.Fatalf("embedded default config is invalid: %v", err)
	}

	// Apply to Config
	config := Config{}
	config.Update(dto)

	// Verify the actual default values are loaded
	if config.CertFile != "" {
		t.Errorf("expected empty CertFile, got %s", config.CertFile)
	}
	if config.KeyFile != "" {
		t.Errorf("expected empty KeyFile, got %s", config.KeyFile)
	}
	if config.LogLevel != slog.LevelInfo {
		t.Errorf("expected LogLevel=INFO, got %v", config.LogLevel)
	}
	if config.CADir != "" {
		t.Errorf("expected empty CADir, got %s", config.CADir)
	}
}
