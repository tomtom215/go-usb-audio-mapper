// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

// End-to-end tests for the installation orchestration (performInstallation) and
// the non-interactive CLI flow (nonInteractiveMode).

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rulePath(cfg *Config, vid, pid string) string {
	return filepath.Join(cfg.UdevRulesPath, "89-usb-soundcard-"+vid+"-"+pid+".rules")
}

func TestPerformInstallation_FullFlow(t *testing.T) {
	scenario := installFakeBin(t)
	fakeSysfs(t, "1")
	fakeModprobeDir(t, true)
	scenarioFile(t, scenario, "udevadm_info.txt", "E: ID_SOUND_ID=my_audio\n")

	cfg := testConfig(t)
	cfg.BackupRules = false
	card := sampleUSBCard()

	msg, err := performInstallation(context.Background(), &card, "my_audio", cfg, newTestExecutor(), newFileAccess())
	if err != nil {
		t.Fatalf("performInstallation: %v", err)
	}
	if !strings.Contains(msg, "my_audio") {
		t.Errorf("message should mention the device name: %q", msg)
	}
	if _, err := os.Stat(rulePath(cfg, "1234", "5678")); err != nil {
		t.Errorf("rule file not created: %v", err)
	}
}

func TestPerformInstallation_SkipReload(t *testing.T) {
	installFakeBin(t)
	fakeModprobeDir(t, false)

	cfg := testConfig(t)
	cfg.BackupRules = false
	cfg.SkipReload = true
	card := sampleUSBCard()

	msg, err := performInstallation(context.Background(), &card, "my_audio", cfg, newTestExecutor(), newFileAccess())
	if err != nil {
		t.Fatalf("performInstallation: %v", err)
	}
	if strings.Contains(msg, "Warning") {
		t.Errorf("skip-reload run should not warn about verification: %q", msg)
	}
	if _, err := os.Stat(rulePath(cfg, "1234", "5678")); err != nil {
		t.Errorf("rule file not created: %v", err)
	}
}

func TestPerformInstallation_VerificationWarning(t *testing.T) {
	installFakeBin(t)
	fakeSysfs(t, "1")
	fakeModprobeDir(t, false)

	cfg := testConfig(t)
	cfg.BackupRules = false
	card := sampleUSBCard()

	// No ID_SOUND_ID override -> all verification methods fail -> a warning is
	// appended but installation still succeeds.
	msg, err := performInstallation(context.Background(), &card, "unverifiable_name", cfg, newTestExecutor(), newFileAccess())
	if err != nil {
		t.Fatalf("performInstallation should not fail on verification issues: %v", err)
	}
	if !strings.Contains(msg, "Warning") {
		t.Errorf("expected a verification warning in message: %q", msg)
	}
}

func TestNonInteractiveMode_Success(t *testing.T) {
	installFakeBin(t)
	fakeSysfs(t, "1")
	fakeModprobeDir(t, true)

	cfg := testConfig(t)
	cfg.NonInteractive = true
	cfg.VendorID = "1234"
	cfg.ProductID = "5678"
	cfg.DeviceName = "my_audio"

	err := nonInteractiveMode(context.Background(), cfg, newTestExecutor(), newFileAccess(), []USBSoundCard{sampleUSBCard()})
	if err != nil {
		t.Fatalf("nonInteractiveMode: %v", err)
	}
	if _, err := os.Stat(rulePath(cfg, "1234", "5678")); err != nil {
		t.Errorf("rule file not created: %v", err)
	}
}

func TestNonInteractiveMode_MissingIDs(t *testing.T) {
	cfg := testConfig(t)
	cfg.NonInteractive = true
	cfg.VendorID = "1234" // product ID missing

	err := nonInteractiveMode(context.Background(), cfg, newTestExecutor(), newFileAccess(), []USBSoundCard{sampleUSBCard()})
	if !errors.Is(err, ErrInvalidDeviceParams) {
		t.Fatalf("expected ErrInvalidDeviceParams, got %v", err)
	}
}

func TestNonInteractiveMode_CardNotFound(t *testing.T) {
	cfg := testConfig(t)
	cfg.NonInteractive = true
	cfg.VendorID = "9999"
	cfg.ProductID = "8888"

	err := nonInteractiveMode(context.Background(), cfg, newTestExecutor(), newFileAccess(), []USBSoundCard{sampleUSBCard()})
	if !errors.Is(err, ErrNoUSBSoundCards) {
		t.Fatalf("expected ErrNoUSBSoundCards, got %v", err)
	}
}

func TestNonInteractiveMode_VirtualRejected(t *testing.T) {
	cfg := testConfig(t)
	cfg.NonInteractive = true
	cfg.VendorID = "1234"
	cfg.ProductID = "5678"

	virtual := sampleUSBCard()
	virtual.IsVirtual = true

	err := nonInteractiveMode(context.Background(), cfg, newTestExecutor(), newFileAccess(), []USBSoundCard{virtual})
	if !errors.Is(err, ErrVirtualDevice) {
		t.Fatalf("expected ErrVirtualDevice, got %v", err)
	}
}

func TestNonInteractiveMode_DryRun(t *testing.T) {
	installFakeBin(t)
	fakeModprobeDir(t, false)

	cfg := testConfig(t)
	cfg.NonInteractive = true
	cfg.DryRun = true
	cfg.VendorID = "1234"
	cfg.ProductID = "5678"

	err := nonInteractiveMode(context.Background(), cfg, newTestExecutor(), newFileAccess(), []USBSoundCard{sampleUSBCard()})
	if err != nil {
		t.Fatalf("nonInteractiveMode dry-run: %v", err)
	}
	if _, err := os.Stat(rulePath(cfg, "1234", "5678")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("dry-run must not create a rule file, stat err = %v", err)
	}
}
