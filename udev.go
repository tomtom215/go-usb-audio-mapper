// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UdevRule represents a complete udev rule configuration
type UdevRule struct {
	Card     USBSoundCard
	Content  string
	Path     string
	Name     string
	DeviceID string
}

// createUdevRule creates a udev rule to give the sound card a persistent name
func createUdevRule(ctx context.Context, card *USBSoundCard, customName string, config *Config) (*UdevRule, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if card.VendorID == "" || card.ProductID == "" {
		return nil, fmt.Errorf("insufficient device information for card %s", card.CardNumber)
	}

	deviceName := card.FriendlyName
	if customName != "" {
		deviceName = cleanupName(customName)
	}

	var ruleBuilder strings.Builder

	// Header with documentation
	ruleBuilder.WriteString("# USB sound card persistent mapping created by usb-soundcard-mapper v")
	ruleBuilder.WriteString(AppVersion)
	ruleBuilder.WriteString("\n# Created: ")
	ruleBuilder.WriteString(time.Now().Format(time.RFC3339))
	ruleBuilder.WriteString("\n# Device: ")
	ruleBuilder.WriteString(card.Vendor)
	ruleBuilder.WriteString(" ")
	ruleBuilder.WriteString(card.Product)
	ruleBuilder.WriteString("\n# VID:PID: ")
	ruleBuilder.WriteString(card.VendorID)
	ruleBuilder.WriteString(":")
	ruleBuilder.WriteString(card.ProductID)

	if card.Serial != "" {
		ruleBuilder.WriteString("\n# Serial: ")
		ruleBuilder.WriteString(card.Serial)
	}

	if card.PhysicalPort != "" {
		ruleBuilder.WriteString("\n# USB Path: ")
		ruleBuilder.WriteString(card.PhysicalPort)
	}

	if card.IsVirtual {
		ruleBuilder.WriteString("\n# Note: This appears to be a virtual audio device")
	}

	ruleBuilder.WriteString("\n\n")

	// Rule 1: Priority rule for ACTION=="add" with full device attributes
	if card.Serial != "" && !strings.Contains(card.Serial, ":") {
		fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", ACTION==\"add\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", ATTRS{serial}==\"%s\", ATTR{id}=\"%s\"\n",
			card.VendorID, card.ProductID, card.Serial, deviceName)
	} else if card.PhysicalPort != "" {
		fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", ACTION==\"add\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", KERNELS==\"%s*\", ATTR{id}=\"%s\"\n",
			card.VendorID, card.ProductID, card.PhysicalPort, deviceName)
	} else {
		fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", ACTION==\"add\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", ATTR{id}=\"%s\"\n",
			card.VendorID, card.ProductID, deviceName)
	}

	// Rule 2: SOUND_INITIALIZED for after sound system is fully running
	if card.Serial != "" && !strings.Contains(card.Serial, ":") {
		fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", ENV{SOUND_INITIALIZED}==\"1\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", ATTRS{serial}==\"%s\", ATTR{id}=\"%s\"\n",
			card.VendorID, card.ProductID, card.Serial, deviceName)
	} else if card.PhysicalPort != "" {
		fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", ENV{SOUND_INITIALIZED}==\"1\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", KERNELS==\"%s*\", ATTR{id}=\"%s\"\n",
			card.VendorID, card.ProductID, card.PhysicalPort, deviceName)
	} else {
		fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", ENV{SOUND_INITIALIZED}==\"1\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", ATTR{id}=\"%s\"\n",
			card.VendorID, card.ProductID, deviceName)
	}

	// Rule 3: Universal rule for any sound card with specific USB vendor/product ID
	fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", ATTR{id}=\"%s\"\n",
		card.VendorID, card.ProductID, deviceName)

	// Rule 4: Alternative rule with KERNEL match
	fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", KERNEL==\"card*\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", ATTR{id}=\"%s\"\n",
		card.VendorID, card.ProductID, deviceName)

	// Rule 5: Symlink rule for easier access
	fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", ACTION==\"add\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", SYMLINK+=\"sound/by-id/%s\"\n",
		card.VendorID, card.ProductID, deviceName)

	// Rule 6: Fallback symlink with ACTION=="change"
	fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", ACTION==\"change\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", SYMLINK+=\"sound/by-id/%s\"\n",
		card.VendorID, card.ProductID, deviceName)

	// Rule 7: Control device symlink
	fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", KERNEL==\"controlC*\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", SYMLINK+=\"sound/%s/control\"\n",
		card.VendorID, card.ProductID, deviceName)

	// Rule 8: PCM playback symlink
	fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", KERNEL==\"pcmC*D*p\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", SYMLINK+=\"sound/%s/pcm_playback\"\n",
		card.VendorID, card.ProductID, deviceName)

	// Rule 9: PCM capture symlink
	fmt.Fprintf(&ruleBuilder, "SUBSYSTEM==\"sound\", KERNEL==\"pcmC*D*c\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", SYMLINK+=\"sound/%s/pcm_capture\"\n",
		card.VendorID, card.ProductID, deviceName)

	ruleFileName := fmt.Sprintf("89-usb-soundcard-%s-%s.rules", card.VendorID, card.ProductID)
	rulePath := filepath.Join(config.UdevRulesPath, ruleFileName)

	rule := &UdevRule{
		Card:     *card,
		Content:  ruleBuilder.String(),
		Path:     rulePath,
		Name:     ruleFileName,
		DeviceID: deviceName,
	}

	return rule, nil
}

