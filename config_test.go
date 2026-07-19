// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

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

func TestValidateConfig_ClampsNonPositiveTimeouts(t *testing.T) {
	config := &Config{
		UdevRulesPath:  "/etc/udev/rules.d",
		MaxBackupCount: 10,
		LogLevel:       LogLevelInfo,
		Timeouts: ConfigurableTimeouts{
			CommandExecution:  0,
			RetryInterval:     -1 * time.Second,
			RuleReloadWait:    0,
			TriggerActionWait: 0,
			LockAcquisition:   0,
			GracefulShutdown:  0,
		},
		ConcurrencyOpts: ConcurrencyOptions{MaxWorkers: 4},
	}

	if err := validateConfig(config); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config.Timeouts.CommandExecution != DefaultTimeouts.CommandExecution {
		t.Errorf("CommandExecution = %v, want default %v",
			config.Timeouts.CommandExecution, DefaultTimeouts.CommandExecution)
	}
	if config.Timeouts.RetryInterval != DefaultTimeouts.RetryInterval {
		t.Errorf("RetryInterval = %v, want default %v",
			config.Timeouts.RetryInterval, DefaultTimeouts.RetryInterval)
	}
	if config.Timeouts.RuleReloadWait != DefaultTimeouts.RuleReloadWait {
		t.Errorf("RuleReloadWait = %v, want default", config.Timeouts.RuleReloadWait)
	}
	if config.Timeouts.TriggerActionWait != DefaultTimeouts.TriggerActionWait {
		t.Errorf("TriggerActionWait = %v, want default", config.Timeouts.TriggerActionWait)
	}
}

func TestValidateConfig_ClampsNegativeRetries(t *testing.T) {
	config := &Config{
		UdevRulesPath:  "/etc/udev/rules.d",
		MaxBackupCount: 10,
		LogLevel:       LogLevelInfo,
		Timeouts:       DefaultTimeouts,
		MaxRetries:     -5,
		ConcurrencyOpts: ConcurrencyOptions{
			MaxWorkers: 4,
		},
	}

	if err := validateConfig(config); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if config.MaxRetries != 0 {
		t.Errorf("MaxRetries = %d, want 0 after clamping", config.MaxRetries)
	}
}

func TestValidateConfig_NormalizesUnknownLogLevel(t *testing.T) {
	config := &Config{
		UdevRulesPath:   "/etc/udev/rules.d",
		MaxBackupCount:  10,
		LogLevel:        LogLevel("debgu"), // typo
		Timeouts:        DefaultTimeouts,
		ConcurrencyOpts: ConcurrencyOptions{MaxWorkers: 4},
	}

	if err := validateConfig(config); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if config.LogLevel != LogLevelInfo {
		t.Errorf("LogLevel = %q, want normalized to info", config.LogLevel)
	}
}

func TestValidateConfig_PreservesValidLogLevels(t *testing.T) {
	for _, level := range []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError} {
		config := &Config{
			UdevRulesPath:   "/etc/udev/rules.d",
			MaxBackupCount:  10,
			LogLevel:        level,
			Timeouts:        DefaultTimeouts,
			ConcurrencyOpts: ConcurrencyOptions{MaxWorkers: 4},
		}
		if err := validateConfig(config); err != nil {
			t.Fatalf("level %q: expected no error, got %v", level, err)
		}
		if config.LogLevel != level {
			t.Errorf("valid level %q was altered to %q", level, config.LogLevel)
		}
	}
}

func TestIsUdevSafeValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		safe  bool
	}{
		{"alphanumeric", "SN123456ABC", true},
		{"with dashes", "SER-001-USB", true},
		{"with dots", "1.2.3", true},
		{"empty", "", true},
		{"double quote", `SN"123`, false},
		{"backslash", `SN\123`, false},
		{"trailing backslash", `SN123\`, false},
		{"newline", "SN\n123", false},
		{"carriage return", "SN\r123", false},
		{"tab", "SN\t123", false},
		{"null byte", "SN\x00123", false},
		{"del char", "SN\x7f123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUdevSafeValue(tt.input); got != tt.safe {
				t.Errorf("isUdevSafeValue(%q) = %v, want %v", tt.input, got, tt.safe)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if AppName != "usb-soundcard-mapper" {
		t.Errorf("unexpected AppName: %s", AppName)
	}
	if AppVersion != "2.2.0" {
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
