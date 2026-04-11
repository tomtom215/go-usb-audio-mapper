// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// performInstallation executes the full installation pipeline for a device
func performInstallation(ctx context.Context, card USBSoundCard, customName string, config Config, executor *CommandExecutor, fileAccess *SafeFileAccess) (string, error) {
	rule, err := createUdevRule(ctx, card, customName, config)
	if err != nil {
		slog.Error("Failed to create udev rule", "error", err)
		return "", fmt.Errorf("failed to create udev rule: %w", err)
	}

	if err := backupExistingUdevRules(card, config, fileAccess); err != nil {
		slog.Warn("Failed to backup existing rules", "error", err)
	}

	if err := installUdevRule(ctx, rule, config, fileAccess); err != nil {
		slog.Error("Failed to install udev rule", "error", err)
		return "", fmt.Errorf("failed to install udev rule: %w", err)
	}

	var warning string
	if !config.SkipReload {
		if err := reloadUdevRules(ctx, executor, config); err != nil {
			slog.Error("Failed to reload udev rules", "error", err)
			return "", fmt.Errorf("failed to reload udev rules: %w", err)
		}

		success, err := verifyUdevRuleInstallation(ctx, executor, card, customName, config)
		if err != nil {
			if errors.Is(err, ErrDeviceDisconnected) {
				warning = "Device appears to have been disconnected. Rules were created but could not be verified."
			} else {
				slog.Warn("Rule verification issue", "error", err)
				warning = fmt.Sprintf("Rules created but verification had issues: %v", err)
			}
		} else if !success {
			warning = "Rules created but verification could not confirm they were applied correctly."
		}
	}

	var messageBuilder strings.Builder
	messageBuilder.WriteString(fmt.Sprintf("Created persistent mapping for %s %s (VID:PID %s:%s) as '%s'\n\n",
		card.Vendor, card.Product,
		card.VendorID, card.ProductID,
		customName))

	messageBuilder.WriteString(fmt.Sprintf(
		"The sound card will use this name consistently across reboots and reconnections.\n"+
			"You can see this device in 'aplay -l' output as card with ID '%s'\n"+
			"once you disconnect and reconnect the device.\n", customName))

	if warning != "" {
		messageBuilder.WriteString("\nWarning: " + warning)
	}

	return messageBuilder.String(), nil
}

// nonInteractiveMode handles the non-interactive operation
func nonInteractiveMode(ctx context.Context, config Config, executor *CommandExecutor, fileAccess *SafeFileAccess, cards []USBSoundCard) error {
	if config.VendorID == "" || config.ProductID == "" {
		return fmt.Errorf("in non-interactive mode, --vendor-id and --product-id are required: %w", ErrInvalidDeviceParams)
	}

	if !vendorIDRegex.MatchString(config.VendorID) {
		return fmt.Errorf("invalid vendor ID format: %s: %w", config.VendorID, ErrInvalidDeviceParams)
	}

	if !productIDRegex.MatchString(config.ProductID) {
		return fmt.Errorf("invalid product ID format: %s: %w", config.ProductID, ErrInvalidDeviceParams)
	}

	var selectedCard USBSoundCard
	found := false

	for _, card := range cards {
		if card.VendorID == config.VendorID && card.ProductID == config.ProductID {
			selectedCard = card
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("no USB sound card found with VID:PID %s:%s: %w",
			config.VendorID, config.ProductID, ErrNoUSBSoundCards)
	}

	if selectedCard.IsVirtual && !config.IgnoreVirtual {
		slog.Warn("Selected device appears to be virtual", "card", selectedCard.String())
		if !config.ForceOverwrite {
			return fmt.Errorf("selected device appears to be virtual: %w", ErrVirtualDevice)
		}
	}

	customName := selectedCard.FriendlyName
	if config.DeviceName != "" {
		customName = cleanupName(config.DeviceName)
	}

	if customName == "" {
		return ErrDeviceNameEmpty
	}

	transaction := NewTransaction()

	transaction.AddOperation(
		func() error {
			return backupExistingUdevRules(selectedCard, config, fileAccess)
		},
		func() error { return nil },
	)

	var rule *UdevRule
	transaction.AddOperation(
		func() error {
			var err error
			rule, err = createUdevRule(ctx, selectedCard, customName, config)
			if err != nil {
				return fmt.Errorf("failed to create udev rule: %w", err)
			}
			return installUdevRule(ctx, rule, config, fileAccess)
		},
		func() error {
			if rule != nil && !config.DryRun {
				if exists, _ := fileExists(rule.Path); exists {
					return os.Remove(rule.Path)
				}
			}
			return nil
		},
	)

	if !config.SkipReload {
		transaction.AddOperation(
			func() error {
				return reloadUdevRules(ctx, executor, config)
			},
			func() error { return nil },
		)
	}

	if err := transaction.Execute(); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	if !config.DryRun && !config.SkipReload {
		success, err := verifyUdevRuleInstallation(ctx, executor, selectedCard, customName, config)
		if err != nil {
			if errors.Is(err, ErrDeviceDisconnected) {
				slog.Warn("Device appears to have been disconnected. Rules were created but could not be verified.")
			} else {
				slog.Warn("Rule verification issue", "error", err)
			}
		} else if !success {
			slog.Warn("Rules created but verification could not confirm they were applied correctly.")
		}
	}

	fmt.Printf("Created persistent mapping for %s %s (VID:PID %s:%s) as '%s'\n",
		selectedCard.Vendor, selectedCard.Product, selectedCard.VendorID,
		selectedCard.ProductID, customName)

	fmt.Println("\nImportant: For the changes to take full effect, please:")
	fmt.Println("1. Disconnect and reconnect the USB sound device, or")
	fmt.Println("2. Reboot your system")

	fmt.Println("\nFor immediate application of rules without rebooting, run:")
	fmt.Printf("sudo udevadm control --reload-rules && sudo udevadm trigger --action=add --subsystem-match=sound\n")

	transaction.Commit()
	return nil
}
