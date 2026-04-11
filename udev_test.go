// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"context"
	"strings"
	"testing"
)

func TestCreateUdevRule_BasicSerial(t *testing.T) {
	card := USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
		ProductID:  "5678",
		Serial:     "ABC123",
		Vendor:     "TestVendor",
		Product:    "TestProduct",
	}

	config := Config{
		UdevRulesPath: "/etc/udev/rules.d",
		Timeouts:      DefaultTimeouts,
	}

	rule, err := createUdevRule(context.Background(), card, "my_audio", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule == nil {
		t.Fatal("expected non-nil rule")
	}

	// Check rule contains serial-based matching
	if !strings.Contains(rule.Content, `ATTRS{serial}=="ABC123"`) {
		t.Error("expected rule to contain serial matching")
	}

	// Check rule contains the device name
	if !strings.Contains(rule.Content, `ATTR{id}="my_audio"`) {
		t.Error("expected rule to contain device name")
	}

	// Check rule file path
	expectedPath := "/etc/udev/rules.d/89-usb-soundcard-1234-5678.rules"
	if rule.Path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, rule.Path)
	}

	// Check all rule types are present
	if !strings.Contains(rule.Content, `ACTION=="add"`) {
		t.Error("expected add action rule")
	}
	if !strings.Contains(rule.Content, `SOUND_INITIALIZED`) {
		t.Error("expected SOUND_INITIALIZED rule")
	}
	if !strings.Contains(rule.Content, `KERNEL=="card*"`) {
		t.Error("expected kernel card rule")
	}
	if !strings.Contains(rule.Content, `SYMLINK+="sound/by-id/my_audio"`) {
		t.Error("expected symlink rule")
	}
	if !strings.Contains(rule.Content, `SYMLINK+="sound/my_audio/control"`) {
		t.Error("expected control symlink rule")
	}
	if !strings.Contains(rule.Content, `pcm_playback`) {
		t.Error("expected playback symlink rule")
	}
	if !strings.Contains(rule.Content, `pcm_capture`) {
		t.Error("expected capture symlink rule")
	}
}

