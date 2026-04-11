<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright 2025 Tom F. (https://github.com/tomtom215) -->

# Contributing to USB Soundcard Mapper

Thank you for your interest in contributing!

## Development Setup

### Prerequisites

- Go 1.25 or later
- Linux system (this is a Linux-only tool)
- System packages: `alsa-utils`, `usbutils`, `udev`

### Building

```bash
git clone https://github.com/tomtom215/go-usb-audio-mapper.git
cd go-usb-audio-mapper

make build       # Build the binary
make test        # Run tests
make test-cover  # Run tests with coverage
make all         # Lint + test + build
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
```

## Testing

Every source file should have a corresponding `_test.go` file.

### Test Categories

| Type | Location | Command |
|------|----------|---------|
| Unit tests | `*_test.go` | `go test ./...` |
| Race detection | `*_test.go` | `go test -race ./...` |
| Coverage | `*_test.go` | `make test-cover` |

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
- Functions requiring root/hardware/terminal are exempt from unit tests

## Quality Gates

All of the following must pass before merging:

1. `gofmt -l .` produces no output
2. `go vet ./...` passes
3. `go test -race ./...` passes
4. `go build ./...` succeeds
5. No file exceeds 500 lines
6. SPDX headers on all new files
7. CHANGELOG.md updated for user-facing changes

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
