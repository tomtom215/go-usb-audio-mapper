// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

// End-to-end tests for sound-system detection and PCI-serial fallback probing.

package main

import (
	"context"
	"testing"
)

func TestDetectSoundSystemType(t *testing.T) {
	tests := []struct {
		name        string
		presentFile string
		want        string
	}{
		{"defaults to alsa", "", "alsa"},
		{"pipewire", "pipewire_present", "pipewire"},
		{"pulseaudio", "pulseaudio_present", "pulseaudio"},
		{"jack", "jackd_present", "jack"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := installFakeBin(t)
			if tt.presentFile != "" {
				scenarioFile(t, scenario, tt.presentFile, "1\n")
			}
			got := detectSoundSystemType(context.Background(), newTestExecutor())
			if got != tt.want {
				t.Errorf("detectSoundSystemType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckPCIFallbackForSerials_True(t *testing.T) {
	installFakeBin(t)
	// Default lsusb -v output contains an "iSerial" line and a ":".
	has, err := checkPCIFallbackForSerials(context.Background(), newTestExecutor())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Error("expected PCI-like serial detection to be true for default fixture")
	}
}

func TestCheckPCIFallbackForSerials_False(t *testing.T) {
	scenario := installFakeBin(t)
	// Verbose output without an iSerial descriptor.
	scenarioFile(t, scenario, "lsusb_v.txt",
		"Bus 001 Device 004: ID 1234:5678 Focusrite-Novation Scarlett 2i2\n")

	has, err := checkPCIFallbackForSerials(context.Background(), newTestExecutor())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Error("expected no PCI-like serials when iSerial is absent")
	}
}
