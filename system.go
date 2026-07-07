// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"
	"time"
)

// checkElevatedPrivileges checks if the process has the necessary privileges
func checkElevatedPrivileges() (bool, error) {
	currentUser, err := user.Current()
	if err != nil {
		return false, fmt.Errorf("failed to get current user: %w", err)
	}

	return currentUser.Uid == "0", nil
}

// checkAndFixPermissions ensures the udev rules directory has the correct permissions
func checkAndFixPermissions(config *Config) error {
	if !pathSafeRegex.MatchString(config.UdevRulesPath) {
		return fmt.Errorf("unsafe udev rules path: %s: %w", config.UdevRulesPath, ErrInvalidPath)
	}

	info, err := os.Stat(config.UdevRulesPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = os.MkdirAll(config.UdevRulesPath, 0o755) // #nosec G301 -- /etc/udev/rules.d is world-readable/traversable (0755) by system convention
			if err != nil {
				return fmt.Errorf("failed to create udev rules directory: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to check udev rules directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s exists but is not a directory", config.UdevRulesPath)
	}

	if info.Mode().Perm()&0o755 != 0o755 {
		slog.Info("Fixing permissions on rules directory", "path", config.UdevRulesPath)
		err = os.Chmod(config.UdevRulesPath, 0o755) // #nosec G302 -- /etc/udev/rules.d must be world-readable/traversable (0755) by system convention
		if err != nil {
			return fmt.Errorf("failed to set permissions on udev rules directory: %w", err)
		}
	}

	return nil
}

// detectSoundSystemType checks what sound system is in use
func detectSoundSystemType(ctx context.Context, executor *CommandExecutor) string {
	_, err := executor.ExecuteCommand(ctx, "pipewire", "--version")
	if err == nil {
		slog.Info("Detected PipeWire sound system")
		return "pipewire"
	}

	_, err = executor.ExecuteCommand(ctx, "pulseaudio", "--version")
	if err == nil {
		slog.Info("Detected PulseAudio sound system")
		return "pulseaudio"
	}

	_, err = executor.ExecuteCommand(ctx, "jackd", "--version")
	if err == nil {
		slog.Info("Detected JACK sound system")
		return "jack"
	}

	slog.Info("Assuming ALSA sound system")
	return "alsa"
}

// checkPCIFallbackForSerials verifies if PCI paths are being used as serial numbers
func checkPCIFallbackForSerials(ctx context.Context, executor *CommandExecutor) (bool, error) {
	output, err := executor.ExecuteCommand(ctx, "lsusb", "-v")
	if err != nil {
		return false, fmt.Errorf("could not check for PCI fallback serial numbers: %w", err)
	}

	hasPCISerials := strings.Contains(output, "iSerial") && strings.Contains(output, ":")
	if hasPCISerials {
		slog.Info("Detected devices with PCI path-like serial numbers. Special handling will be applied.")
	}

	return hasPCISerials, nil
}

// setupSignalHandling sets up graceful shutdown on system signals
func setupSignalHandling(ctx context.Context, cancel context.CancelFunc, resourceTracker *ResourceTracker, config *Config) {
	c := make(chan os.Signal, 1)
	// os.Interrupt is syscall.SIGINT on Unix, so it is not listed twice.
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP,
		syscall.SIGQUIT, syscall.SIGABRT)

	go func() {
		select {
		case <-ctx.Done():
			return
		case sig := <-c:
			slog.Info("Received signal, shutting down gracefully", "signal", sig)

			cancel()

			// Perform cleanup in the background and enforce a hard deadline:
			// if graceful shutdown overruns GracefulShutdown, force-exit rather
			// than risk hanging on a stuck resource.
			done := make(chan struct{})
			go func() {
				if err := resourceTracker.WaitForCompletion(config.Timeouts.GracefulShutdown); err != nil {
					slog.Error("Error waiting for operations to complete", "error", err)
				}
				for _, err := range resourceTracker.CleanupAll() {
					slog.Error("Error during resource cleanup", "error", err)
				}
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(config.Timeouts.GracefulShutdown):
				slog.Error("Graceful shutdown timed out, forcing exit")
				os.Exit(1)
			}
		}
	}()
}

// initLogger initializes the structured logger
func initLogger(level LogLevel) {
	var logLevel slog.Level
	switch level {
	case LogLevelDebug:
		logLevel = slog.LevelDebug
	case LogLevelInfo:
		logLevel = slog.LevelInfo
	case LogLevelWarn:
		logLevel = slog.LevelWarn
	case LogLevelError:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(time.Now().Format(time.RFC3339)),
				}
			}
			return a
		},
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
