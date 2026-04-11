<!-- SPDX-License-Identifier: MIT -->

# ADR-0003: Transaction-Based File Operations

## Status

Accepted

## Context

The tool writes system configuration files (`/etc/udev/rules.d/`, `/etc/modprobe.d/`). A partial write (e.g., power loss, signal interrupt) could leave the system in an inconsistent state with broken udev rules.

## Decision

All multi-step file operations use a `Transaction` type that executes operations sequentially and rolls back in reverse order on failure. File writes use atomic temp-file-then-rename to prevent partial writes.

## Consequences

- **Positive:** System is never left in a half-configured state
- **Positive:** Existing rules are backed up before modification
- **Positive:** Failed installations clean up after themselves
- **Negative:** Slightly more code complexity than direct writes
