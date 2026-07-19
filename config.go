// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"fmt"
	"log/slog"
	"regexp"
	"time"
)

// Application constants
const (
	AppName    = "usb-soundcard-mapper"
	AppVersion = "2.1.0"

	udevRulesDir   = "/etc/udev/rules.d"
	maxBackupCount = 10          // Maximum number of backups to keep per device
	maxQueueSize   = 100         // Maximum size of operation queues
	maxFileSize    = 1024 * 1024 // 1MB maximum file size for safety
)

// System integration paths. These are variables rather than constants so that
// tests can redirect them to temporary directories and exercise the detection
// and verification pipeline without real hardware. At runtime they always hold
// their default values.
var (
	sysClassSoundPath = "/sys/class/sound"
	modprobeDir       = "/etc/modprobe.d"
)

// Pre-compiled regular expressions for improved performance and safety
var (
	vendorIDRegex    = regexp.MustCompile(`^[0-9a-fA-F]{4}$`)
	productIDRegex   = regexp.MustCompile(`^[0-9a-fA-F]{4}$`)
	serialRegex      = regexp.MustCompile(`^[^<>|&;()$\r\n\t\x00]+$`)
	unsafeCharsRegex = regexp.MustCompile(`[<>|&;()$\r\n\t\x00]`)
	fileNameRegex    = regexp.MustCompile(`^[a-zA-Z0-9_\-.]+$`)
	pathSafeRegex    = regexp.MustCompile(`^[a-zA-Z0-9_\-./]+$`)
	nonAlphaNumRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	cardRegex        = regexp.MustCompile(`card (\d+):.*\[(.+)\].*\[(.+)\]`)

	// udevUnsafeValueRegex matches any character that cannot appear inside a
	// double-quoted udev rule match value without corrupting the rule: a double
	// quote closes the string, a backslash may be treated as an escape, and any
	// control character (including CR/LF) breaks udev's single-line grammar.
	// Values that match are considered unrepresentable and the caller falls back
	// to a different, always-safe match key (e.g. the physical USB port).
	udevUnsafeValueRegex = regexp.MustCompile(`["\\\x00-\x1f\x7f]`)
)

// isUdevSafeValue reports whether s can be embedded verbatim inside a
// double-quoted udev rule match without producing a malformed rule.
func isUdevSafeValue(s string) bool {
	return !udevUnsafeValueRegex.MatchString(s)
}

// LogLevel represents the logging verbosity level
type LogLevel string

// Log level constants
const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// ConfigurableTimeouts holds configurable timeout values
type ConfigurableTimeouts struct {
	CommandExecution  time.Duration
	RuleReloadWait    time.Duration
	TriggerActionWait time.Duration
	RetryInterval     time.Duration
	GracefulShutdown  time.Duration
	LockAcquisition   time.Duration
}

// DefaultTimeouts provides default values for timeouts
var DefaultTimeouts = ConfigurableTimeouts{
	CommandExecution:  5 * time.Second,
	RuleReloadWait:    1 * time.Second,
	TriggerActionWait: 2 * time.Second,
	RetryInterval:     500 * time.Millisecond,
	GracefulShutdown:  5 * time.Second,
	LockAcquisition:   2 * time.Second,
}

// Config holds application configuration
type Config struct {
	UdevRulesPath   string
	ListOnly        bool
	NonInteractive  bool
	DeviceName      string
	VendorID        string
	ProductID       string
	LogLevel        LogLevel
	SkipReload      bool
	DryRun          bool
	ConcurrencyOpts ConcurrencyOptions
	BackupRules     bool
	Timeouts        ConfigurableTimeouts
	MaxRetries      int
	ForceOverwrite  bool
	IgnoreVirtual   bool
	MaxBackupCount  int
	ResourceLimits  ResourceLimits
}

// ResourceLimits defines limits to prevent resource exhaustion
type ResourceLimits struct {
	MaxConcurrentOps    int
	MaxQueueSize        int
	MaxFileSize         int64
	MaxBackupsPerDevice int
}

