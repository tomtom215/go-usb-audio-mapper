// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"flag"
	"io"
	"os"
	"testing"
	"time"
)

// withArgs installs a fresh flag set and os.Args for a single parseFlags call,
// restoring the originals on cleanup. parseFlags uses the global flag package,
// so each invocation needs a clean CommandLine to avoid "flag redefined".
func withArgs(t *testing.T, args []string) {
	t.Helper()
	oldArgs := os.Args
	oldCmd := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCmd
	})
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	flag.CommandLine = fs
	os.Args = args
}

func baseConfig() Config {
	return Config{
		UdevRulesPath: udevRulesDir,
		Timeouts:      DefaultTimeouts,
		MaxRetries:    3,
	}
}

func TestParseFlags_Defaults(t *testing.T) {
	withArgs(t, []string{"usb-soundcard-mapper"})

	cfg := baseConfig()
	parseFlags(&cfg)

	if cfg.UdevRulesPath != udevRulesDir {
		t.Errorf("UdevRulesPath = %q, want %q", cfg.UdevRulesPath, udevRulesDir)
	}
	if cfg.ListOnly || cfg.NonInteractive || cfg.DryRun || cfg.ForceOverwrite {
		t.Error("boolean flags should default to false")
	}
	if cfg.MaxBackupCount != maxBackupCount {
		t.Errorf("MaxBackupCount = %d, want %d", cfg.MaxBackupCount, maxBackupCount)
	}
	if cfg.LogLevel != LogLevelInfo {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.Timeouts.CommandExecution != 5*time.Second {
		t.Errorf("CommandExecution = %v, want 5s", cfg.Timeouts.CommandExecution)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
}

func TestParseFlags_CustomValues(t *testing.T) {
	withArgs(t, []string{
		"usb-soundcard-mapper",
		"--rules-path", "/tmp/custom-rules",
		"--non-interactive",
		"--vendor-id", "1234",
		"--product-id", "5678",
		"--name", "mydev",
		"--skip-reload",
		"--dry-run",
		"--force",
		"--ignore-virtual",
		"--max-backups", "5",
		"--log-level", "debug",
		"--command-timeout", "9",
		"--lock-timeout", "4",
		"--graceful-timeout", "6",
		"--retries", "7",
	})

	cfg := baseConfig()
	parseFlags(&cfg)

	assertions := []struct {
		name string
		cond bool
	}{
		{"rules-path", cfg.UdevRulesPath == "/tmp/custom-rules"},
		{"non-interactive", cfg.NonInteractive},
		{"vendor-id", cfg.VendorID == "1234"},
		{"product-id", cfg.ProductID == "5678"},
		{"name", cfg.DeviceName == "mydev"},
		{"skip-reload", cfg.SkipReload},
		{"dry-run", cfg.DryRun},
		{"force", cfg.ForceOverwrite},
		{"ignore-virtual", cfg.IgnoreVirtual},
		{"max-backups", cfg.MaxBackupCount == 5},
		{"log-level", cfg.LogLevel == LogLevelDebug},
		{"command-timeout", cfg.Timeouts.CommandExecution == 9*time.Second},
		{"lock-timeout", cfg.Timeouts.LockAcquisition == 4*time.Second},
		{"graceful-timeout", cfg.Timeouts.GracefulShutdown == 6*time.Second},
		{"retries", cfg.MaxRetries == 7},
	}
	for _, a := range assertions {
		if !a.cond {
			t.Errorf("flag %q was not parsed into config correctly", a.name)
		}
	}
}

func TestParseFlags_ListOnly(t *testing.T) {
	withArgs(t, []string{"usb-soundcard-mapper", "--list"})

	cfg := baseConfig()
	parseFlags(&cfg)

	if !cfg.ListOnly {
		t.Error("expected ListOnly to be true")
	}
}
