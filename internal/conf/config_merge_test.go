package conf

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// TestMissingKeysInDropin tests what happens when a drop-in file
// doesn't specify certain keys - they should NOT overwrite the base config
func TestMissingKeysInDropin(t *testing.T) {
	tmpDir := t.TempDir()
	mainConfigPath := filepath.Join(tmpDir, "config.toml")
	dropinDir := filepath.Join(tmpDir, "config.toml.d")
	os.Mkdir(dropinDir, 0755)

	// Main config has all values set
	mainConfig := `
cert-file = "/etc/rhc/main.pem"
key-file = "/etc/rhc/main.key"
log-level = "INFO"
ca-dir = "/etc/pki/tls/certs"
`
	os.WriteFile(mainConfigPath, []byte(mainConfig), 0644)

	// Drop-in file only sets log-level, nothing else
	// The other fields should be preserved from main config
	dropinConfig := `
log-level = "DEBUG"
`
	os.WriteFile(filepath.Join(dropinDir, "10-debug.toml"), []byte(dropinConfig), 0644)

	cs := &ConfigSource{Path: mainConfigPath, DropInDir: dropinDir}
	config, err := cs.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: only log-level is overridden, everything else from main config
	if config.CertFile != "/etc/rhc/main.pem" {
		t.Errorf("expected CertFile=/etc/rhc/main.pem (preserved!), got %s", config.CertFile)
	}
	if config.KeyFile != "/etc/rhc/main.key" {
		t.Errorf("expected KeyFile=/etc/rhc/main.key (preserved!), got %s", config.KeyFile)
	}
	if config.LogLevel != slog.LevelDebug {
		t.Errorf("expected LogLevel=DEBUG (overridden), got %v", config.LogLevel)
	}
	if config.CADir != "/etc/pki/tls/certs" {
		t.Errorf("expected CADir=/etc/pki/tls/certs (preserved!), got %s", config.CADir)
	}
}

// TestEmptyStringOverwrite tests if we can actually set values to empty strings
func TestEmptyStringOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	mainConfigPath := filepath.Join(tmpDir, "config.toml")
	dropinDir := filepath.Join(tmpDir, "config.toml.d")
	os.Mkdir(dropinDir, 0755)

	// Main config has non-empty values
	mainConfig := `
cert-file = "/etc/rhc/cert.pem"
ca-dir = "/etc/pki/tls/certs"
`
	os.WriteFile(mainConfigPath, []byte(mainConfig), 0644)

	// Drop-in tries to set them to empty values
	dropinConfig := `
cert-file = ""
ca-dir = ""
`
	os.WriteFile(filepath.Join(dropinDir, "10-override.toml"), []byte(dropinConfig), 0644)

	cs := &ConfigSource{Path: mainConfigPath, DropInDir: dropinDir}
	config, err := cs.Read()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// This test verifies that empty string values can be set
	t.Logf("CertFile: got %q, want %q", config.CertFile, "")
	t.Logf("CADir: got %q, want %q", config.CADir, "")

	// Check if empty values were applied
	if config.CertFile != "" {
		t.Errorf("cert-file was not overridden to empty: got %s", config.CertFile)
	}
	if config.CADir != "" {
		t.Errorf("ca-dir was not overridden to empty: got %s", config.CADir)
	}
}
