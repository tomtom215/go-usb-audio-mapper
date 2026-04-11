// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"log/slog"
	"os"
	"testing"
)

func TestInitLogger_Levels(t *testing.T) {
	levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, "invalid"}

	for _, level := range levels {
		t.Run(string(level), func(t *testing.T) {
			// Should not panic
			initLogger(level)
		})
	}
}

func TestInitLogger_DefaultsToInfo(t *testing.T) {
	initLogger("bogus")

	// Should still have a functioning logger
	slog.Info("test log message")
}

func TestCheckAndFixPermissions_ValidDir(t *testing.T) {
	dir := t.TempDir()

	config := Config{
		UdevRulesPath: dir,
	}

	err := checkAndFixPermissions(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAndFixPermissions_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	newDir := dir + "/newdir"

	config := Config{
		UdevRulesPath: newDir,
	}

	err := checkAndFixPermissions(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory")
	}
}

func TestCheckAndFixPermissions_FileInsteadOfDir(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	config := Config{
		UdevRulesPath: tmpFile.Name(),
	}

	err = checkAndFixPermissions(config)
	if err == nil {
		t.Fatal("expected error when path is a file, not a directory")
	}
}

func TestCheckAndFixPermissions_UnsafePath(t *testing.T) {
	config := Config{
		UdevRulesPath: "/tmp/test path with spaces",
	}

	err := checkAndFixPermissions(config)
	if err == nil {
		t.Fatal("expected error for unsafe path")
	}
}

func TestCheckElevatedPrivileges(t *testing.T) {
	elevated, err := checkElevatedPrivileges()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In CI, might be root or not
	uid := os.Getuid()
	if uid == 0 && !elevated {
		t.Error("expected elevated=true when running as root")
	}
	if uid != 0 && elevated {
		t.Error("expected elevated=false when not running as root")
	}
}
