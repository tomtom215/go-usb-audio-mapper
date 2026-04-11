<!-- SPDX-License-Identifier: MIT -->

# ADR-0001: Single Binary Architecture

## Status

Accepted

## Context

This tool runs as a system utility on Linux machines. It needs root privileges, access to `/dev`, `/sys`, and udev. Users install it alongside system packages.

## Decision

Ship as a single statically-linked Go binary with no external runtime dependencies. All code lives in the `main` package split across focused files. No internal library packages.

## Consequences

- **Positive:** Zero-dependency install (`curl` + `chmod`), easy cross-compilation, no shared library issues
- **Positive:** Single `main` package keeps the build simple and avoids import cycle risks
- **Negative:** Cannot be imported as a library (acceptable — this is a CLI tool)