func TestCreateUdevRule_PhysicalPort(t *testing.T) {
	card := USBSoundCard{
		CardNumber:   "2",
		VendorID:     "abcd",
		ProductID:    "ef01",
		PhysicalPort: "1-2.3",
		Vendor:       "TestVendor",
		Product:      "TestProduct",
		FriendlyName: "default_name",
	}

	config := Config{
		UdevRulesPath: "/etc/udev/rules.d",
		Timeouts:      DefaultTimeouts,
	}

	rule, err := createUdevRule(context.Background(), card, "", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(rule.Content, `KERNELS=="1-2.3*"`) {
		t.Error("expected rule to contain physical port matching")
	}
}

func TestCreateUdevRule_FallbackToVIDPID(t *testing.T) {
	card := USBSoundCard{
		CardNumber:   "3",
		VendorID:     "1111",
		ProductID:    "2222",
		Vendor:       "TestVendor",
		Product:      "TestProduct",
		FriendlyName: "fallback_name",
	}

	config := Config{
		UdevRulesPath: "/etc/udev/rules.d",
		Timeouts:      DefaultTimeouts,
	}

	rule, err := createUdevRule(context.Background(), card, "my_device", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have VID:PID matching without serial or port
	if strings.Contains(rule.Content, "ATTRS{serial}") {
		t.Error("expected no serial matching for device without serial")
	}
	if strings.Contains(rule.Content, "KERNELS==") {
		t.Error("expected no KERNELS matching for device without physical port")
	}
}

func TestCreateUdevRule_MissingVendorID(t *testing.T) {
	card := USBSoundCard{
		CardNumber: "1",
		ProductID:  "5678",
	}

	config := Config{
		UdevRulesPath: "/etc/udev/rules.d",
		Timeouts:      DefaultTimeouts,
	}

	_, err := createUdevRule(context.Background(), card, "test", config)
	if err == nil {
		t.Fatal("expected error for missing vendor ID")
	}
}

func TestCreateUdevRule_MissingProductID(t *testing.T) {
	card := USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
	}

	config := Config{
		UdevRulesPath: "/etc/udev/rules.d",
		Timeouts:      DefaultTimeouts,
	}

	_, err := createUdevRule(context.Background(), card, "test", config)
	if err == nil {
		t.Fatal("expected error for missing product ID")
	}
}

func TestCreateUdevRule_CanceledContext(t *testing.T) {
	card := USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
		ProductID:  "5678",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	config := Config{
		UdevRulesPath: "/etc/udev/rules.d",
		Timeouts:      DefaultTimeouts,
	}

	_, err := createUdevRule(ctx, card, "test", config)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestCreateUdevRule_VirtualDeviceNote(t *testing.T) {
	card := USBSoundCard{
		CardNumber:   "1",
		VendorID:     "1234",
		ProductID:    "5678",
		Vendor:       "Virtual",
		Product:      "Loopback",
		IsVirtual:    true,
		FriendlyName: "virtual_dev",
	}

	config := Config{
		UdevRulesPath: "/etc/udev/rules.d",
		Timeouts:      DefaultTimeouts,
	}

	rule, err := createUdevRule(context.Background(), card, "", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(rule.Content, "virtual audio device") {
		t.Error("expected virtual device note in rule header")
	}
}

func TestCreateUdevRule_CustomNameCleaned(t *testing.T) {
	card := USBSoundCard{
		CardNumber:   "1",
		VendorID:     "1234",
		ProductID:    "5678",
		Vendor:       "Test",
		Product:      "Audio",
		FriendlyName: "default",
	}

	config := Config{
		UdevRulesPath: "/etc/udev/rules.d",
		Timeouts:      DefaultTimeouts,
	}

	rule, err := createUdevRule(context.Background(), card, "my-audio.v2", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Name should be cleaned
	if strings.Contains(rule.Content, "my-audio.v2") {
		t.Error("expected name to be cleaned, but found uncleaned version")
	}
	if !strings.Contains(rule.Content, "my_audio_v2") {
		t.Error("expected cleaned name 'my_audio_v2' in rule")
	}
}

func TestCreateUdevRule_PCILikeSerialUsesPort(t *testing.T) {
	card := USBSoundCard{
		CardNumber:   "1",
		VendorID:     "1234",
		ProductID:    "5678",
		Serial:       "0000:00:1a.0",
		PhysicalPort: "1-2",
		Vendor:       "Test",
		Product:      "Audio",
		FriendlyName: "default",
	}

	config := Config{
		UdevRulesPath: "/etc/udev/rules.d",
		Timeouts:      DefaultTimeouts,
	}

	rule, err := createUdevRule(context.Background(), card, "test_dev", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use KERNELS matching instead of serial (PCI-like serial has colon)
	if strings.Contains(rule.Content, `ATTRS{serial}=="0000:00:1a.0"`) {
		t.Error("expected PCI-like serial to not be used in rule")
	}
	if !strings.Contains(rule.Content, `KERNELS=="1-2*"`) {
		t.Error("expected KERNELS matching with physical port")
	}
}

func TestCreateUdevRule_Header(t *testing.T) {
	card := USBSoundCard{
		CardNumber:   "1",
		VendorID:     "1234",
		ProductID:    "5678",
		Serial:       "SER001",
		PhysicalPort: "1-2",
		Vendor:       "Test",
		Product:      "Audio",
		FriendlyName: "default",
	}

	config := Config{
		UdevRulesPath: "/etc/udev/rules.d",
		Timeouts:      DefaultTimeouts,
	}

	rule, err := createUdevRule(context.Background(), card, "test", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(rule.Content, "usb-soundcard-mapper v"+AppVersion) {
		t.Error("expected version in header")
	}
	if !strings.Contains(rule.Content, "Test Audio") {
		t.Error("expected device name in header")
	}
	if !strings.Contains(rule.Content, "1234:5678") {
		t.Error("expected VID:PID in header")
	}
	if !strings.Contains(rule.Content, "Serial: SER001") {
		t.Error("expected serial in header")
	}
	if !strings.Contains(rule.Content, "USB Path: 1-2") {
		t.Error("expected USB path in header")
	}
}
