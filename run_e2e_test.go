// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

// End-to-end tests that drive run() — the whole CLI orchestration — in-process
// against fake commands and temp directories. These cover flag parsing through
// detection, installation and reporting without a real device or a subprocess.

package main

import (
	"flag"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// runCLI resets the global flag/arg/signal state, invokes run() with the given
// argv, and returns its exit code plus captured stdout.
func runCLI(t *testing.T, args ...string) (exitCode int, stdout string) {
	t.Helper()
	oldArgs := os.Args
	oldCmd := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCmd
		signal.Reset(os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGABRT)
	})

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	flag.CommandLine = fs
	os.Args = args

	stdout = captureStdout(t, func() { exitCode = run() })
	return exitCode, stdout
}

func TestRun_List(t *testing.T) {
	installFakeBin(t)
	fakeSysfs(t, "1")

	code, out := runCLI(t, "usb-soundcard-mapper", "--list", "--retries", "0", "--log-level", "error")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "Scarlett 2i2") || !strings.Contains(out, "1234:5678") {
		t.Errorf("--list output missing device details:\n%s", out)
	}
}

func TestRun_ListNoCards(t *testing.T) {
	scenario := installFakeBin(t)
	fakeSysfs(t, "0")
	scenarioFile(t, scenario, "aplay_l.txt",
		"card 0: PCH [HDA Intel PCH], device 0: ALC892 Analog [ALC892 Analog]\n")

	code, out := runCLI(t, "usb-soundcard-mapper", "--list", "--retries", "0", "--log-level", "error")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "No USB sound cards found") {
		t.Errorf("expected no-cards message, got:\n%s", out)
	}
}

func TestRun_NonInteractiveDryRun(t *testing.T) {
	installFakeBin(t)
	fakeSysfs(t, "1")

	code, out := runCLI(t, "usb-soundcard-mapper",
		"--non-interactive", "--vendor-id", "1234", "--product-id", "5678",
		"--dry-run", "--retries", "0", "--log-level", "error")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "Rule Content") || !strings.Contains(out, `ATTRS{idVendor}=="1234"`) {
		t.Errorf("dry-run should print the generated rule:\n%s", out)
	}
}

func TestRun_InvalidVendorID(t *testing.T) {
	// Config validation fails before any command execution.
	code, _ := runCLI(t, "usb-soundcard-mapper", "--vendor-id", "ZZZZ", "--log-level", "error")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 for invalid vendor ID", code)
	}
}

func TestRun_FullNonInteractiveInstall(t *testing.T) {
	// The non-interactive install path (not --list, not --dry-run) requires
	// root privileges in run(); skip when the test process is not elevated
	// (e.g. on CI runners). The installation logic itself is covered without
	// root by TestNonInteractiveMode_Success.
	if os.Geteuid() != 0 {
		t.Skip("requires root privileges")
	}

	installFakeBin(t)
	fakeSysfs(t, "1")
	fakeModprobeDir(t, true)
	rulesDir := t.TempDir()

	code, _ := runCLI(t, "usb-soundcard-mapper",
		"--non-interactive", "--vendor-id", "1234", "--product-id", "5678",
		"--rules-path", rulesDir, "--skip-reload", "--retries", "0", "--log-level", "error")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	rule := filepath.Join(rulesDir, "89-usb-soundcard-1234-5678.rules")
	if _, err := os.Stat(rule); err != nil {
		t.Errorf("expected installed rule file %s: %v", rule, err)
	}
}
