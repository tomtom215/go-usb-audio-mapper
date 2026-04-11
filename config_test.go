package main

import (
	"testing"
	"time"
)

func TestValidateConfig_ValidConfig(t *testing.T) {
	config := &Config{
		UdevRulesPath:  "/etc/udev/rules.d",
		MaxBackupCount: 10,
		Timeouts:       DefaultTimeouts,
		ConcurrencyOpts: ConcurrencyOptions{
			MaxWorkers:     4,
			OperationQueue: 100,
		},
		ResourceLimits: ResourceLimits{
			MaxConcurrentOps:    4,
			MaxQueueSize:        100,
			MaxFileSize:         maxFileSize,
			MaxBackupsPerDevice: 10,
		},
	}

	err := validateConfig(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateConfig_UnsafePath(t *testing.T) {
	config := &Config{
		UdevRulesPath:  "/etc/../../../tmp/evil path",
		MaxBackupCount: 10,
		Timeouts:       DefaultTimeouts,
	}

	err := validateConfig(config)
	if err == nil {
		t.Fatal("expected error for unsafe path, got nil")
	}
}

func TestValidateConfig_InvalidVendorID(t *testing.T) {
	config := &Config{
		UdevRulesPath:  "/etc/udev/rules.d",
		VendorID:       "ZZZZ",
		MaxBackupCount: 10,
		Timeouts:       DefaultTimeouts,
	}

	err := validateConfig(config)
	if err == nil {
		t.Fatal("expected error for invalid vendor ID, got nil")
	}
}

func TestValidateConfig_InvalidProductID(t *testing.T) {
	config := &Config{
		UdevRulesPath:  "/etc/udev/rules.d",
		ProductID:      "not-hex",
		MaxBackupCount: 10,
		Timeouts:       DefaultTimeouts,
	}

	err := validateConfig(config)
	if err == nil {
		t.Fatal("expected error for invalid product ID, got nil")
	}
}

func TestValidateConfig_ValidVendorAndProductID(t *testing.T) {
	config := &Config{
		UdevRulesPath:  "/etc/udev/rules.d",
		VendorID:       "1234",
		ProductID:      "abcd",
		MaxBackupCount: 10,
		Timeouts:       DefaultTimeouts,
		ConcurrencyOpts: ConcurrencyOptions{
			MaxWorkers: 4,
		},
	}

	err := validateConfig(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateConfig_SetsDefaults(t *testing.T) {
	config := &Config{
		UdevRulesPath:  "/etc/udev/rules.d",
		MaxBackupCount: 0,
		Timeouts: ConfigurableTimeouts{
			LockAcquisition:  0,
			GracefulShutdown: 0,
		},
		ConcurrencyOpts: ConcurrencyOptions{
			MaxWorkers: 4,
		},
	}

	err := validateConfig(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.MaxBackupCount != maxBackupCount {
		t.Errorf("expected MaxBackupCount to be set to default %d, got %d", maxBackupCount, config.MaxBackupCount)
	}

	if config.Timeouts.LockAcquisition != DefaultTimeouts.LockAcquisition {
		t.Errorf("expected LockAcquisition timeout to be set to default")
	}

	if config.Timeouts.GracefulShutdown != DefaultTimeouts.GracefulShutdown {
		t.Errorf("expected GracefulShutdown timeout to be set to default")
	}
}

func TestValidateConfig_DryRunAndForceWarning(t *testing.T) {
	config := &Config{
		UdevRulesPath:  "/etc/udev/rules.d",
		DryRun:         true,
		ForceOverwrite: true,
		MaxBackupCount: 10,
		Timeouts:       DefaultTimeouts,
		ConcurrencyOpts: ConcurrencyOptions{
			MaxWorkers: 4,
		},
	}

	// Should not error, just warn
	err := validateConfig(config)
	if err != nil {
		t.Fatalf("expected no error for dry-run+force combo, got %v", err)
	}
}

func TestDefaultTimeouts_Values(t *testing.T) {
	if DefaultTimeouts.CommandExecution != 5*time.Second {
		t.Errorf("expected CommandExecution 5s, got %v", DefaultTimeouts.CommandExecution)
	}
	if DefaultTimeouts.RuleReloadWait != 1*time.Second {
		t.Errorf("expected RuleReloadWait 1s, got %v", DefaultTimeouts.RuleReloadWait)
	}
	if DefaultTimeouts.RetryInterval != 500*time.Millisecond {
		t.Errorf("expected RetryInterval 500ms, got %v", DefaultTimeouts.RetryInterval)
	}
}

func TestConstants(t *testing.T) {
	if AppName != "usb-soundcard-mapper" {
		t.Errorf("unexpected AppName: %s", AppName)
	}
	if AppVersion != "2.1.0" {
		t.Errorf("unexpected AppVersion: %s", AppVersion)
	}
	if maxBackupCount != 10 {
		t.Errorf("unexpected maxBackupCount: %d", maxBackupCount)
	}
	if maxQueueSize != 100 {
		t.Errorf("unexpected maxQueueSize: %d", maxQueueSize)
	}
	if maxFileSize != 1024*1024 {
		t.Errorf("unexpected maxFileSize: %d", maxFileSize)
	}
}
