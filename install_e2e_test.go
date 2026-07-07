// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

// End-to-end tests for the udev rule installation, modprobe, reload and
// verification pipeline. All filesystem side effects are redirected to temp
// dirs and all commands to fakes, so nothing touches the real system.

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newFileAccess() *SafeFileAccess {
	return NewSafeFileAccess(NewResourceTracker())
}

func TestInstallUdevRule_WritesFile(t *testing.T) {
	cfg := testConfig(t)
	cfg.BackupRules = false
	fakeModprobeDir(t, false) // skip modprobe branch

	card := sampleUSBCard()
	rule, err := createUdevRule(context.Background(), &card, "my_audio", cfg)
	if err != nil {
		t.Fatalf("createUdevRule: %v", err)
	}

	if err := installUdevRule(context.Background(), rule, cfg, newFileAccess()); err != nil {
		t.Fatalf("installUdevRule: %v", err)
	}

	got, err := os.ReadFile(rule.Path)
	if err != nil {
		t.Fatalf("rule file not written: %v", err)
	}
	if string(got) != rule.Content {
		t.Error("written rule content does not match generated content")
	}
	if !strings.Contains(string(got), `ATTR{id}="my_audio"`) {
		t.Error("rule file missing device name")
	}
}

func TestInstallUdevRule_DryRunWritesNothing(t *testing.T) {
	cfg := testConfig(t)
	cfg.DryRun = true

	card := sampleUSBCard()
	rule, err := createUdevRule(context.Background(), &card, "my_audio", cfg)
	if err != nil {
		t.Fatalf("createUdevRule: %v", err)
	}

	if err := installUdevRule(context.Background(), rule, cfg, newFileAccess()); err != nil {
		t.Fatalf("installUdevRule: %v", err)
	}
	if _, err := os.Stat(rule.Path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("dry-run must not write a rule file, stat err = %v", err)
	}
}

func TestInstallUdevRule_CreatesModprobeConfig(t *testing.T) {
	cfg := testConfig(t)
	cfg.BackupRules = false
	mpDir := fakeModprobeDir(t, true)

	card := sampleUSBCard()
	rule, err := createUdevRule(context.Background(), &card, "my_audio", cfg)
	if err != nil {
		t.Fatalf("createUdevRule: %v", err)
	}
	if err := installUdevRule(context.Background(), rule, cfg, newFileAccess()); err != nil {
		t.Fatalf("installUdevRule: %v", err)
	}

	conf := filepath.Join(mpDir, "99-soundcard-1234-5678.conf")
	data, err := os.ReadFile(conf)
	if err != nil {
		t.Fatalf("modprobe config not written: %v", err)
	}
	if !strings.Contains(string(data), "options snd_usb_audio index=-2") {
		t.Errorf("unexpected modprobe content: %s", data)
	}
}

func TestInstallUdevRule_BackupOnOverwrite(t *testing.T) {
	cfg := testConfig(t)
	cfg.BackupRules = true
	cfg.ForceOverwrite = true
	fakeModprobeDir(t, false)

	card := sampleUSBCard()
	rule, err := createUdevRule(context.Background(), &card, "my_audio", cfg)
	if err != nil {
		t.Fatalf("createUdevRule: %v", err)
	}

	// Pre-existing rule file with different content.
	if err := os.WriteFile(rule.Path, []byte("# old content\n"), 0o644); err != nil {
		t.Fatalf("seed existing rule: %v", err)
	}

	if err := installUdevRule(context.Background(), rule, cfg, newFileAccess()); err != nil {
		t.Fatalf("installUdevRule: %v", err)
	}

	got, _ := os.ReadFile(rule.Path)
	if string(got) != rule.Content {
		t.Error("rule file was not overwritten with new content")
	}

	// A timestamped backup of the old content should exist next to it.
	backups, _ := filepath.Glob(rule.Path + ".bak.*")
	if len(backups) == 0 {
		t.Fatal("expected a .bak backup of the previous rule file")
	}
	old, _ := os.ReadFile(backups[0])
	if string(old) != "# old content\n" {
		t.Errorf("backup does not contain previous content, got %q", old)
	}
}

