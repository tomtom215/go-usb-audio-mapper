package main

import (
	"testing"
)

func TestUSBSoundCard_String(t *testing.T) {
	card := USBSoundCard{
		CardNumber:   "1",
		Vendor:       "TestVendor",
		Product:      "TestProduct",
		VendorID:     "1234",
		ProductID:    "5678",
		Serial:       "ABC123",
		PhysicalPort: "1-2",
		IsVirtual:    false,
	}

	s := card.String()

	expectedParts := []string{
		"Card: 1",
		"Device: TestVendor TestProduct",
		"VID:PID: 1234:5678",
		"Serial: ABC123",
		"Port: 1-2",
	}

	for _, part := range expectedParts {
		if !contains(s, part) {
			t.Errorf("String() output missing %q, got: %s", part, s)
		}
	}
}

func TestUSBSoundCard_StringVirtual(t *testing.T) {
	card := USBSoundCard{
		CardNumber: "0",
		Vendor:     "Virtual",
		Product:    "Loopback",
		VendorID:   "0000",
		ProductID:  "0001",
		IsVirtual:  true,
	}

	s := card.String()
	if !contains(s, "Type: Virtual") {
		t.Errorf("expected Virtual type in string, got: %s", s)
	}
}

func TestUSBSoundCard_StringNoSerialNoPort(t *testing.T) {
	card := USBSoundCard{
		CardNumber: "2",
		Vendor:     "TestVendor",
		Product:    "TestProduct",
		VendorID:   "abcd",
		ProductID:  "ef01",
	}

	s := card.String()
	if contains(s, "Serial:") {
		t.Errorf("unexpected Serial in string: %s", s)
	}
	if contains(s, "Port:") {
		t.Errorf("unexpected Port in string: %s", s)
	}
}

func TestUSBSoundCard_Validate_Valid(t *testing.T) {
	card := &USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
		ProductID:  "5678",
		Serial:     "ABC123",
	}

	err := card.Validate()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUSBSoundCard_Validate_MissingCardNumber(t *testing.T) {
	card := &USBSoundCard{
		VendorID:  "1234",
		ProductID: "5678",
	}

	err := card.Validate()
	if err == nil {
		t.Fatal("expected error for missing card number")
	}
}

func TestUSBSoundCard_Validate_MissingVendorID(t *testing.T) {
	card := &USBSoundCard{
		CardNumber: "1",
		ProductID:  "5678",
	}

	err := card.Validate()
	if err == nil {
		t.Fatal("expected error for missing vendor ID")
	}
}

func TestUSBSoundCard_Validate_MissingProductID(t *testing.T) {
	card := &USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
	}

	err := card.Validate()
	if err == nil {
		t.Fatal("expected error for missing product ID")
	}
}

func TestUSBSoundCard_Validate_InvalidVendorIDFormat(t *testing.T) {
	card := &USBSoundCard{
		CardNumber: "1",
		VendorID:   "ZZZZ",
		ProductID:  "5678",
	}

	err := card.Validate()
	if err == nil {
		t.Fatal("expected error for invalid vendor ID format")
	}
}

func TestUSBSoundCard_Validate_InvalidProductIDFormat(t *testing.T) {
	card := &USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
		ProductID:  "xyz",
	}

	err := card.Validate()
	if err == nil {
		t.Fatal("expected error for invalid product ID format")
	}
}

func TestUSBSoundCard_Validate_InvalidSerial(t *testing.T) {
	card := &USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
		ProductID:  "5678",
		Serial:     "$(evil)",
	}

	err := card.Validate()
	if err == nil {
		t.Fatal("expected error for invalid serial")
	}
}

func TestUSBSoundCard_Validate_EmptySerial(t *testing.T) {
	card := &USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
		ProductID:  "5678",
		Serial:     "",
	}

	err := card.Validate()
	if err != nil {
		t.Fatalf("expected no error for empty serial, got %v", err)
	}
}

func TestDeviceRegistry_AddAndGetDevices(t *testing.T) {
	reg := NewDeviceRegistry()

	card1 := USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
		ProductID:  "5678",
		Serial:     "SER001",
	}

	card2 := USBSoundCard{
		CardNumber: "2",
		VendorID:   "abcd",
		ProductID:  "ef01",
		Serial:     "SER002",
	}

	reg.AddDevice(card1)
	reg.AddDevice(card2)

	devices := reg.GetDevices()
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
}

func TestDeviceRegistry_GetDevice(t *testing.T) {
	reg := NewDeviceRegistry()

	card := USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
		ProductID:  "5678",
		Serial:     "SER001",
	}

	reg.AddDevice(card)

	retrieved, ok := reg.GetDevice(card)
	if !ok {
		t.Fatal("expected to find device in registry")
	}

	if retrieved.VendorID != card.VendorID {
		t.Errorf("expected VendorID %s, got %s", card.VendorID, retrieved.VendorID)
	}
}

