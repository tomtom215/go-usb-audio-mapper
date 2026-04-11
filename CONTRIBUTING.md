# Contributing to USB Soundcard Mapper

Thank you for your interest in contributing! This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.22 or later
- Linux system (this is a Linux-only tool)
- System packages: `alsa-utils`, `usbutils`, `udev`

### Building

```bash
# Clone the repository
git clone https://github.com/tomtom215/go-usb-audio-mapper.git
cd go-usb-audio-mapper

# Build
make build

# Run tests
make test

# Run tests with coverage
make test-cover

# Run all checks (lint + test + build)
make all
```

## Project Structure

```
.
├── main.go           # Entry point, flag parsing, orchestration
├── config.go         # Configuration types, validation, constants
├── errors.go         # Sentinel error definitions
├── device.go         # USBSoundCard type, registry, detection logic
├── command.go        # CommandExecutor, argument validation
├── transaction.go    # Transaction type with rollback support
├── resource.go       # ResourceTracker for cleanup management
├── fileops.go        # File locking, atomic writes, path helpers
├── udev.go           # Udev rule creation, installation, verification
├── backup.go         # Rule backup and udev system testing
├── system.go         # Privilege checks, permissions, signal handling
├── ui.go             # Bubble Tea terminal UI
├── operations.go     # Installation pipeline, non-interactive mode
├── *_test.go         # Test files (one per source file)
├── Makefile          # Build, test, lint targets
├── .goreleaser.yml   # Release automation
└── .github/
    └── workflows/
        └── ci.yml    # CI/CD pipeline
```

### Design Principles

- **Single responsibility**: Each file has one clear purpose
- **No file over 500 lines**: Keeps code navigable and maintainable
- **Security first**: All inputs validated, commands sanitized, paths checked
- **Transaction safety**: Critical operations use transactions with rollback
- **Resource tracking**: All resources tracked for clean shutdown

## Making Changes

1. Create a feature branch from `main`
2. Make your changes
3. Ensure all tests pass: `make test`
4. Ensure code is formatted: `gofmt -w .`
5. Ensure vet passes: `go vet ./...`
6. Submit a pull request

## Testing

Every source file should have a corresponding `_test.go` file. When adding new functionality:

- Add unit tests for all exported functions
- Add unit tests for important unexported functions
- Test error paths, not just happy paths
- Use table-driven tests where appropriate

Note: Functions that require root privileges, hardware access, or a terminal UI are difficult to unit test. Focus testing on the business logic and validation paths.

```bash
# Run all tests
make test

# Run specific test
go test -v -run TestFunctionName ./...

# Run with coverage
make test-cover
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Use meaningful variable and function names
- Add comments for non-obvious logic
- Use structured logging (`slog`) for all log output

## Reporting Issues

When reporting bugs, please include:

- Go version (`go version`)
- Linux distribution and version
- Sound system in use (ALSA, PulseAudio, PipeWire)
- Full error output with `--log-level debug`
- Output of `aplay -l` and `lsusb`