func TestInstallUdevRule_SkipsIdenticalContent(t *testing.T) {
	cfg := testConfig(t)
	cfg.BackupRules = false
	fakeModprobeDir(t, false)

	card := sampleUSBCard()
	rule, err := createUdevRule(context.Background(), &card, "my_audio", cfg)
	if err != nil {
		t.Fatalf("createUdevRule: %v", err)
	}

	// Seed the target with identical content; install must be a no-op write.
	if err := os.WriteFile(rule.Path, []byte(rule.Content), 0o644); err != nil {
		t.Fatalf("seed rule: %v", err)
	}
	info1, _ := os.Stat(rule.Path)

	if err := installUdevRule(context.Background(), rule, cfg, newFileAccess()); err != nil {
		t.Fatalf("installUdevRule: %v", err)
	}
	got, _ := os.ReadFile(rule.Path)
	if string(got) != rule.Content {
		t.Error("content changed unexpectedly")
	}
	if info1 == nil {
		t.Fatal("expected to stat seeded file")
	}
}

func TestCreateModprobeConfig_SkipsWhenDirMissing(t *testing.T) {
	cfg := testConfig(t)
	fakeModprobeDir(t, false)

	card := sampleUSBCard()
	if err := createModprobeConfig(&card, cfg, newFileAccess()); err != nil {
		t.Fatalf("expected nil when modprobe dir is absent, got %v", err)
	}
}

func TestCreateModprobeConfig_DryRunSkips(t *testing.T) {
	cfg := testConfig(t)
	cfg.DryRun = true
	mpDir := fakeModprobeDir(t, true)

	card := sampleUSBCard()
	if err := createModprobeConfig(&card, cfg, newFileAccess()); err != nil {
		t.Fatalf("createModprobeConfig: %v", err)
	}
	if entries, _ := os.ReadDir(mpDir); len(entries) != 0 {
		t.Errorf("dry-run must not write modprobe files, found %d", len(entries))
	}
}

func TestReloadUdevRules_Success(t *testing.T) {
	installFakeBin(t)
	cfg := testConfig(t)

	if err := reloadUdevRules(context.Background(), newTestExecutor(), cfg); err != nil {
		t.Fatalf("reloadUdevRules: %v", err)
	}
}

func TestReloadUdevRules_DryRunSkips(t *testing.T) {
	cfg := testConfig(t)
	cfg.DryRun = true
	// No fakes installed; a dry run must not exec anything.
	if err := reloadUdevRules(context.Background(), newTestExecutor(), cfg); err != nil {
		t.Fatalf("reloadUdevRules dry-run: %v", err)
	}
}

func TestReloadUdevRules_ControlFailure(t *testing.T) {
	scenario := installFakeBin(t)
	scenarioFile(t, scenario, "udevadm_control_rc", "1\n")
	cfg := testConfig(t)

	err := reloadUdevRules(context.Background(), newTestExecutor(), cfg)
	if err == nil {
		t.Fatal("expected error when udevadm control fails")
	}
}

func TestVerifyUdevRuleInstallation_DryRun(t *testing.T) {
	cfg := testConfig(t)
	cfg.DryRun = true
	card := sampleUSBCard()

	ok, err := verifyUdevRuleInstallation(context.Background(), newTestExecutor(), &card, "my_audio", cfg)
	if err != nil || !ok {
		t.Fatalf("dry-run verify should return (true, nil), got (%v, %v)", ok, err)
	}
}

func TestVerifyUdevRuleInstallation_DeviceDisconnected(t *testing.T) {
	installFakeBin(t)
	fakeSysfs(t) // no card1
	cfg := testConfig(t)
	card := sampleUSBCard()

	ok, err := verifyUdevRuleInstallation(context.Background(), newTestExecutor(), &card, "my_audio", cfg)
	if ok || !errors.Is(err, ErrDeviceDisconnected) {
		t.Fatalf("expected (false, ErrDeviceDisconnected), got (%v, %v)", ok, err)
	}
}

func TestVerifyUdevRuleInstallation_SuccessViaUdevadmInfo(t *testing.T) {
	scenario := installFakeBin(t)
	fakeSysfs(t, "1")
	scenarioFile(t, scenario, "udevadm_info.txt", "E: ID_SOUND_ID=my_audio\n")
	cfg := testConfig(t)
	card := sampleUSBCard()

	ok, err := verifyUdevRuleInstallation(context.Background(), newTestExecutor(), &card, "my_audio", cfg)
	if err != nil || !ok {
		t.Fatalf("expected successful verification, got (%v, %v)", ok, err)
	}
}

func TestVerifyUdevRuleInstallation_AllMethodsFail(t *testing.T) {
	installFakeBin(t)
	fakeSysfs(t, "1")
	cfg := testConfig(t)
	card := sampleUSBCard()

	ok, err := verifyUdevRuleInstallation(context.Background(), newTestExecutor(), &card, "name_not_present", cfg)
	if ok || !errors.Is(err, ErrRuleVerificationFail) {
		t.Fatalf("expected (false, ErrRuleVerificationFail), got (%v, %v)", ok, err)
	}
}
