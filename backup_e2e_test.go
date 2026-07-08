// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

// End-to-end tests for rule backup and the udev-system self test.

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func countBackups(t *testing.T, cfg *Config) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(cfg.UdevRulesPath, "backups"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("read backups dir: %v", err)
	}
	n := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "backup_1234_5678_") {
			n++
		}
	}
	return n
}

func TestBackupExistingUdevRules_CopiesMatching(t *testing.T) {
	cfg := testConfig(t)
	card := sampleUSBCard()

	// A pre-existing third-party rule that matches the search patterns.
	existing := filepath.Join(cfg.UdevRulesPath, "10-my-usb-soundcard-1234-5678.rules")
	if err := os.WriteFile(existing, []byte("# existing rule\n"), 0o644); err != nil {
		t.Fatalf("seed existing rule: %v", err)
	}

	if err := backupExistingUdevRules(&card, cfg, newFileAccess()); err != nil {
		t.Fatalf("backupExistingUdevRules: %v", err)
	}
	if got := countBackups(t, cfg); got == 0 {
		t.Fatal("expected at least one backup to be created")
	}
}

func TestBackupExistingUdevRules_DisabledSkips(t *testing.T) {
	cfg := testConfig(t)
	cfg.BackupRules = false
	card := sampleUSBCard()

	existing := filepath.Join(cfg.UdevRulesPath, "10-my-usb-soundcard-1234-5678.rules")
	if err := os.WriteFile(existing, []byte("# existing rule\n"), 0o644); err != nil {
		t.Fatalf("seed existing rule: %v", err)
	}

	if err := backupExistingUdevRules(&card, cfg, newFileAccess()); err != nil {
		t.Fatalf("backupExistingUdevRules: %v", err)
	}
	if got := countBackups(t, cfg); got != 0 {
		t.Errorf("backups disabled but %d backups created", got)
	}
}

func TestBackupExistingUdevRules_SkipsOwnRule(t *testing.T) {
	cfg := testConfig(t)
	card := sampleUSBCard()

	// The tool's own rule file must not be backed up as a foreign rule.
	own := filepath.Join(cfg.UdevRulesPath, "89-usb-soundcard-1234-5678.rules")
	if err := os.WriteFile(own, []byte("# our own rule\n"), 0o644); err != nil {
		t.Fatalf("seed own rule: %v", err)
	}

	if err := backupExistingUdevRules(&card, cfg, newFileAccess()); err != nil {
		t.Fatalf("backupExistingUdevRules: %v", err)
	}
	if got := countBackups(t, cfg); got != 0 {
		t.Errorf("own rule should not be backed up, but %d backups created", got)
	}
}

func TestTestUdevSystem_Success(t *testing.T) {
	installFakeBin(t)
	cfg := testConfig(t)

	ok, err := testUdevSystem(context.Background(), newTestExecutor(), cfg, newFileAccess())
	if err != nil || !ok {
		t.Fatalf("expected (true, nil), got (%v, %v)", ok, err)
	}
	// The temporary test rule must be cleaned up.
	if _, err := os.Stat(filepath.Join(cfg.UdevRulesPath, "99-test-usb-soundcard-mapper.rules")); !os.IsNotExist(err) {
		t.Errorf("test rule file was not removed, stat err = %v", err)
	}
}

func TestTestUdevSystem_DryRun(t *testing.T) {
	cfg := testConfig(t)
	cfg.DryRun = true

	ok, err := testUdevSystem(context.Background(), newTestExecutor(), cfg, newFileAccess())
	if err != nil || !ok {
		t.Fatalf("dry-run should return (true, nil), got (%v, %v)", ok, err)
	}
}
