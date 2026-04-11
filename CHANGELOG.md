<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright 2025 Tom F. (https://github.com/tomtom215) -->

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.1.0] — 2026-04-11

### Breaking Changes

- `DeviceRegistry` no longer uses `USBSoundCard` as map key (contained uncomparable fields)
- `CheckCommands()` no longer requires `context.Context` or `*CommandExecutor` parameters
- `uiModel.operationLock` changed from `sync.Mutex` to `*sync.Mutex` (fixes go vet violations)

### Added

- **Modular architecture**: Refactored monolithic `main.go` (3,200+ lines) into 13 focused modules, each under 500 lines with single responsibilities
- `go.mod` / `go.sum` — proper Go module support
- Comprehensive test suite (80+ tests across 8 test files) with race detector
- `Makefile` with `build`, `test`, `test-cover`, `lint`, `vet`, `fmt`, `clean`, `install` targets
- `.github/workflows/ci.yml` — CI pipeline across Go 1.22-1.24, multi-arch builds, automated releases
- `.goreleaser.yml` — multi-arch release automation (amd64, arm64, armv7)
- `.gitignore` — standard Go project ignores
- `CONTRIBUTING.md` — development guide with project structure and design principles
- `SECURITY.md` — vulnerability reporting and 90-day coordinated disclosure policy
- `CHANGELOG.md` — Keep a Changelog format
- `GOVERNANCE.md` — project governance, roles, and decision-making
- `RELEASING.md` — release process checklist
- `CITATION.cff` — academic/software citation metadata
- `codecov.yml` — coverage targets and exclusions
- `.golangci.yml` — comprehensive linter configuration
- GitHub issue templates (bug report, feature request) and PR template
- Architecture Decision Records (`docs/adr/`)
- SPDX license headers on every source file
- AI ethics notice in main entry point
- Pre-compiled `unsafeCharsRegex` for serial number sanitization
- Pre-compiled `nonAlphaNumRegex` for name cleanup
- Pre-compiled `cardRegex` for aplay output parsing

### Fixed

- **`sanitizeSerial()` was inverted** — replaced valid strings with `"_"` instead of replacing dangerous characters
- **Format string `%s` appeared literally in output** — `WriteString()` used instead of `Sprintf` in success message
- **Panic-prone VID:PID parsing** — `strings.Split(result, "(VID:PID ")[1][:4]` would panic on index out of range
- **`sync.Mutex` copied by value** — 20+ `go vet` violations from Bubble Tea model copying embedded mutex
- **Duplicate regex compilation** — `regexp.MustCompile()` called inside functions on every invocation
- `strings.Replace` with count `-1` replaced with `strings.ReplaceAll`
- Duplicate `// Check` comment removed

### Changed

- Version bumped from 2.0.2 to 2.1.0
- `DeviceRegistry` uses generated string keys instead of struct map keys
- README updated with badges, project structure, complete CLI documentation

## [2.0.2] — 2025-12-01

### Added

- Production hardening release
- Transaction-based operations with automatic rollback
- Resource tracking with cleanup on shutdown
- Graceful signal handling (SIGINT, SIGTERM, SIGHUP, SIGQUIT)
- Atomic file writes with file locking via `gofrs/flock`
- Configurable timeouts for all operations
- Structured JSON logging via `log/slog`
- Backup creation of existing rules (max 10 per device)
- Virtual audio device detection and safety prompts
- Multiple udev rule matching strategies per device
- PCM playback/capture symlink creation
- File size limits and resource exhaustion protection

## [2.0.0] — 2025-11-15

### Added

- Complete rewrite with Interactive terminal UI (Bubble Tea framework)
- Non-interactive mode for scripting and automation
- Device validation with vendor/product ID format checking
- Serial number sanitization
- Physical port-based device identification
- Dry-run mode for previewing changes
- Modprobe configuration generation

## [1.0.0] — 2025-10-01

### Added

- Initial release
- Basic USB sound card detection via `aplay -l`
- Udev rule generation with vendor/product ID matching
- Command-line interface

[Unreleased]: https://github.com/tomtom215/go-usb-audio-mapper/compare/v2.1.0...HEAD
[2.1.0]: https://github.com/tomtom215/go-usb-audio-mapper/compare/v2.0.2...v2.1.0
[2.0.2]: https://github.com/tomtom215/go-usb-audio-mapper/compare/v2.0.0...v2.0.2
[2.0.0]: https://github.com/tomtom215/go-usb-audio-mapper/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/tomtom215/go-usb-audio-mapper/releases/tag/v1.0.0
