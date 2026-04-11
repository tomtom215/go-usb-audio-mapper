<!-- SPDX-License-Identifier: MIT -->

# ADR-0004: Command Execution Safety

## Status

Accepted

## Context

The tool executes system commands (`lsusb`, `aplay`, `udevadm`) as root. Shell injection through device serial numbers or user input is a real risk.

## Decision

- All commands executed via `exec.CommandContext` with `exec.LookPath` — never through a shell
- All command arguments validated against pre-compiled regex before execution
- Path arguments checked against `pathSafeRegex`; shell metacharacters (`&&`, `||`, `;`, backtick) rejected
- Configurable timeouts and retry with jittered backoff
- Process resources tracked and cleaned up on shutdown

## Consequences

- **Positive:** Shell injection is structurally impossible
- **Positive:** Hung commands cannot block the tool indefinitely
- **Negative:** Some legitimate arguments with special characters are rejected (acceptable tradeoff)
