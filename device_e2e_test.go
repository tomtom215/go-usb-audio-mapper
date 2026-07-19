// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

// End-to-end detection tests that exercise the real command-execution and
// parsing pipeline against fake lsusb/aplay/udevadm binaries — no hardware.

package main

import (
	"context"
	"errors"
	"strings"
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

func TestGetUSBSoundCards_MultipleDistinctCards(t *testing.T) {
	// A field deployment often has several USB recorders attached at once. Each
	// must be enumerated and named from its own attributes without collision.
	scenario := installFakeBin(t)
	fakeSysfs(t, "1", "2")
	scenarioFile(t, scenario, "aplay_l.txt",
		"**** List of PLAYBACK Hardware Devices ****\n"+
			"card 0: PCH [HDA Intel PCH], device 0: ALC892 Analog [ALC892 Analog]\n"+
			"card 1: Device [USB Audio Device], device 0: USB Audio [USB Audio]\n"+
			"card 2: Recorder [USB Audio Recorder], device 0: USB Audio [USB Audio]\n")
	scenarioFile(t, scenario, "udevadm_attr_walk_card1.txt",
		`    KERNELS=="1-2.3"`+"\n"+
			`    DRIVERS=="snd_usb_audio"`+"\n"+
			`    ATTRS{idVendor}=="1234"`+"\n"+
			`    ATTRS{idProduct}=="5678"`+"\n"+
			`    ATTRS{serial}=="SN0001"`+"\n")
	scenarioFile(t, scenario, "udevadm_attr_walk_card2.txt",
		`    KERNELS=="3-1"`+"\n"+
			`    DRIVERS=="snd_usb_audio"`+"\n"+
			`    ATTRS{idVendor}=="abcd"`+"\n"+
			`    ATTRS{idProduct}=="ef01"`+"\n"+
			`    ATTRS{serial}=="SN0002"`+"\n")

	cards, err := GetUSBSoundCards(context.Background(), newTestExecutor(), testConfig(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("expected 2 USB cards, got %d: %+v", len(cards), cards)
	}

	byVID := map[string]USBSoundCard{}
	for _, c := range cards {
		byVID[c.VendorID] = c
		if err := c.Validate(); err != nil {
			t.Errorf("card %s failed validation: %v", c.CardNumber, err)
		}
	}

	c1, ok1 := byVID["1234"]
	c2, ok2 := byVID["abcd"]
	if !ok1 || !ok2 {
		t.Fatalf("expected both VID 1234 and abcd, got %+v", byVID)
	}
	if c1.FriendlyName == c2.FriendlyName {
		t.Errorf("distinct devices produced identical friendly names: %q", c1.FriendlyName)
	}
	if c1.FriendlyName != "usb_1234_5678_SN0001" {
		t.Errorf("card1 FriendlyName = %q", c1.FriendlyName)
	}
	if c2.FriendlyName != "usb_abcd_ef01_SN0002" {
		t.Errorf("card2 FriendlyName = %q", c2.FriendlyName)
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

func TestGetCardDetails_UnsafeSerialUsesPortName(t *testing.T) {
	scenario := installFakeBin(t)
	fakeSysfs(t, "1")
	// A serial containing a backslash survives extraction and sanitization but
	// cannot be a udev match key, so the friendly name must derive from the
	// physical port instead of the serial.
	scenarioFile(t, scenario, "udevadm_attr_walk.txt",
		`    KERNELS=="1-2.3"`+"\n"+
			`    DRIVERS=="snd_usb_audio"`+"\n"+
			`    ATTRS{idVendor}=="1234"`+"\n"+
			`    ATTRS{idProduct}=="5678"`+"\n"+
			"    ATTRS{serial}==\"SN\\123\"\n")

	card, err := getCardDetails(context.Background(), newTestExecutor(), "1", testConfig(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.FriendlyName != "usb_1234_5678_port1_2_3" {
		t.Errorf("FriendlyName = %q, want port-based name for unsafe serial", card.FriendlyName)
	}

	// The generated rule must fall back to the port and stay well-formed.
	rule, err := createUdevRule(context.Background(), &card, "", testConfig(t))
	if err != nil {
		t.Fatalf("createUdevRule: %v", err)
	}
	if strings.Contains(rule.Content, "ATTRS{serial}") {
		t.Errorf("unsafe serial leaked into rule match:\n%s", rule.Content)
	}
	if !strings.Contains(rule.Content, `KERNELS=="1-2.3*"`) {
		t.Errorf("expected port-based matching in rule:\n%s", rule.Content)
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
