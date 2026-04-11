package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// backupExistingUdevRules creates a backup of existing rules files
func backupExistingUdevRules(card USBSoundCard, config Config, fileAccess *SafeFileAccess) error {
	if !config.BackupRules || config.DryRun {
		return nil
	}

	backupDir := filepath.Join(config.UdevRulesPath, "backups")

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		slog.Warn("Failed to create backup directory", "error", err)
	}

	patterns := []string{
		fmt.Sprintf("*usb*soundcard*%s*%s*.rules", card.VendorID, card.ProductID),
		fmt.Sprintf("*usb*sound*%s*%s*.rules", card.VendorID, card.ProductID),
		fmt.Sprintf("*sound*%s*%s*.rules", card.VendorID, card.ProductID),
		fmt.Sprintf("*card*%s*%s*.rules", card.VendorID, card.ProductID),
		fmt.Sprintf("*audio*%s*%s*.rules", card.VendorID, card.ProductID),
	}

	existingBackups := 0
	backupPrefix := fmt.Sprintf("backup_%s_%s_", card.VendorID, card.ProductID)

	entries, err := os.ReadDir(backupDir)
	if err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), backupPrefix) {
				existingBackups++
			}
		}
	}

	if existingBackups >= config.MaxBackupCount {
		slog.Warn("Backup limit reached, skipping additional backups",
			"limit", config.MaxBackupCount,
			"existing", existingBackups)
		return nil
	}

	backupCount := 0
	for _, pattern := range patterns {
		if backupCount >= config.MaxBackupCount {
			slog.Info("Reached maximum backup count", "max", config.MaxBackupCount)
			break
		}

		matches, err := filepath.Glob(filepath.Join(config.UdevRulesPath, pattern))
		if err != nil {
			slog.Error("Error searching for existing rules", "error", err)
			continue
		}

		for _, match := range matches {
			if strings.Contains(match, fmt.Sprintf("89-usb-soundcard-%s-%s.rules", card.VendorID, card.ProductID)) {
				continue
			}

			timestamp := time.Now().Format("20060102150405")
			backupFile := filepath.Join(backupDir, fmt.Sprintf("%s%s_%s",
				backupPrefix, filepath.Base(match), timestamp))

			slog.Info("Backing up existing rule file", "source", match, "backup", backupFile)

			transaction := NewTransaction()

			var content []byte
			transaction.AddOperation(
				func() error {
					_, err := fileAccess.LockFile(match, config.Timeouts.LockAcquisition)
					if err != nil {
						return fmt.Errorf("failed to acquire lock on file during backup: %w", err)
					}
					defer fileAccess.UnlockFile(match)

					c, err := os.ReadFile(match)
					if err != nil {
						return fmt.Errorf("failed to read file: %w", err)
					}

					content = c
					return nil
				},
				func() error { return nil },
			)

			transaction.AddOperation(
				func() error {
					return atomicWriteFile(backupFile, content, 0644, fileAccess, config.Timeouts.LockAcquisition)
				},
				func() error {
					if exists, _ := fileExists(backupFile); exists {
						return os.Remove(backupFile)
					}
					return nil
				},
			)

			if err := transaction.Execute(); err != nil {
				slog.Error("Failed to back up rule file", "source", match, "error", err)
				continue
			}

			backupCount++
			if backupCount >= config.MaxBackupCount {
				slog.Info("Reached maximum backup count", "max", config.MaxBackupCount)
				break
			}
		}
	}

	if backupCount > 0 {
		slog.Info("Created backups of existing rule files", "count", backupCount)
	}
	return nil
}

// testUdevSystem performs a basic test of the udev system
func testUdevSystem(ctx context.Context, executor *CommandExecutor, config Config, fileAccess *SafeFileAccess) (bool, error) {
	slog.Info("Testing if udev rule system is working properly...")

	if config.DryRun {
		slog.Info("Dry run mode - skipping udev system test")
		return true, nil
	}

	transaction := NewTransaction()

	testRuleFile := filepath.Join(config.UdevRulesPath, "99-test-usb-soundcard-mapper.rules")
	testRuleContent := "# Test rule to check if udev is functioning properly\n"

	transaction.AddOperation(
		func() error {
			_, err := fileAccess.LockFile(testRuleFile, config.Timeouts.LockAcquisition)
			if err != nil {
				return fmt.Errorf("failed to acquire lock on test rule file: %w", err)
			}
			defer fileAccess.UnlockFile(testRuleFile)
			return os.WriteFile(testRuleFile, []byte(testRuleContent), 0644)
		},
		func() error {
			removeErr := os.Remove(testRuleFile)
			if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				slog.Error("Failed to remove test udev rule", "error", removeErr)
			}
			return nil
		},
	)

	transaction.AddOperation(
		func() error {
			_, err := executor.ExecuteCommand(ctx, "udevadm", "control", "--reload-rules")
			if err != nil {
				return fmt.Errorf("failed to reload udev rules during test: %w", err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(config.Timeouts.RuleReloadWait):
			}
			return nil
		},
		func() error { return nil },
	)

	transaction.AddOperation(
		func() error {
			_, err := executor.ExecuteCommand(ctx, "udevadm", "trigger", "--action=add", "--subsystem-match=usb")
			if err != nil {
				return fmt.Errorf("failed to trigger udev rules during test: %w", err)
			}
			return nil
		},
		func() error { return nil },
	)

	transaction.AddOperation(
		func() error {
			removeErr := os.Remove(testRuleFile)
			if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				slog.Error("Failed to remove test udev rule", "error", removeErr)
				return fmt.Errorf("failed to remove test rule: %w", removeErr)
			}
			return nil
		},
		func() error { return nil },
	)

	if err := transaction.Execute(); err != nil {
		return false, fmt.Errorf("udev system test failed: %w", err)
	}

	transaction.Commit()
	slog.Info("Udev system test passed")
	return true, nil
}
