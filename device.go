// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DeviceStatus represents current device status
type DeviceStatus int

const (
	DeviceStatusConnected DeviceStatus = iota
	DeviceStatusDisconnected
	DeviceStatusUnknown
)

// USBSoundCard represents a USB sound card device with all necessary attributes
type USBSoundCard struct {
	CardNumber    string
	DevicePath    string
	Vendor        string
	Product       string
	VendorID      string
	ProductID     string
	Serial        string
	BusID         string
	DeviceID      string
	PhysicalPort  string
	FriendlyName  string
	Detected      time.Time
	Status        DeviceStatus
	IsVirtual     bool
	ValidationErr error
}

// String returns a formatted representation of the sound card
func (c USBSoundCard) String() string {
	var attrs []string
	attrs = append(attrs, fmt.Sprintf("Card: %s", c.CardNumber))
	attrs = append(attrs, fmt.Sprintf("Device: %s %s", c.Vendor, c.Product))
	attrs = append(attrs, fmt.Sprintf("VID:PID: %s:%s", c.VendorID, c.ProductID))
	if c.Serial != "" {
		attrs = append(attrs, fmt.Sprintf("Serial: %s", c.Serial))
	}
	if c.PhysicalPort != "" {
		attrs = append(attrs, fmt.Sprintf("Port: %s", c.PhysicalPort))
	}
	if c.IsVirtual {
		attrs = append(attrs, "Type: Virtual")
	}
	return strings.Join(attrs, ", ")
}

// Validate validates the sound card attributes
func (c *USBSoundCard) Validate() error {
	if c.CardNumber == "" {
		return errors.New("missing card number")
	}

	if c.VendorID == "" {
		return errors.New("missing vendor ID")
	}

	if c.ProductID == "" {
		return errors.New("missing product ID")
	}

	if !vendorIDRegex.MatchString(c.VendorID) {
		return fmt.Errorf("invalid vendor ID format: %s", c.VendorID)
	}

	if !productIDRegex.MatchString(c.ProductID) {
		return fmt.Errorf("invalid product ID format: %s", c.ProductID)
	}

	if c.Serial != "" {
		if !serialRegex.MatchString(c.Serial) {
			return fmt.Errorf("invalid serial number format: %s", c.Serial)
		}
	}

	return nil
}

// DeviceRegistry manages a thread-safe collection of sound cards
type DeviceRegistry struct {
	devices map[string]USBSoundCard
	mu      sync.RWMutex
}

// NewDeviceRegistry creates a new device registry
func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{
		devices: make(map[string]USBSoundCard),
	}
}

// AddDevice adds a device to the registry
func (dr *DeviceRegistry) AddDevice(card USBSoundCard) {
	dr.mu.Lock()
	defer dr.mu.Unlock()

	key := dr.generateDeviceKey(card)
	card.Detected = time.Now()
	dr.devices[key] = card
}

// GetDevices returns all devices in the registry
func (dr *DeviceRegistry) GetDevices() []USBSoundCard {
	dr.mu.RLock()
	defer dr.mu.RUnlock()

	devices := make([]USBSoundCard, 0, len(dr.devices))
	for _, device := range dr.devices {
		devices = append(devices, device)
	}

	return devices
}

// GetDevice retrieves a specific device by key
func (dr *DeviceRegistry) GetDevice(card USBSoundCard) (USBSoundCard, bool) {
	dr.mu.RLock()
	defer dr.mu.RUnlock()

	key := dr.generateDeviceKey(card)
	device, exists := dr.devices[key]
	return device, exists
}

// UpdateDeviceStatus updates the status of a device
func (dr *DeviceRegistry) UpdateDeviceStatus(card USBSoundCard, status DeviceStatus) {
	dr.mu.Lock()
	defer dr.mu.Unlock()

	key := dr.generateDeviceKey(card)
	if device, exists := dr.devices[key]; exists {
		device.Status = status
		dr.devices[key] = device
	}
}

// generateDeviceKey creates a unique key for a device
func (dr *DeviceRegistry) generateDeviceKey(card USBSoundCard) string {
	if card.Serial != "" && !strings.Contains(card.Serial, ":") {
		return fmt.Sprintf("%s:%s:%s", card.VendorID, card.ProductID, card.Serial)
	} else if card.PhysicalPort != "" {
		return fmt.Sprintf("%s:%s:%s", card.VendorID, card.ProductID, card.PhysicalPort)
	}

	return fmt.Sprintf("%s:%s:%s", card.VendorID, card.ProductID, card.CardNumber)
}

