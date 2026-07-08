<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright 2025 Tom F. (https://github.com/tomtom215) -->

# Contributing to USB Soundcard Mapper

Thank you for your interest in contributing!

## Development Setup

### Prerequisites

- Go 1.26 or later
- Linux system (this is a Linux-only tool)
- System packages: `alsa-utils`, `usbutils`, `udev`
- Optional (for the full `make all`): `shellcheck`

### Building

```bash
git clone https://github.com/tomtom215/go-usb-audio-mapper.git
cd go-usb-audio-mapper

make build       # Build the binary
make test        # Run tests (race detector)
make test-cover  # Run tests with coverage
make e2e         # Hardware-free end-to-end smoke test of the binary
make shellcheck  # Lint shell scripts and fake-command fixtures
make all         # Lint + shellcheck + test + e2e + build
make help        # Show all targets
```

## Coding Standards

### File Headers

Every `.go` file must start with SPDX license headers:

```go
// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)
```

### File Size

Source files should stay under **500 lines**. When a file grows past this, split it into focused modules with single responsibilities.

### Error Handling

- Use `fmt.Errorf("context: %w", err)` to wrap errors with context
- Use sentinel errors (defined in `errors.go`) for specific failure cases
- Never use `panic()` in library/production code
- `os.Exit()` only in `main()`

### Naming

- Follow standard Go conventions
- Use meaningful, descriptive names
- Test functions: `Test{Component}_{Scenario}_{Expected}` (e.g., `TestCleanupName_StartsWithNumber_AddPrefix`)

### Logging

- Use `log/slog` for all log output (structured JSON)
- Use appropriate levels: `Debug` for verbose, `Info` for normal, `Warn` for recoverable, `Error` for failures

## Project Structure

```
main.go           Entry point, flag parsing, orchestration
config.go         Configuration types, validation, constants, regex
errors.go         Sentinel error definitions
device.go         USBSoundCard type, registry, detection, helpers
command.go        CommandExecutor, argument safety validation
transaction.go    Transaction type with atomic rollback
resource.go       ResourceTracker for lifecycle management
fileops.go        File locking, atomic writes, path helpers
udev.go           Udev rule creation, installation, verification
backup.go         Rule backup and udev system testing
system.go         Privileges, permissions, signal handling, logging
ui.go             Bubble Tea interactive terminal UI
operations.go     Installation pipeline, non-interactive mode
testdata/fakebin/ Fake lsusb/aplay/udevadm commands for hardware-free tests
scripts/e2e.sh    End-to-end smoke test of the compiled binary
```

## Testing

Every source file should have a corresponding `_test.go` file.

### Test Categories

| Type | Location | Command |
|------|----------|---------|
| Unit tests | `*_test.go` | `go test ./...` |
| Hardware-free E2E | `*_e2e_test.go` | `go test ./...` |
| Race detection | `*_test.go` | `go test -race ./...` |
| Coverage | `*_test.go` | `make test-cover` |
| Binary smoke test | `scripts/e2e.sh` | `make e2e` |

### Hardware-free testing

Detection and installation shell out to `lsusb`, `aplay`, and `udevadm`. Tests
never touch real hardware or `/etc`: the fake commands in `testdata/fakebin/`
are placed on `PATH`, and the `sysClassSoundPath` / `modprobeDir` package
variables (plus `--rules-path`) are redirected to temp directories. Scenarios
are steered by dropping override files into a `FAKE_DEV_DIR` (see
`helpers_test.go`). Keep the fakes `shellcheck`-clean.

### Running Tests

```bash
make test                               # All tests with race detector
go test -v -run TestFunctionName ./...  # Specific test
make test-cover                         # Coverage report
```

### What to Test

- All exported functions
- Important unexported functions
- Error paths, not just happy paths
- Use table-driven tests where appropriate
- Functions that shell out to system commands or read hardware paths are tested
  via the fake commands in `testdata/fakebin/` (see "Hardware-free testing");
  only the interactive terminal program (`runUI`) is exempt

## Quality Gates

All of the following must pass before merging:

1. `gofmt -l .` produces no output
2. `go vet ./...` passes
3. `golangci-lint run ./...` passes (v2 config in `.golangci.yml`)
4. `go test -race ./...` passes
5. `go build ./...` succeeds
6. `make e2e` passes and `make shellcheck` is clean
7. `govulncheck ./...` reports no vulnerabilities
8. No file exceeds 500 lines
9. SPDX headers on all new files
10. CHANGELOG.md updated for user-facing changes

## PR Checklist

- [ ] Code compiles cleanly
- [ ] All tests pass with race detector
- [ ] Code is formatted with `gofmt`
- [ ] `go vet` passes
- [ ] New code has tests
- [ ] SPDX license headers on new files
- [ ] No file exceeds 500 lines
- [ ] CHANGELOG.md updated (if user-facing)

## Reporting Issues

When reporting bugs, include:

- Go version (`go version`)
- Linux distribution and version
- Sound system in use (ALSA, PulseAudio, PipeWire)
- Full error output with `--log-level debug`
- Output of `aplay -l` and `lsusb`
