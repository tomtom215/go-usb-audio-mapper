// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeCommands are the system binaries the tool shells out to, faked under
// testdata/fakebin for hardware-free end-to-end tests.
var fakeCommands = []string{"aplay", "udevadm", "lsusb", "pipewire", "pulseaudio", "jackd"}

// installFakeBin copies the fake USB/audio commands into a fresh temp dir,
// prepends that dir to PATH for the duration of the test, and returns a
// scenario directory (exported as FAKE_DEV_DIR) into which a test may drop
// override files to steer the fakes. Using t.Setenv marks the test as
// non-parallel, which is required because it also mutates process-global PATH
// and the sysClassSoundPath/modprobeDir package variables.
func installFakeBin(t *testing.T) string {
	t.Helper()

	srcDir, err := filepath.Abs(filepath.Join("testdata", "fakebin"))
	if err != nil {
		t.Fatalf("resolve fakebin dir: %v", err)
	}

	binDir := t.TempDir()
	for _, name := range fakeCommands {
		data, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			t.Fatalf("read fake %q: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(binDir, name), data, 0o755); err != nil { //nolint:gosec // fixtures must be executable
			t.Fatalf("write fake %q: %v", name, err)
		}
	}

	scenarioDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_DEV_DIR", scenarioDir)
	return scenarioDir
}

// scenarioFile drops an override file into the active FAKE_DEV_DIR so a test can
// customize what the fake commands emit (e.g. a virtual-device attribute walk).
func scenarioFile(t *testing.T, scenarioDir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(scenarioDir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write scenario file %q: %v", name, err)
	}
}

// fakeSysfs points sysClassSoundPath at a temp dir populated with the requested
// card directories (fakeSysfs(t, "1") creates card1/), restoring the original
// value on cleanup so the detection code believes those cards are present.
func fakeSysfs(t *testing.T, cards ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, c := range cards {
		if err := os.MkdirAll(filepath.Join(root, "card"+c), 0o755); err != nil {
			t.Fatalf("create fake sysfs card%s: %v", c, err)
		}
	}
	orig := sysClassSoundPath
	sysClassSoundPath = root
	t.Cleanup(func() { sysClassSoundPath = orig })
	return root
}

// fakeModprobeDir points modprobeDir at an existing temp dir, restoring the
// original on cleanup. Pass exists=false to point it at a path that does not
// exist (so createModprobeConfig takes its "directory missing" branch).
func fakeModprobeDir(t *testing.T, exists bool) string {
	t.Helper()
	dir := t.TempDir()
	if !exists {
		dir = filepath.Join(dir, "absent")
	}
	orig := modprobeDir
	modprobeDir = dir
	t.Cleanup(func() { modprobeDir = orig })
	return dir
}

// testConfig returns a Config with a fresh temp rules directory and short
// timeouts so reload/verify tests do not sleep for real seconds.
func testConfig(t *testing.T) *Config {
	t.Helper()
	return &Config{
		UdevRulesPath:  t.TempDir(),
		BackupRules:    true,
		MaxBackupCount: 10,
		Timeouts: ConfigurableTimeouts{
			CommandExecution:  5 * time.Second,
			RuleReloadWait:    time.Millisecond,
			TriggerActionWait: time.Millisecond,
			RetryInterval:     time.Millisecond,
			GracefulShutdown:  time.Second,
			LockAcquisition:   2 * time.Second,
		},
	}
}

// sampleUSBCard returns a fully-populated non-virtual card matching the default
// fake-command scenario.
func sampleUSBCard() USBSoundCard {
	return USBSoundCard{
		CardNumber:   "1",
		DevicePath:   "/dev/snd/card1",
		Vendor:       "Focusrite-Novation",
		Product:      "Scarlett 2i2",
		VendorID:     "1234",
		ProductID:    "5678",
		Serial:       "SN123456ABC",
		PhysicalPort: "1-2.3",
		FriendlyName: "usb_1234_5678_SN123456ABC",
		Status:       DeviceStatusConnected,
	}
}