// ConcurrencyOptions configures the concurrency behavior
type ConcurrencyOptions struct {
	MaxWorkers     int
	OperationQueue int
}

// validateConfig validates the configuration for consistency and safety
func validateConfig(config *Config) error {
	// Validate UdevRulesPath
	if !pathSafeRegex.MatchString(config.UdevRulesPath) {
		return fmt.Errorf("unsafe udev rules path: %s: %w", config.UdevRulesPath, ErrInvalidPath)
	}

	// Make sure MaxBackupCount is reasonable
	if config.MaxBackupCount <= 0 {
		config.MaxBackupCount = maxBackupCount
	}

	// Set resource limits if not specified
	if config.ResourceLimits.MaxConcurrentOps <= 0 {
		config.ResourceLimits.MaxConcurrentOps = config.ConcurrencyOpts.MaxWorkers
	}

	if config.ResourceLimits.MaxQueueSize <= 0 {
		config.ResourceLimits.MaxQueueSize = maxQueueSize
	}

	if config.ResourceLimits.MaxFileSize <= 0 {
		config.ResourceLimits.MaxFileSize = maxFileSize
	}

	// Check for conflicting options
	if config.DryRun && config.ForceOverwrite {
		slog.Warn("Both --dry-run and --force specified; dry run takes precedence")
	}

	// If vendor ID or product ID is specified, validate their format
	if config.VendorID != "" {
		if !vendorIDRegex.MatchString(config.VendorID) {
			return fmt.Errorf("invalid vendor ID format: %s: %w", config.VendorID, ErrInvalidDeviceParams)
		}
	}

	if config.ProductID != "" {
		if !productIDRegex.MatchString(config.ProductID) {
			return fmt.Errorf("invalid product ID format: %s: %w", config.ProductID, ErrInvalidDeviceParams)
		}
	}

	// Ensure LockAcquisition timeout is set
	if config.Timeouts.LockAcquisition <= 0 {
		config.Timeouts.LockAcquisition = DefaultTimeouts.LockAcquisition
	}

	// Ensure GracefulShutdown timeout is set
	if config.Timeouts.GracefulShutdown <= 0 {
		config.Timeouts.GracefulShutdown = DefaultTimeouts.GracefulShutdown
	}

	// A non-positive command-execution timeout makes every exec context expire
	// immediately, which silently breaks all detection and installation. Rather
	// than fail an unattended field deployment over a mistyped flag, fall back to
	// the safe default and warn. The remaining wait/retry timeouts are clamped
	// too so a bad value cannot turn a bounded wait into a busy loop or a
	// zero-length sleep.
	if config.Timeouts.CommandExecution <= 0 {
		slog.Warn("Non-positive command-execution timeout; using default",
			"provided", config.Timeouts.CommandExecution,
			"default", DefaultTimeouts.CommandExecution)
		config.Timeouts.CommandExecution = DefaultTimeouts.CommandExecution
	}
	if config.Timeouts.RetryInterval <= 0 {
		config.Timeouts.RetryInterval = DefaultTimeouts.RetryInterval
	}
	if config.Timeouts.RuleReloadWait <= 0 {
		config.Timeouts.RuleReloadWait = DefaultTimeouts.RuleReloadWait
	}
	if config.Timeouts.TriggerActionWait <= 0 {
		config.Timeouts.TriggerActionWait = DefaultTimeouts.TriggerActionWait
	}

	// A negative retry count would skip command execution entirely; clamp it to
	// zero (meaning "one attempt, no retries").
	if config.MaxRetries < 0 {
		slog.Warn("Negative retry count; clamping to zero", "provided", config.MaxRetries)
		config.MaxRetries = 0
	}

	// Normalize the log level. An unrecognized value (typically a typo such as
	// "debgu") would otherwise be silently treated as info and could hide the
	// very diagnostics an operator was trying to enable, so surface it and fall
	// back to info explicitly.
	switch config.LogLevel {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		slog.Warn("Unknown log level; defaulting to info", "provided", string(config.LogLevel))
		config.LogLevel = LogLevelInfo
	}

	return nil
}