// installUdevRule writes the rule to the filesystem using transactions
func installUdevRule(ctx context.Context, rule *UdevRule, config *Config, fileAccess *SafeFileAccess) error {
	slog.Info("Installing udev rule",
		"device", rule.Card.String(),
		"rule_path", rule.Path)

	if config.DryRun {
		slog.Info("Dry run mode - rule would be written to:", "path", rule.Path)
		fmt.Println("--- Rule Content ---")
		fmt.Println(rule.Content)
		fmt.Println("--------------------")
		return nil
	}

	transaction := NewTransaction()

	// Directory creation
	transaction.AddOperation(
		func() error {
			return os.MkdirAll(config.UdevRulesPath, 0755)
		},
		func() error { return nil },
	)

	// Backup if needed
	exists, err := fileExists(rule.Path)
	if err != nil {
		return fmt.Errorf("error checking if rule file exists: %w", err)
	}

	var backupPath string
	if exists && config.BackupRules {
		backupPath = rule.Path + ".bak." + time.Now().Format("20060102150405")

		transaction.AddOperation(
			func() error {
				content, err := os.ReadFile(rule.Path)
				if err != nil {
					return fmt.Errorf("failed to read existing rule file: %w", err)
				}
				return atomicWriteFile(backupPath, content, 0644, fileAccess, config.Timeouts.LockAcquisition)
			},
			func() error {
				if backupPath != "" {
					if exists, _ := fileExists(backupPath); exists {
						return os.Remove(backupPath)
					}
				}
				return nil
			},
		)
	}

	// Write rule if content differs
	shouldWrite := true
	if exists && !config.ForceOverwrite {
		content, err := os.ReadFile(rule.Path)
		if err == nil && string(content) == rule.Content {
			shouldWrite = false
			slog.Info("Rule file already exists with identical content, skipping write", "path", rule.Path)
		}
	}

	if shouldWrite {
		transaction.AddOperation(
			func() error {
				return atomicWriteFile(rule.Path, []byte(rule.Content), 0644, fileAccess, config.Timeouts.LockAcquisition)
			},
			func() error {
				if backupPath != "" && exists {
					content, err := os.ReadFile(backupPath)
					if err != nil {
						return fmt.Errorf("failed to read backup for rollback: %w", err)
					}
					return atomicWriteFile(rule.Path, content, 0644, fileAccess, config.Timeouts.LockAcquisition)
				} else if !exists {
					return os.Remove(rule.Path)
				}
				return nil
			},
		)
	}

	// Modprobe config
	transaction.AddOperation(
		func() error {
			return createModprobeConfig(&rule.Card, config, fileAccess)
		},
		func() error { return nil },
	)

	if err := transaction.Execute(); err != nil {
		slog.Error("Installation transaction failed", "error", err)
		return fmt.Errorf("installation failed: %w", ErrTransactionFailed)
	}

	transaction.Commit()
	return nil
}

// createModprobeConfig creates a modprobe configuration for better device handling
func createModprobeConfig(card *USBSoundCard, config *Config, fileAccess *SafeFileAccess) error {
	modprobePath := "/etc/modprobe.d"
	exists, err := directoryExists(modprobePath)
	if err != nil {
		return fmt.Errorf("error checking modprobe directory: %w", err)
	}

	if !exists {
		return nil
	}

	modprobeFile := filepath.Join(modprobePath, fmt.Sprintf("99-soundcard-%s-%s.conf",
		card.VendorID, card.ProductID))

	exists, err = fileExists(modprobeFile)
	if err != nil {
		return fmt.Errorf("error checking modprobe file: %w", err)
	}

	if exists && !config.ForceOverwrite {
		return nil
	}

	if config.DryRun {
		slog.Info("Dry run mode - would create modprobe configuration", "path", modprobeFile)
		return nil
	}

	var contentBuilder strings.Builder
	contentBuilder.WriteString(fmt.Sprintf("# Modprobe options for USB sound card %s %s\n",
		card.Vendor, card.Product))
	contentBuilder.WriteString("options snd_usb_audio index=-2\n")

	if err := atomicWriteFile(modprobeFile, []byte(contentBuilder.String()), 0644, fileAccess, config.Timeouts.LockAcquisition); err != nil {
		return fmt.Errorf("failed to write modprobe configuration: %w", err)
	}

	return nil
}

