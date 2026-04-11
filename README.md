<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright 2025 Tom F. (https://github.com/tomtom215) -->

# USB Soundcard Mapper

[![CI](https://github.com/tomtom215/go-usb-audio-mapper/actions/workflows/ci.yml/badge.svg)](https://github.com/tomtom215/go-usb-audio-mapper/actions/workflows/ci.yml)
[![Security](https://github.com/tomtom215/go-usb-audio-mapper/actions/workflows/security.yml/badge.svg)](https://github.com/tomtom215/go-usb-audio-mapper/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/tomtom215/go-usb-audio-mapper/graph/badge.svg)](https://codecov.io/gh/tomtom215/go-usb-audio-mapper)
[![Go Report Card](https://goreportcard.com/badge/github.com/tomtom215/go-usb-audio-mapper)](https://goreportcard.com/report/github.com/tomtom215/go-usb-audio-mapper)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/badge/go-1.24%2B-blue.svg)](https://go.dev/)

A production-grade utility for creating persistent udev mappings for USB audio devices on Linux systems.

## Overview

USB Soundcard Mapper solves a common problem for audio professionals and enthusiasts working with USB audio interfaces on Linux: ensuring consistent device naming across disconnects, reconnects, and system reboots.

When you connect multiple USB audio devices to a Linux system, they are assigned arbitrary card numbers (e.g., `card0`, `card1`) based on the order they are detected. These card numbers can change when devices are reconnected or when the system reboots, which causes issues with audio applications that reference specific device names.

This utility creates persistent udev rules that assign consistent, meaningful names to your USB audio devices, ensuring they remain stable regardless of connection order or system reboots.

### Key Features

- Automatic detection of USB audio devices with detailed information extraction
- Interactive terminal UI for device selection and naming (Bubble Tea)
- Non-interactive mode for scripting and automation
- Transaction-based operations with automatic rollback on failure
- Comprehensive logging with configurable verbosity (structured JSON)
- Automatic validation and verification of applied rules
- Support for virtual audio devices (with safety prompts)
- Atomic file writes with file locking
- Backup creation of existing rules
- Graceful signal handling (SIGINT, SIGTERM)

## System Requirements

- Linux system with udev (most modern distributions)
- Required commands: `lsusb`, `aplay`, `udevadm`
- Root privileges (for writing udev rules)
- Go 1.24+ (for building from source)

## Installation

### From Binary Releases

Download the latest release from the [GitHub Releases](https://github.com/tomtom215/go-usb-audio-mapper/releases) page:

```bash
# Download the latest release (replace X.Y.Z with the version number)
curl -LO https://github.com/tomtom215/go-usb-audio-mapper/releases/download/vX.Y.Z/usb-soundcard-mapper_X.Y.Z_linux_amd64.tar.gz

# Extract
tar xzf usb-soundcard-mapper_X.Y.Z_linux_amd64.tar.gz

# Install (requires root)
sudo install -m 755 usb-soundcard-mapper /usr/local/bin/
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/tomtom215/go-usb-audio-mapper.git
cd go-usb-audio-mapper

# Build using Make
make build

# Or build directly with Go
go build -o usb-soundcard-mapper .

# Install to system (requires root)
sudo make install
```

### Development

```bash
# Run all checks (lint + test + build)
make all

# Run tests with coverage
make test-cover

# See all available targets
make help
```

## Usage

### Interactive Mode

The easiest way to use the utility is in interactive mode:

```bash
# Must be run with root privileges
sudo usb-soundcard-mapper
```

This will:
1. Detect all USB sound cards connected to your system
2. Present an interactive terminal UI for selecting a device
3. Allow you to customize the device name or use the suggested one
4. Create and apply the udev rule
5. Verify the rule has been successfully applied

### Non-Interactive Mode

For automation and scripting, use the non-interactive mode:

```bash
sudo usb-soundcard-mapper --non-interactive --vendor-id 1234 --product-id 5678 --name my_audio_interface
```

### List Mode

To view all connected USB audio devices without making changes:

```bash
usb-soundcard-mapper --list
```

### Dry Run Mode

To see what changes would be made without actually applying them:

```bash
sudo usb-soundcard-mapper --dry-run
```

### Command Line Options

```
Usage: usb-soundcard-mapper [options]

Creates persistent device mappings for USB sound cards.

Options:
  --rules-path string       Path to udev rules directory (default "/etc/udev/rules.d")
  --list                    List USB sound cards and exit
  --non-interactive         Non-interactive mode
  --name string             Custom name for the device (non-interactive mode)
  --vendor-id string        Vendor ID (non-interactive mode)
  --product-id string       Product ID (non-interactive mode)
  --skip-reload             Skip reloading udev rules after creating them
  --dry-run                 Show what would be done without making changes
  --force                   Force overwrite existing rules and accept virtual devices
  --ignore-virtual          Ignore virtual audio devices
  --max-backups int         Maximum number of backups to keep per device (default 10)
  --log-level string        Log level: debug, info, warn, error (default "info")
  --command-timeout int     Command execution timeout in seconds (default 5)
  --lock-timeout int        File lock acquisition timeout in seconds (default 2)
  --graceful-timeout int    Graceful shutdown timeout in seconds (default 5)
  --retries int             Maximum number of retries for commands (default 3)
```

## Architecture

```
┌─────────────────────────────────────────────────┐
│  CLI (main.go)                                  │
│  Flag parsing, orchestration, signal handling   │
└──────────────────────┬──────────────────────────┘
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
┌──────────────┐ ┌──────────┐ ┌──────────────────┐
│ Interactive  │ │  List    │ │ Non-Interactive   │
│ UI (ui.go)   │ │  Mode   │ │ (operations.go)   │
│ Bubble Tea   │ │         │ │ Scripting/CI      │
└──────┬───────┘ └─────────┘ └────────┬──────────┘
       │                              │
       └──────────────┬───────────────┘
                      ▼
┌─────────────────────────────────────────────────┐
│  Device Detection (device.go)                   │
│  aplay -l → udevadm → lsusb → USBSoundCard     │
└──────────────────────┬──────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────┐
│  Udev Rule Engine (udev.go)                     │
│  Rule creation, installation, verification      │
└──────────────────────┬──────────────────────────┘
                       │
       ┌───────────────┼───────────────┐
       ▼               ▼               ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│ Transaction │ │ Atomic File │ │  Command    │
│ (rollback)  │ │ Writes      │ │  Executor   │
│             │ │ (file lock) │ │  (safe exec)│
└─────────────┘ └─────────────┘ └─────────────┘
```

## How It Works

1. The utility detects all USB audio devices connected to your system
2. For each device, it extracts:
   - Vendor and product IDs
   - Serial number (if available)
   - Physical port information
   - Bus and device information
   - Vendor and product names
3. It creates a udev rule that:
   - Applies a consistent name to the device based on its unique attributes
   - Creates symbolic links for easier application access
   - Handles different udev event types (add, change) for robustness
4. The rule is installed in `/etc/udev/rules.d/`
5. Udev rules are reloaded and triggered to apply the changes

## Project Status

| Phase | Status |
|-------|--------|
| Core device detection | Done |
| Udev rule generation (9 rule types per device) | Done |
| Interactive terminal UI (Bubble Tea) | Done |
| Non-interactive mode for automation | Done |
| Transaction-based operations with rollback | Done |
| Atomic file writes with locking | Done |
| Command execution safety (injection prevention) | Done |
| Signal handling and graceful shutdown | Done |
| Modular architecture (13 files, all <500 lines) | Done |
| Comprehensive test suite (80+ tests) | Done |
| CI/CD pipeline (lint, test, build, release) | Done |
| Security scanning (govulncheck) | Done |
| Production documentation (SECURITY, CHANGELOG, ADRs) | Done |

## Project Structure

```
.
├── main.go              # Entry point, flag parsing, orchestration
├── config.go            # Configuration types, validation, constants, regex
├── errors.go            # Sentinel error definitions
├── device.go            # USBSoundCard type, registry, detection, helpers
├── command.go           # CommandExecutor, argument safety validation
├── transaction.go       # Transaction type with atomic rollback
├── resource.go          # ResourceTracker for lifecycle management
├── fileops.go           # File locking, atomic writes, path helpers
├── udev.go              # Udev rule creation, installation, verification
├── backup.go            # Rule backup and udev system testing
├── system.go            # Privileges, permissions, signal handling, logging
├── ui.go                # Bubble Tea interactive terminal UI
├── operations.go        # Installation pipeline, non-interactive mode
├── *_test.go            # Tests (one per source file)
├── docs/adr/            # Architecture Decision Records
├── Makefile             # Build, test, lint targets
├── .golangci.yml        # Linter configuration
├── .goreleaser.yml      # Release automation config
├── codecov.yml          # Coverage targets
├── CHANGELOG.md         # Keep a Changelog format
├── SECURITY.md          # Vulnerability reporting policy
├── GOVERNANCE.md        # Project governance
├── RELEASING.md         # Release process checklist
├── CITATION.cff         # Software citation metadata
└── .github/
    ├── workflows/       # CI, coverage, security workflows
    ├── ISSUE_TEMPLATE/  # Bug report, feature request templates
    └── PULL_REQUEST_TEMPLATE.md
```

## Best Practices

### Device Naming

When naming your devices, consider:

- Use descriptive names that identify the function or model
- Avoid spaces or special characters
- Keep names short but meaningful
- Use prefixes for different types of interfaces (e.g., `mic_`, `mixer_`)

Example: `focusrite_2i2` for a Focusrite Scarlett 2i2 interface

### For Audio Production Systems

- Create mappings for all your devices before setting up your audio software
- Verify each mapping by disconnecting and reconnecting the device
- Consider creating a backup of your working udev rules configuration
- Test with your audio software to ensure it correctly identifies the devices

## Troubleshooting

### Common Issues

#### Device Not Detected

- Ensure the device is properly connected and powered on
- Check if the device appears in `lsusb` output
- Check if the device appears in `aplay -l` output
- Try a different USB port or cable

#### Name Not Applied After Rule Creation

- Disconnect and reconnect the device
- Run `sudo udevadm control --reload-rules && sudo udevadm trigger --action=add --subsystem-match=sound`
- Check system logs: `journalctl -u systemd-udevd`
- Verify the rule file exists: `ls -l /etc/udev/rules.d/89-usb-soundcard-*.rules`

#### Permission Denied Errors

- Ensure you're running the utility with `sudo`
- Check permissions on the udev rules directory: `ls -la /etc/udev/rules.d/`
- Verify your user has sudo privileges

#### Conflicts with Existing Rules

- Use `--force` to overwrite existing rules
- Manually check for conflicting rules: `grep -r "ATTRS{idVendor}" /etc/udev/rules.d/`
- Backups are created automatically (use `--max-backups` to control count)

### Debugging Techniques

```bash
# Enable debug logging
sudo usb-soundcard-mapper --log-level debug

# Test udev rule manual application
sudo udevadm test $(udevadm info --query=path --name=/dev/snd/cardX)

# View detailed device information
sudo udevadm info --attribute-walk --name=/dev/snd/cardX

# Monitor udev events
sudo udevadm monitor --environment --udev
```

## Uninstallation

### Remove Created Rules

```bash
# List all rules created by the utility
ls /etc/udev/rules.d/89-usb-soundcard-*.rules

# Remove a specific rule
sudo rm /etc/udev/rules.d/89-usb-soundcard-XXXX-YYYY.rules

# Reload udev rules
sudo udevadm control --reload-rules
```

### Complete Uninstallation

```bash
# Remove all created rules
sudo rm /etc/udev/rules.d/89-usb-soundcard-*.rules

# Remove modprobe configurations (if created)
sudo rm /etc/modprobe.d/99-soundcard-*.conf

# Reload udev rules
sudo udevadm control --reload-rules
sudo udevadm trigger

# Remove the binary (if installed)
sudo rm /usr/local/bin/usb-soundcard-mapper
```

## Advanced Usage

### Integration with Audio Software Setup Scripts

```bash
# Map all connected USB audio devices with default names
for device in $(usb-soundcard-mapper --list | grep VID:PID | awk '{print $3}'); do
  vid=$(echo $device | cut -d: -f1)
  pid=$(echo $device | cut -d: -f2)
  sudo usb-soundcard-mapper --non-interactive --vendor-id $vid --product-id $pid
done
```

### Custom Rules Directory

```bash
sudo usb-soundcard-mapper --rules-path /path/to/custom/rules/dir
```

### Handling Virtual Devices

```bash
# Skip virtual devices entirely
sudo usb-soundcard-mapper --ignore-virtual

# Force mapping of virtual devices
sudo usb-soundcard-mapper --force
```

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, quality gates, and PR checklist.

## Security

To report a vulnerability, see [SECURITY.md](SECURITY.md).

## License

[MIT License](LICENSE) - See the LICENSE file for details.
