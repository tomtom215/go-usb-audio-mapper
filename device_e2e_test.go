// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

// End-to-end detection tests that exercise the real command-execution and
// parsing pipeline against fake lsusb/aplay/udevadm binaries — no hardware.

package main

import (
	"context"
	"errors"
	"testing"
)

func TestGetUSBSoundCards_DefaultScenario(t *testing.T) {
	installFakeBin(t)
	fakeSysfs(t, "1")
	cfg := testConfig(t)

	cards, err := GetUSBSoundCards(context.Background(), newTestExecutor(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 USB card, got %d: %+v", len(cards), cards)
	}

	c := cards[0]
	checks := map[string]struct{ got, want string }{
		"CardNumber":   {c.CardNumber, "1"},
		"VendorID":     {c.VendorID, "1234"},
		"ProductID":    {c.ProductID, "5678"},
		"Serial":       {c.Serial, "SN123456ABC"},
		"PhysicalPort": {c.PhysicalPort, "1-2.3"},
		"Vendor":       {c.Vendor, "Focusrite-Novation"},
		"Product":      {c.Product, "Scarlett 2i2"},
		"FriendlyName": {c.FriendlyName, "usb_1234_5678_SN123456ABC"},
	}
	for field, cv := range checks {
		if cv.got != cv.want {
			t.Errorf("%s = %q, want %q", field, cv.got, cv.want)
		}
	}
	if c.IsVirtual {
		t.Error("expected non-virtual device")
	}
	if err := c.Validate(); err != nil {
		t.Errorf("detected card should validate, got: %v", err)
	}
}

func TestGetUSBSoundCards_NoUSBCards(t *testing.T) {
	scenario := installFakeBin(t)
	fakeSysfs(t, "0")
	// Only a non-USB onboard card is present.
	scenarioFile(t, scenario, "aplay_l.txt",
		"**** List of PLAYBACK Hardware Devices ****\n"+
			"card 0: PCH [HDA Intel PCH], device 0: ALC892 Analog [ALC892 Analog]\n")

	_, err := GetUSBSoundCards(context.Background(), newTestExecutor(), testConfig(t))
	if !errors.Is(err, ErrNoUSBSoundCards) {
		t.Fatalf("expected ErrNoUSBSoundCards, got %v", err)
	}
}

func TestGetUSBSoundCards_DeviceDisconnected(t *testing.T) {
	installFakeBin(t)
	// aplay reports card1, but sysfs has no card1 -> disconnected mid-detection.
	fakeSysfs(t)

	cards, err := GetUSBSoundCards(context.Background(), newTestExecutor(), testConfig(t))
	if err == nil {
		t.Fatalf("expected error when card path is missing, got cards=%+v", cards)
	}
	if len(cards) != 0 {
		t.Errorf("expected no cards, got %d", len(cards))
	}
}

func TestGetUSBSoundCards_VirtualIgnored(t *testing.T) {
	scenario := installFakeBin(t)
	fakeSysfs(t, "1")
	scenarioFile(t, scenario, "udevadm_attr_walk.txt",
		`    KERNELS=="1-2.3"`+"\n"+
			`    DRIVERS=="snd_aloop"`+"\n"+
			`    ATTRS{idVendor}=="1234"`+"\n"+
			`    ATTRS{idProduct}=="5678"`+"\n")

	cfg := testConfig(t)
	cfg.IgnoreVirtual = true

	_, err := GetUSBSoundCards(context.Background(), newTestExecutor(), cfg)
	if !errors.Is(err, ErrNoUSBSoundCards) {
		t.Fatalf("expected virtual device to be skipped -> ErrNoUSBSoundCards, got %v", err)
	}
}

func TestGetUSBSoundCards_VirtualKeptWhenNotIgnored(t *testing.T) {
	scenario := installFakeBin(t)
	fakeSysfs(t, "1")
	scenarioFile(t, scenario, "udevadm_attr_walk.txt",
		`    KERNELS=="1-2.3"`+"\n"+
			`    DRIVERS=="snd_aloop"`+"\n"+
			`    ATTRS{idVendor}=="1234"`+"\n"+
			`    ATTRS{idProduct}=="5678"`+"\n")

	cards, err := GetUSBSoundCards(context.Background(), newTestExecutor(), testConfig(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cards) != 1 || !cards[0].IsVirtual {
		t.Fatalf("expected 1 virtual card, got %+v", cards)
	}
}

func TestGetCardDetails_PopulatesAllFields(t *testing.T) {
	installFakeBin(t)
	fakeSysfs(t, "1")

	card, err := getCardDetails(context.Background(), newTestExecutor(), "1", testConfig(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.VendorID != "1234" || card.ProductID != "5678" {
		t.Errorf("VID:PID = %s:%s, want 1234:5678", card.VendorID, card.ProductID)
	}
	if card.BusID != "1" {
		t.Errorf("BusID = %q, want 1 (from physical port 1-2.3)", card.BusID)
	}
	if card.DevicePath != "/dev/snd/card1" {
		t.Errorf("DevicePath = %q", card.DevicePath)
	}
}

func TestGetCardDetails_InsufficientInfo(t *testing.T) {
	scenario := installFakeBin(t)
	fakeSysfs(t, "1")
	// Attribute walk without idVendor/idProduct.
	scenarioFile(t, scenario, "udevadm_attr_walk.txt", `    KERNELS=="1-2.3"`+"\n")

	_, err := getCardDetails(context.Background(), newTestExecutor(), "1", testConfig(t))
	if err == nil {
		t.Fatal("expected error for card lacking vendor/product IDs")
	}
}

func TestFindAllUSBDevices(t *testing.T) {
	installFakeBin(t)

	devices, err := findAllUSBDevices(context.Background(), newTestExecutor())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bus 001 Device 004 -> key "1:4" after leading-zero trimming.
	dev, ok := devices["1:4"]
	if !ok {
		t.Fatalf("expected device key 1:4, got keys %v", devices)
	}
	if dev["vendorID"] != "1234" || dev["productID"] != "5678" {
		t.Errorf("device 1:4 = %s:%s, want 1234:5678", dev["vendorID"], dev["productID"])
	}
}
