<!-- SPDX-License-Identifier: MIT -->

# ADR-0002: Modular File Organization

## Status

Accepted

## Context

The original implementation was a single 3,200-line `main.go`. This made navigation, testing, and maintenance difficult.

## Decision

Split into 13 focused files, each under 500 lines, with single responsibilities:

- `config.go` — types, validation, constants
- `errors.go` — sentinel errors
- `device.go` — USB device types, detection
- `command.go` — safe command execution
- `transaction.go` — atomic operations with rollback
- `resource.go` — resource lifecycle tracking
- `fileops.go` — file locking, atomic writes
- `udev.go` — rule creation, installation, verification
- `backup.go` — rule backup, system testing
- `system.go` — privileges, signals, logging
- `ui.go` — Bubble Tea terminal UI
- `operations.go` — installation pipeline

## Consequences

- **Positive:** Each file is self-contained and testable in isolation
- **Positive:** New contributors can understand one file at a time
- **Positive:** Git blame and diffs are meaningful per-file
- **Negative:** Slightly more files to navigate (mitigated by clear naming)