// reloadUdevRules triggers a reload of udev rules
func reloadUdevRules(ctx context.Context, executor *CommandExecutor, config *Config) error {
	if config.DryRun {
		slog.Info("Dry run mode - skipping udev rules reload")
		return nil
	}

	transaction := NewTransaction()

	transaction.AddOperation(
		func() error {
			_, err := executor.ExecuteCommand(ctx, "udevadm", "control", "--reload-rules")
			if err != nil {
				return fmt.Errorf("failed to reload udev rules: %w", err)
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
			_, err := executor.ExecuteCommand(ctx, "udevadm", "trigger", "--action=add", "--subsystem-match=sound")
			if err != nil {
				return fmt.Errorf("failed to trigger udev rules with add action: %w", err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(config.Timeouts.TriggerActionWait):
			}
			return nil
		},
		func() error { return nil },
	)

	transaction.AddOperation(
		func() error {
			_, err := executor.ExecuteCommand(ctx, "udevadm", "trigger", "--action=change", "--subsystem-match=sound")
			if err != nil {
				return fmt.Errorf("failed to trigger udev rules with change action: %w", err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(config.Timeouts.TriggerActionWait):
			}
			return nil
		},
		func() error { return nil },
	)

	if err := transaction.Execute(); err != nil {
		return err
	}

	return nil
}

// verifyUdevRuleInstallation checks if the rule is properly installed
func verifyUdevRuleInstallation(ctx context.Context, executor *CommandExecutor, card *USBSoundCard, customName string, config *Config) (bool, error) {
	if config.DryRun {
		slog.Info("Dry run mode - skipping rule verification")
		return true, nil
	}

	cardPath := fmt.Sprintf("/sys/class/sound/card%s", card.CardNumber)
	exists, err := pathExists(cardPath)
	if err != nil {
		return false, fmt.Errorf("error checking card path: %w", err)
	}

	if !exists {
		return false, ErrDeviceDisconnected
	}

	verificationMethods := []struct {
		name     string
		function func() (bool, error)
	}{
		{
			name: "udevadm info",
			function: func() (bool, error) {
				output, err := executor.ExecuteCommand(ctx, "udevadm", "info", "--path", cardPath)
				if err != nil {
					return false, fmt.Errorf("failed to get udevadm info: %w", err)
				}
				return strings.Contains(output, fmt.Sprintf("ID_SOUND_ID=%s", customName)), nil
			},
		},
		{
			name: "symlink check",
			function: func() (bool, error) {
				symlinkPath := fmt.Sprintf("/dev/sound/by-id/%s", customName)
				exists, err := fileExists(symlinkPath)
				if err != nil {
					return false, fmt.Errorf("error checking symlink existence: %w", err)
				}
				return exists, nil
			},
		},
		{
			name: "udevadm trigger",
			function: func() (bool, error) {
				_, err := executor.ExecuteCommand(ctx, "udevadm", "trigger", "--action=add",
					"--property-match=SUBSYSTEM=sound",
					fmt.Sprintf("--property-match=ID_VENDOR_ID=%s", card.VendorID),
					fmt.Sprintf("--property-match=ID_MODEL_ID=%s", card.ProductID))

				if err != nil {
					return false, fmt.Errorf("failed to trigger specific udev rules: %w", err)
				}

				select {
				case <-ctx.Done():
					return false, ctx.Err()
				case <-time.After(config.Timeouts.TriggerActionWait):
				}

				symlinkPath := fmt.Sprintf("/dev/sound/by-id/%s", customName)
				exists, err := fileExists(symlinkPath)
				if err != nil {
					return false, fmt.Errorf("error checking symlink existence: %w", err)
				}
				return exists, nil
			},
		},
		{
			name: "aplay output",
			function: func() (bool, error) {
				aplayOutput, err := executor.ExecuteCommand(ctx, "aplay", "-L")
				if err != nil {
					return false, fmt.Errorf("failed to check aplay -L output: %w", err)
				}
				return strings.Contains(aplayOutput, customName), nil
			},
		},
	}

	for _, method := range verificationMethods {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		success, err := method.function()
		if err != nil {
			slog.Warn(fmt.Sprintf("Verification method %s failed", method.name), "error", err)
			continue
		}

		if success {
			slog.Info(fmt.Sprintf("Verified successful udev rule installation via %s!", method.name))
			return true, nil
		}
	}

	return false, ErrRuleVerificationFail
}
