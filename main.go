// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

// USB Soundcard Mapper
// A robust utility for creating persistent udev mappings for USB audio devices
// Version: 2.1.0 (production readiness release)

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resourceTracker := NewResourceTracker()
	fileAccess := NewSafeFileAccess(resourceTracker)

	config := Config{
		UdevRulesPath:  udevRulesDir,
		LogLevel:       LogLevelInfo,
		BackupRules:    true,
		Timeouts:       DefaultTimeouts,
		MaxRetries:     3,
		ForceOverwrite: false,
		IgnoreVirtual:  false,
		MaxBackupCount: maxBackupCount,
		ConcurrencyOpts: ConcurrencyOptions{
			MaxWorkers:     4,
			OperationQueue: maxQueueSize,
		},
		ResourceLimits: ResourceLimits{
			MaxConcurrentOps:    4,
			MaxQueueSize:        maxQueueSize,
			MaxFileSize:         maxFileSize,
			MaxBackupsPerDevice: maxBackupCount,
		},
	}

	parseFlags(&config)

	if err := validateConfig(&config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	initLogger(config.LogLevel)

	slog.Info(fmt.Sprintf("Starting %s v%s", AppName, AppVersion),
		"rules_path", config.UdevRulesPath,
		"interactive", !config.NonInteractive,
		"dry_run", config.DryRun,
		"force", config.ForceOverwrite,
		"ignore_virtual", config.IgnoreVirtual)

	setupSignalHandling(ctx, cancel, resourceTracker, &config)

	executor := NewCommandExecutor(&config, resourceTracker)

	if err := CheckCommands(); err != nil {
		slog.Error("Command check failed", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	elevated, err := checkElevatedPrivileges()
	if err != nil {
		slog.Error("Failed to check privileges", "error", err)
		fmt.Fprintf(os.Stderr, "Error checking privileges: %v\n", err)
		return 1
	}

	if !elevated && !config.ListOnly && !config.DryRun {
		slog.Error("Insufficient privileges", "error", ErrInsufficientPrivs)
		fmt.Fprintf(os.Stderr, "This application requires root privileges to create udev rules.\nPlease run with sudo.\n")
		return 1
	}

	if !config.ListOnly && !config.DryRun {
		if err := checkAndFixPermissions(&config); err != nil {
			slog.Error("Permission check failed", "error", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}

	if !config.ListOnly && !config.DryRun {
		success, err := testUdevSystem(ctx, executor, &config, fileAccess)
		if err != nil {
			slog.Error("Udev system test failed", "error", err)
			fmt.Fprintf(os.Stderr, "Warning: Udev system test failed - %v\n", err)
		} else if !success {
			slog.Error("Udev system test failed", "error", ErrUdevSystemFailure)
			fmt.Fprintf(os.Stderr, "Warning: Udev system test failed - rules may not apply correctly\n")
		}
	}

	hasPCISerials, err := checkPCIFallbackForSerials(ctx, executor)
	if err != nil {
		slog.Warn("Failed to check for PCI fallback serials", "error", err)
	} else {
		slog.Debug("PCI fallback serial detection", "has_pci_serials", hasPCISerials)
	}

	soundSystem, err := detectSoundSystemType(ctx, executor)
	if err != nil {
		slog.Warn("Failed to detect sound system", "error", err)
	} else {
		slog.Info("Sound system detection", "system", soundSystem)
	}

	allUSBDevices, err := findAllUSBDevices(ctx, executor)
	if err != nil {
		slog.Error("Failed to enumerate all USB devices", "error", err)
	} else {
		slog.Debug("USB devices found", "count", len(allUSBDevices))
	}

	if ctx.Err() != nil {
		slog.Info("Operation interrupted, shutting down")
		fmt.Println("Operation interrupted. Shutting down...")
		errs := resourceTracker.CleanupAll()
		for _, err := range errs {
			slog.Error("Error during resource cleanup", "error", err)
		}
		return 0
	}

	cards, err := GetUSBSoundCards(ctx, executor, &config)
	if err != nil {
		if errors.Is(err, ErrNoUSBSoundCards) {
			slog.Error("No USB sound cards found")
			fmt.Println("No USB sound cards found.")
			return 0
		}
		slog.Error("Failed to get USB sound cards", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if config.ListOnly {
		showDeviceList(cards)
		return 0
	}

	if config.NonInteractive {
		err := nonInteractiveMode(ctx, &config, executor, fileAccess, cards)
		if err != nil {
			slog.Error("Non-interactive mode failed", "error", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	}

	// Interactive mode
	result, err := runUI(ctx, cards, &config, executor, fileAccess, resourceTracker)
	if err != nil {
		if errors.Is(err, ErrOperationCancelled) {
			slog.Info("Operation cancelled by user")
			fmt.Println("Operation cancelled by user.")
			return 0
		}

		if errors.Is(err, context.Canceled) {
			slog.Info("Operation interrupted, shutting down")
			fmt.Println("Operation interrupted. Shutting down...")
			return 0
		}

		slog.Error("UI error", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Println(result)

	if !strings.Contains(result, "operation cancelled") {
		fmt.Println("\nTo verify the rule files exist, run:")
		fmt.Printf("sudo ls -l %s\n", config.UdevRulesPath)
	}

	if errs := resourceTracker.CleanupAll(); len(errs) > 0 {
		slog.Warn("Some errors occurred during final cleanup", "count", len(errs))
		for _, err := range errs {
			slog.Error("Cleanup error", "error", err)
		}
	}

	return 0
}

// parseFlags sets up and parses command line flags into the config
func parseFlags(config *Config) {
	flag.StringVar(&config.UdevRulesPath, "rules-path", udevRulesDir, "Path to udev rules directory")
	flag.BoolVar(&config.ListOnly, "list", false, "List USB sound cards and exit")
	flag.BoolVar(&config.NonInteractive, "non-interactive", false, "Non-interactive mode")
	flag.StringVar(&config.DeviceName, "name", "", "Custom name for the device (non-interactive mode)")
	flag.StringVar(&config.VendorID, "vendor-id", "", "Vendor ID (non-interactive mode)")
	flag.StringVar(&config.ProductID, "product-id", "", "Product ID (non-interactive mode)")
	flag.BoolVar(&config.SkipReload, "skip-reload", false, "Skip reloading udev rules after creating them")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Show what would be done without making changes")
	flag.BoolVar(&config.ForceOverwrite, "force", false, "Force overwrite existing rules and accept virtual devices")
	flag.BoolVar(&config.IgnoreVirtual, "ignore-virtual", false, "Ignore virtual audio devices")
	flag.IntVar(&config.MaxBackupCount, "max-backups", maxBackupCount, "Maximum number of backups to keep per device")

	var logLevelStr string
	flag.StringVar(&logLevelStr, "log-level", string(LogLevelInfo), "Log level (debug, info, warn, error)")

	var commandTimeout int
	flag.IntVar(&commandTimeout, "command-timeout", int(DefaultTimeouts.CommandExecution/time.Second), "Command execution timeout in seconds")

	var lockTimeout int
	flag.IntVar(&lockTimeout, "lock-timeout", int(DefaultTimeouts.LockAcquisition/time.Second), "File lock acquisition timeout in seconds")

	var gracefulTimeoutSec int
	flag.IntVar(&gracefulTimeoutSec, "graceful-timeout", int(DefaultTimeouts.GracefulShutdown/time.Second), "Graceful shutdown timeout in seconds")

	var retries int
	flag.IntVar(&retries, "retries", config.MaxRetries, "Maximum number of retries for commands")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", AppName)
		fmt.Fprintf(os.Stderr, "Creates persistent device mappings for USB sound cards.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --list                  # List all USB sound cards\n", AppName)
		fmt.Fprintf(os.Stderr, "  %s                         # Interactive mode\n", AppName)
		fmt.Fprintf(os.Stderr, "  %s --non-interactive --vendor-id 1234 --product-id 5678 --name my_mic\n", AppName)
		fmt.Fprintf(os.Stderr, "  %s --dry-run --non-interactive --vendor-id 1234 --product-id 5678  # Show rule without creating it\n", AppName)
		fmt.Fprintf(os.Stderr, "  %s --force --ignore-virtual # Force overwrite existing rules and ignore virtual devices\n", AppName)
	}

	flag.Parse()

	config.LogLevel = LogLevel(logLevelStr)
	config.Timeouts.CommandExecution = time.Duration(commandTimeout) * time.Second
	config.Timeouts.LockAcquisition = time.Duration(lockTimeout) * time.Second
	config.Timeouts.GracefulShutdown = time.Duration(gracefulTimeoutSec) * time.Second
	config.MaxRetries = retries
}