func TestDeviceRegistry_GetDevice_NotFound(t *testing.T) {
	reg := NewDeviceRegistry()

	card := USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
		ProductID:  "5678",
	}

	_, ok := reg.GetDevice(card)
	if ok {
		t.Fatal("expected device not to be found in empty registry")
	}
}

func TestDeviceRegistry_UpdateDeviceStatus(t *testing.T) {
	reg := NewDeviceRegistry()

	card := USBSoundCard{
		CardNumber: "1",
		VendorID:   "1234",
		ProductID:  "5678",
		Serial:     "SER001",
		Status:     DeviceStatusConnected,
	}

	reg.AddDevice(card)
	reg.UpdateDeviceStatus(card, DeviceStatusDisconnected)

	retrieved, ok := reg.GetDevice(card)
	if !ok {
		t.Fatal("expected to find device in registry")
	}

	if retrieved.Status != DeviceStatusDisconnected {
		t.Errorf("expected status Disconnected, got %d", retrieved.Status)
	}
}

func TestDeviceRegistry_GenerateDeviceKey(t *testing.T) {
	reg := NewDeviceRegistry()

	tests := []struct {
		name     string
		card     USBSoundCard
		expected string
	}{
		{
			name: "with serial",
			card: USBSoundCard{
				VendorID:  "1234",
				ProductID: "5678",
				Serial:    "SER001",
			},
			expected: "1234:5678:SER001",
		},
		{
			name: "with PCI-like serial uses port",
			card: USBSoundCard{
				VendorID:     "1234",
				ProductID:    "5678",
				Serial:       "0000:00:1a.0",
				PhysicalPort: "1-2",
			},
			expected: "1234:5678:1-2",
		},
		{
			name: "with physical port",
			card: USBSoundCard{
				VendorID:     "1234",
				ProductID:    "5678",
				PhysicalPort: "1-2",
			},
			expected: "1234:5678:1-2",
		},
		{
			name: "fallback to card number",
			card: USBSoundCard{
				VendorID:   "1234",
				ProductID:  "5678",
				CardNumber: "0",
			},
			expected: "1234:5678:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := reg.generateDeviceKey(tt.card)
			if key != tt.expected {
				t.Errorf("expected key %q, got %q", tt.expected, key)
			}
		})
	}
}

func TestSanitizeSerial(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"clean serial", "ABC123", "ABC123"},
		{"with semicolons", "ABC;123", "ABC_123"},
		{"with pipes", "ABC|123", "ABC_123"},
		{"with ampersands", "ABC&&123", "ABC__123"},
		{"with dollar sign", "ABC$123", "ABC_123"},
		{"with parentheses", "ABC(123)", "ABC_123_"},
		{"with angle brackets", "ABC<123>", "ABC_123_"},
		{"with tabs", "ABC\t123", "ABC_123"},
		{"empty string", "", ""},
		{"normal with dashes", "SER-001-USB", "SER-001-USB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeSerial(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeSerial(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsVirtualDriver(t *testing.T) {
	tests := []struct {
		driver   string
		expected bool
	}{
		{"snd_dummy", true},
		{"snd_aloop", true},
		{"snd_virmidi", true},
		{"snd_pcm_oss", true},
		{"snd_usb_audio", false},
		{"snd_hda_intel", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			result := isVirtualDriver(tt.driver)
			if result != tt.expected {
				t.Errorf("isVirtualDriver(%q) = %v, want %v", tt.driver, result, tt.expected)
			}
		})
	}
}

func TestCleanupName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"alphanumeric", "my_device_1", "my_device_1"},
		{"with spaces", "my device", "my_device"},
		{"with special chars", "my-device.v2", "my_device_v2"},
		{"starts with number", "1234_device", "usb_1234_device"},
		{"empty string", "", ""},
		{"very long name", "abcdefghijklmnopqrstuvwxyz_abcdefghijklmnopqrstuvwxyz_abcdefghijklmnopqrstuvwxyz", "abcdefghijklmnopqrstuvwxyz_abcdefghijklmnopqrstuvwxyz_abcdefghij"},
		{"with slashes", "my/device/path", "my_device_path"},
		{"unicode chars", "device_\u00e9\u00e8", "device___"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanupName(tt.input)
			if result != tt.expected {
				t.Errorf("cleanupName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCleanupName_MaxLength(t *testing.T) {
	longName := ""
	for i := 0; i < 100; i++ {
		longName += "a"
	}

	result := cleanupName(longName)
	if len(result) > 64 {
		t.Errorf("expected max length 64, got %d", len(result))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
