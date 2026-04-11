---
name: Bug Report
about: Report a bug or unexpected behavior
title: ""
labels: bug
assignees: ""
---

## Description

A clear description of what the bug is.

## Steps to Reproduce

1. Run `sudo usb-soundcard-mapper ...`
2. Select ...
3. See error

## Expected Behavior

What you expected to happen.

## Actual Behavior

What actually happened. Include full error output.

## Environment

- **OS:** (e.g., Ubuntu 24.04, Debian 12, Arch Linux)
- **Go version:** (if built from source)
- **Binary version:** (`usb-soundcard-mapper --help` header)
- **Sound system:** (ALSA, PulseAudio, PipeWire, JACK)

## Debug Output

```
# Run with debug logging and paste the output:
sudo usb-soundcard-mapper --log-level debug 2>&1
```

## Device Info

```
# Paste output of:
aplay -l
lsusb
```
