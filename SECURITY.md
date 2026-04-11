<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright 2025 Tom F. (https://github.com/tomtom215) -->

# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 2.1.x   | Yes       |
| 2.0.x   | Yes       |
| < 2.0   | No        |

## Scope

This policy covers the `usb-soundcard-mapper` binary and all source code in this repository. Since the tool operates with root privileges and writes system configuration files (udev rules, modprobe configs), security is treated as a first-class concern.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please use one of the following channels:

1. **GitHub Security Advisories** (preferred): [Create a draft advisory](https://github.com/tomtom215/go-usb-audio-mapper/security/advisories/new)
2. **Email**: Report directly via GitHub profile contact

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Affected version(s)
- Impact assessment (what an attacker could achieve)
- Any proposed fix (optional but appreciated)

## Disclosure Process

We follow a **90-day coordinated disclosure** timeline:

| Phase | Timeline | Action |
|-------|----------|--------|
| Acknowledge | 0-3 business days | Confirm receipt of report |
| Triage | Days 1-14 | Assess severity, identify affected code |
| Fix | Days 15-90 | Develop, test, and release patch |
| Disclose | Day 90 | Public disclosure with CVE if applicable |

Security fixes may expedite this timeline for critical vulnerabilities.

## Security Considerations

This tool has a significant security surface because it:

- **Runs as root** to write udev rules
- **Executes system commands** (lsusb, aplay, udevadm)
- **Writes to system directories** (/etc/udev/rules.d, /etc/modprobe.d)

### Built-in Protections

- Command argument validation to prevent shell injection
- Path validation to prevent directory traversal
- Input sanitization for serial numbers and device names
- Atomic file writes with file locking
- Pre-compiled regex for all validation patterns
- `exec.LookPath` for command resolution (no shell evaluation)
- File size limits to prevent resource exhaustion
- Transaction rollback on failure

## Attribution

Security researchers will be credited in release notes and advisories unless anonymity is requested.