// GetUSBSoundCards detects all USB sound cards in the system
func GetUSBSoundCards(ctx context.Context, executor *CommandExecutor, config Config) ([]USBSoundCard, error) {
	registry := NewDeviceRegistry()

	output, err := executor.ExecuteCommand(ctx, "aplay", "-l")
	if err != nil {
		return nil, fmt.Errorf("failed to list sound cards: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(output))

	var cards []USBSoundCard
	var errs []error

	var cardNumbers []string
	for scanner.Scan() {
		line := scanner.Text()
		matches := cardRegex.FindStringSubmatch(line)
		if matches != nil && len(matches) >= 4 {
			cardNumber := matches[1]

			if !strings.Contains(strings.ToLower(line), "usb") {
				continue
			}

			cardNumbers = append(cardNumbers, cardNumber)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning aplay output: %w", err)
	}

	for _, cardNum := range cardNumbers {
		if ctx.Err() != nil {
			return cards, ctx.Err()
		}

		card, err := getCardDetails(ctx, executor, cardNum, config)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get details for card %s: %w", cardNum, err))
			continue
		}

		if card.IsVirtual && config.IgnoreVirtual {
			slog.Info("Skipping virtual device", "card", card.String())
			continue
		}

		if err := card.Validate(); err != nil {
			card.ValidationErr = err
			slog.Warn("Card validation failed", "card", card.CardNumber, "error", err)
		}

		cards = append(cards, card)
		registry.AddDevice(card)
	}

	if len(cards) == 0 && len(errs) == 0 {
		return nil, ErrNoUSBSoundCards
	}

	if len(errs) > 0 {
		var errStrings []string
		for _, err := range errs {
			errStrings = append(errStrings, err.Error())
		}

		if len(cards) == 0 {
			return nil, fmt.Errorf("failed to process sound cards: %s", strings.Join(errStrings, "; "))
		}

		slog.Warn("Some cards could not be processed", "errors", strings.Join(errStrings, "; "))
	}

	return cards, nil
}

// getCardDetails gets detailed information about a sound card
func getCardDetails(ctx context.Context, executor *CommandExecutor, cardNumber string, config Config) (USBSoundCard, error) {
	card := USBSoundCard{
		CardNumber: cardNumber,
		DevicePath: fmt.Sprintf("/dev/snd/card%s", cardNumber),
		Status:     DeviceStatusConnected,
	}

	sysfsPath := fmt.Sprintf("/sys/class/sound/card%s", cardNumber)

	if ok, err := pathExists(sysfsPath); !ok {
		if err != nil {
			return card, fmt.Errorf("error checking card path: %w", err)
		}
		return card, ErrDeviceDisconnected
	}

	output, err := executor.ExecuteCommand(ctx, "udevadm", "info", "--attribute-walk", "--path", sysfsPath)
	if err != nil {
		return card, fmt.Errorf("failed to get device info: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	vendorRegexp := regexp.MustCompile(`ATTRS{idVendor}=="([^"]*)"`)
	productRegexp := regexp.MustCompile(`ATTRS{idProduct}=="([^"]*)"`)
	serialRegexpLocal := regexp.MustCompile(`ATTRS{serial}=="([^"]*)"`)
	busPathRegexp := regexp.MustCompile(`KERNELS=="([0-9\-\.]+)"`)
	driverRegexp := regexp.MustCompile(`DRIVERS=="([^"]*)"`)

	isVirtualDevice := false

	for scanner.Scan() {
		line := scanner.Text()

		if matches := vendorRegexp.FindStringSubmatch(line); matches != nil && card.VendorID == "" {
			card.VendorID = matches[1]
		}

		if matches := productRegexp.FindStringSubmatch(line); matches != nil && card.ProductID == "" {
			card.ProductID = matches[1]
		}

		if matches := serialRegexpLocal.FindStringSubmatch(line); matches != nil && card.Serial == "" {
			card.Serial = sanitizeSerial(matches[1])
		}

		if matches := busPathRegexp.FindStringSubmatch(line); matches != nil && card.PhysicalPort == "" {
			card.PhysicalPort = matches[1]

			parts := strings.Split(matches[1], "-")
			if len(parts) >= 2 {
				card.BusID = parts[0]
				if len(parts) >= 3 {
					card.DeviceID = strings.Split(parts[1], ".")[0]
				}
			}
		}

		if matches := driverRegexp.FindStringSubmatch(line); matches != nil {
			driverName := matches[1]
			if isVirtualDriver(driverName) {
				isVirtualDevice = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return card, fmt.Errorf("error scanning udevadm output: %w", err)
	}

	card.IsVirtual = isVirtualDevice

	if card.IsVirtual && !config.IgnoreVirtual {
		slog.Warn("Virtual audio device detected", "card", cardNumber)
	}

	if card.VendorID == "" || card.ProductID == "" {
		return card, fmt.Errorf("insufficient device information for card %s", card.CardNumber)
	}

	if card.VendorID != "" && card.ProductID != "" && ctx.Err() == nil {
		lsusbOutput, err := executor.ExecuteCommand(ctx, "lsusb", "-d", fmt.Sprintf("%s:%s", card.VendorID, card.ProductID))
		if err == nil && len(lsusbOutput) > 0 {
			lsusbRegexp := regexp.MustCompile(`ID [0-9a-f]+:[0-9a-f]+ (.+)`)
			if matches := lsusbRegexp.FindStringSubmatch(lsusbOutput); matches != nil {
				fullName := matches[1]

				parts := strings.SplitN(fullName, " ", 2)
				if len(parts) >= 2 {
					card.Vendor = parts[0]
					card.Product = parts[1]
				} else {
					card.Vendor = "USB"
					card.Product = fullName
				}
			}
		}
	}

	if card.Vendor == "" {
		card.Vendor = fmt.Sprintf("USB-%s", card.VendorID)
	}
	if card.Product == "" {
		card.Product = fmt.Sprintf("Audio-%s", card.ProductID)
	}

	if card.Serial != "" && !strings.Contains(card.Serial, ":") {
		cleanSerial := cleanupName(card.Serial)
		card.FriendlyName = fmt.Sprintf("usb_%s_%s_%s", card.VendorID, card.ProductID, cleanSerial)
	} else if card.PhysicalPort != "" {
		card.FriendlyName = fmt.Sprintf("usb_%s_%s_port%s", card.VendorID, card.ProductID,
			strings.ReplaceAll(card.PhysicalPort, "-", "_"))
	} else {
		card.FriendlyName = fmt.Sprintf("usb_%s_%s_%s", card.VendorID, card.ProductID, card.CardNumber)
	}

	card.FriendlyName = cleanupName(card.FriendlyName)

	return card, nil
}

// sanitizeSerial sanitizes a serial number to prevent security issues
func sanitizeSerial(serial string) string {
	return unsafeCharsRegex.ReplaceAllString(serial, "_")
}

// isVirtualDriver checks if a driver name indicates a virtual audio device
func isVirtualDriver(driver string) bool {
	virtualDrivers := []string{
		"snd_dummy", "snd_aloop", "snd_virmidi", "snd_pcm_oss",
		"snd_mixer_oss", "snd_seq", "snd_seq_dummy", "snd_seq_oss",
	}

	for _, vd := range virtualDrivers {
		if driver == vd {
			return true
		}
	}

	return false
}

// cleanupName ensures the generated name is valid for udev rules
func cleanupName(name string) string {
	name = nonAlphaNumRegex.ReplaceAllString(name, "_")

	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "usb_" + name
	}

	maxLength := 64
	if len(name) > maxLength {
		name = name[:maxLength]
	}

	return name
}

// showDeviceList displays a list of USB sound cards
func showDeviceList(cards []USBSoundCard) {
	if len(cards) == 0 {
		fmt.Println("No USB sound cards found.")
		return
	}

	fmt.Println("USB Sound Cards:")
	fmt.Println("---------------")

	for i, card := range cards {
		fmt.Printf("%d. Card %s: %s %s (VID:PID %s:%s)\n",
			i+1, card.CardNumber, card.Vendor, card.Product, card.VendorID, card.ProductID)

		if card.Serial != "" {
			fmt.Printf("   Serial: %s\n", card.Serial)
		}

		if card.PhysicalPort != "" {
			fmt.Printf("   Physical Port: %s\n", card.PhysicalPort)
		}

		if card.IsVirtual {
			fmt.Printf("   Type: Virtual Device\n")
		}

		fmt.Printf("   Suggested Name: %s\n", card.FriendlyName)

		if card.ValidationErr != nil {
			fmt.Printf("   Validation Warning: %s\n", card.ValidationErr)
		}

		fmt.Println()
	}
}

// findAllUSBDevices gets information about all connected USB devices
func findAllUSBDevices(ctx context.Context, executor *CommandExecutor) (map[string]map[string]string, error) {
	devices := make(map[string]map[string]string)

	output, err := executor.ExecuteCommand(ctx, "lsusb")
	if err != nil {
		return nil, fmt.Errorf("failed to run lsusb: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	lsusbRegexp := regexp.MustCompile(`Bus (\d{3}) Device (\d{3}): ID ([0-9a-f]{4}):([0-9a-f]{4}) (.+)`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := lsusbRegexp.FindStringSubmatch(line)

		if matches != nil && len(matches) >= 6 {
			busNum := matches[1]
			devNum := matches[2]
			vendorID := matches[3]
			productID := matches[4]
			deviceName := matches[5]

			busNum = strings.TrimLeft(busNum, "0")
			devNum = strings.TrimLeft(devNum, "0")

			deviceID := fmt.Sprintf("%s:%s", busNum, devNum)

			devices[deviceID] = map[string]string{
				"bus":       busNum,
				"device":    devNum,
				"vendorID":  vendorID,
				"productID": productID,
				"name":      deviceName,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return devices, fmt.Errorf("error scanning lsusb output: %w", err)
	}

	return devices, nil
}
