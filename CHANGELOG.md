<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright 2025 Tom F. (https://github.com/tomtom215) -->

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Reliability hardening for unattended 24/7 field use.** New edge-case tests
  and end-to-end scenarios covering udev-unsafe serials, misconfigured
  timeouts/retries, unknown log levels, backup-filename collisions, and
  multi-device detection. The fake `udevadm` now serves per-card attribute walks
  (`udevadm_attr_walk_card<N>.txt`) so scenarios can give each attached recorder
  distinct attributes; `scripts/e2e.sh` gained cases for invalid product IDs and
  for verifying that clamped/normalized bad flags still complete cleanly.
- `isUdevSafeValue`/`hasUsableSerial` helpers and `uniqueTimestampedPath` for
  collision-free backups.
- **Hardware-free end-to-end test harness.** Fake `lsusb`/`aplay`/`udevadm` and
  sound-server commands under `testdata/fakebin/`, a Go integration layer that
  drives the full detection → install → verify pipeline against them, and
  `scripts/e2e.sh` which exercises the compiled binary's CLI surface. No USB
  audio hardware is required. Test functions grew from ~80 to 149 and statement
  coverage from 28.9% to 77.6%.
- Injectable `sysClassSoundPath` and `modprobeDir` seams so detection and
  verification can run against temporary directories in tests (runtime defaults
  are unchanged).
- `make e2e`, `make shellcheck`, and `make golangci` targets; `make all` now runs
  lint + shellcheck + test + e2e + build.
- CI now runs `shellcheck` over the scripts/fixtures and executes the end-to-end
  binary smoke test.

### Changed

- **`validateConfig` now self-heals misconfiguration instead of running broken.**
  Non-positive command/retry/wait timeouts and negative retry counts are clamped
  to safe defaults with a warning, and an unrecognized `--log-level` is
  normalized to `info` rather than silently ignored — so a mistyped flag in a
  remote deployment can no longer make every command time out instantly, skip
  execution entirely, or hide diagnostics.
- Refreshed dependencies: `golang.org/x/sys` 0.46.0 → 0.47.0,
  `golang.org/x/text` 0.39.0 → 0.40.0, and `github.com/mattn/go-isatty`
  0.0.22 → 0.0.23; `govulncheck ./...` remains clean.
- **Toolchain upgraded to Go 1.26** (`go.mod` `go 1.26.0` / `toolchain go1.26.5`);
  CI, README, and CONTRIBUTING updated accordingly. Minimum supported Go is now
  1.26.
- Migrated `.golangci.yml` to the golangci-lint **v2** schema (the previous v1
  configuration no longer loads under golangci-lint ≥ 2.0) and upgraded the CI
  `golangci-lint-action` to `v8`.
- Updated indirect dependencies to their latest releases, including
  `golang.org/x/text` 0.3.8 → 0.39.0 and `golang.org/x/sys` 0.38.0 → 0.46.0.
- Signal handling now enforces the graceful-shutdown deadline with an explicit
  timer instead of an unused derived context (also clears gosec G118).
- Hoisted the remaining per-call `regexp.MustCompile` calls in `device.go` to
  package scope.

### Fixed

- **Malformed udev rules from unusual serial numbers.** A USB serial containing
  a backslash, double quote, or control character was interpolated verbatim into
  `ATTRS{serial}=="…"`, producing a rule udev silently rejects so the mapping
  never applied. Such serials are now detected and the generator falls back to
  physical-port (or VID:PID) matching, which is always representable, keeping the
  rule well-formed. Naming and matching share one `hasUsableSerial` decision so
  they never diverge.
- **Backup files could overwrite one another.** Backup names used a
  second-granularity timestamp, so two backups created within the same second
  collided and a known-good rule could be lost; names are now made unique with a
  counter suffix.
- **Negative retry budget skipped execution and wrapped a nil error.** A
  negative `--retries` value caused commands never to run and returned a
  confusing `%!w(<nil>)` error; the retry loop now clamps the budget defensively.
- **Unhandled panics now fail cleanly.** `run()` recovers from any unexpected
  panic, logs it as structured data with a full stack, and exits with a distinct
  code (`2`) instead of crashing with a raw stack trace.
- Unhandled `*os.File.Close()` errors in the `atomicWriteFile` error paths
  (gosec G104).
- Replaced `WriteString(fmt.Sprintf(...))` with `fmt.Fprintf` throughout
  (staticcheck QF1012).
- Removed dead code: the unused `DeviceRegistry` instantiation in
  `GetUSBSoundCards`, the unused `subtitleStyle`, the unused `ExecTimeout` and
  `gracefulTimeout` constants, and a redundant vendor/product guard.
- Tightened the backups directory permissions to `0o750`.

### Security

- Resolved govulncheck **GO-2026-4602** (`os` standard library, fixed in
  go1.25.8) by building on the Go 1.26 toolchain; `govulncheck ./...` now reports
  no vulnerabilities.
- Documented the intentional subprocess-execution, directory-permission, and
  file-read patterns with justified `// #nosec` annotations so that a
  zero-exclusion `gosec` run is also clean.

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
