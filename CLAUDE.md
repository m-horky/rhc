# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`rhc` is a Go-based CLI client for connecting RHEL systems to Red Hat services. It serves as a front-end alternative to `subscription-manager` and `insights-client`, performing three main steps: registering with Red Hat Subscription Management, registering with Red Hat Insights, and activating the `yggdrasil`/`rhcd` service.

## Build System

This project uses the Meson build system:

```bash
# Build the project
meson setup builddir
meson compile -C builddir

# The resulting binary will be in builddir/rhc
```

For package creation:
```bash
# Create RPM package using packit
packit build locally
```

## Testing

### Unit Tests
```bash
# Run Go unit tests
go test ./...

# Run specific package tests
go test ./internal/systemd
```

### Integration Tests
Integration tests are located in `./integration-tests/` and require root privileges:

```bash
# Install requirements
dnf -y install python3-pip python3-pytest tmt+all
pip install -r integration-tests/requirements.txt

# Run integration tests locally (requires settings.toml configuration)
export ENV_FOR_DYNACONF=local
pytest -s -vvv --log-level=DEBUG

# Run with tmt
tmt --root . -vvv run --all --environment ENV_FOR_DYNACONF=prod
```

## Code Architecture

The codebase follows Go best practices with a modular structure:

### Package Structure
- **`cmd/rhc/`**: Main CLI application entry point
  - `main.go` - CLI setup and global flags
  - `connect.go`, `disconnect.go`, `status.go` - Command implementations
  - `canonical_facts.go`, `collector.go` - Utility commands
- **`pkg/`**: Reusable business logic modules
  - `config/` - Configuration and constants management
  - `rhsm/` - Red Hat Subscription Management integration
  - `insights/` - Red Hat Insights client integration
  - `activation/` - Service activation using systemd
  - `facts/` - System canonical facts collection
  - `features/` - Feature flag handling
  - `interactive/` - UI components (spinners, colors, formatting)
  - `util/` - General utility functions
  - `logging/` - Logging utilities and message handling
- **`internal/`**: Internal packages not meant for external use
  - `systemd/` - systemd D-Bus integration
  - `http/` - HTTP client utilities

### Command Structure
The application uses `urfave/cli/v2` for command-line interface with support for:
- Machine-readable JSON output (`--format json`)
- Configuration files in TOML format
- Structured result types for each command (ConnectResult, DisconnectResult, SystemStatus)

### Key Dependencies
- `github.com/urfave/cli/v2` - CLI framework
- `github.com/coreos/go-systemd/v22` - systemd integration
- `github.com/godbus/dbus/v5` - D-Bus communication
- `github.com/briandowns/spinner` - CLI spinners

## Development Guidelines

### Code Style
- Follow [Effective Go](https://go.dev/doc/effective_go), [CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments), and [Go Proverbs](https://go-proverbs.github.io/)
- Use [Conventional Commits](https://www.conventionalcommits.org) for commit messages
- Communicate errors through return values, not logging
- Code should exist in a package only if it can be useful when imported exclusively

### Remote Debugging
For debugging in VMs, use delve:
```bash
sudo go install github.com/go-delve/delve/cmd/dlv@latest
sudo /root/go/bin/dlv debug --api-version 2 --headless --listen 0.0.0.0:2345 ./ -- connect --username NNN --password ***
```

## Key Files
- `meson.build` - Build configuration with version and ldflags setup
- `go.mod` - Go module dependencies (Go 1.23.0+)
- `constants.go` - Application constants
- `util.go` - Utility functions
- `features.go` - Feature flag handling
- `activate.go` - Service activation logic
