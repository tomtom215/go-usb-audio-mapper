// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileExists_ExistingFile(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	exists, err := fileExists(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected file to exist")
	}
}

func TestFileExists_NonExistent(t *testing.T) {
	exists, err := fileExists(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected file not to exist")
	}
}

func TestFileExists_Directory(t *testing.T) {
	dir := t.TempDir()

	exists, err := fileExists(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected directory to not be reported as file")
	}
}

func TestFileExists_UnsafePath(t *testing.T) {
	_, err := fileExists("/tmp/test file with spaces")
	if err == nil {
		t.Fatal("expected error for unsafe path")
	}
}

func TestDirectoryExists_ExistingDir(t *testing.T) {
	dir := t.TempDir()

	exists, err := directoryExists(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected directory to exist")
	}
}

func TestDirectoryExists_NonExistent(t *testing.T) {
	exists, err := directoryExists(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected directory not to exist")
	}
}

func TestDirectoryExists_FileNotDir(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	exists, err := directoryExists(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected file not to be reported as directory")
	}
}

func TestPathExists_File(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	exists, err := pathExists(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected path to exist")
	}
}

func TestPathExists_Dir(t *testing.T) {
	dir := t.TempDir()

	exists, err := pathExists(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected path to exist")
	}
}

func TestPathExists_NonExistent(t *testing.T) {
	exists, err := pathExists(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected path not to exist")
	}
}

func TestAtomicWriteFile_Success(t *testing.T) {
	dir := t.TempDir()
	rt := NewResourceTracker()
	fa := NewSafeFileAccess(rt)
	target := filepath.Join(dir, "testfile.txt")

	err := atomicWriteFile(target, []byte("hello world"), 0o644, fa, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(content))
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	if info.Mode().Perm() != 0o644 {
		t.Errorf("expected permissions 0o644, got %o", info.Mode().Perm())
	}
}

func TestAtomicWriteFile_Overwrite(t *testing.T) {
	dir := t.TempDir()
	rt := NewResourceTracker()
	fa := NewSafeFileAccess(rt)
	target := filepath.Join(dir, "testfile.txt")

	// Write initial content
	err := atomicWriteFile(target, []byte("initial"), 0o644, fa, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Overwrite
	err = atomicWriteFile(target, []byte("updated"), 0o644, fa, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != "updated" {
		t.Errorf("expected 'updated', got %q", string(content))
	}
}

func TestAtomicWriteFile_UnsafePath(t *testing.T) {
	rt := NewResourceTracker()
	fa := NewSafeFileAccess(rt)

	err := atomicWriteFile("/tmp/test file", []byte("data"), 0o644, fa, 2*time.Second)
	if err == nil {
		t.Fatal("expected error for unsafe path")
	}
}

func TestAtomicWriteFile_ExceedsMaxSize(t *testing.T) {
	dir := t.TempDir()
	rt := NewResourceTracker()
	fa := NewSafeFileAccess(rt)
	target := filepath.Join(dir, "bigfile.txt")

	// Create data larger than maxFileSize
	bigData := make([]byte, maxFileSize+1)

	err := atomicWriteFile(target, bigData, 0o644, fa, 2*time.Second)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
}

func TestSafeFileAccess_LockAndUnlock(t *testing.T) {
	dir := t.TempDir()
	rt := NewResourceTracker()
	fa := NewSafeFileAccess(rt)

	lockFile := filepath.Join(dir, "test.lock")

	lock, err := fa.LockFile(lockFile, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	if lock == nil {
		t.Fatal("expected non-nil lock")
	}

	err = fa.UnlockFile(lockFile)
	if err != nil {
		t.Fatalf("failed to unlock: %v", err)
	}
}

func TestSafeFileAccess_LockUnsafePath(t *testing.T) {
	rt := NewResourceTracker()
	fa := NewSafeFileAccess(rt)

	_, err := fa.LockFile("/tmp/bad path", 2*time.Second)
	if err == nil {
		t.Fatal("expected error for unsafe path")
	}
}

func TestSafeFileAccess_UnlockNonexistent(t *testing.T) {
	rt := NewResourceTracker()
	fa := NewSafeFileAccess(rt)

	err := fa.UnlockFile(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("expected no error unlocking non-locked file, got %v", err)
	}
}

func TestSafeFileAccess_CleanupAllLocks(t *testing.T) {
	dir := t.TempDir()
	rt := NewResourceTracker()
	fa := NewSafeFileAccess(rt)

	lockFile1 := filepath.Join(dir, "test1.lock")
	lockFile2 := filepath.Join(dir, "test2.lock")

	_, _ = fa.LockFile(lockFile1, 2*time.Second)
	_, _ = fa.LockFile(lockFile2, 2*time.Second)

	// Should not panic or error
	fa.CleanupAllLocks()
}

func TestUniqueTimestampedPath_AvoidsCollisions(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "89-usb-soundcard-1234-5678.rules.bak.")
	const ts = "20260719120000"

	// First call: nothing exists yet, so the plain timestamped path is returned.
	p1 := uniqueTimestampedPath(base, ts)
	if p1 != base+ts {
		t.Fatalf("first path = %q, want %q", p1, base+ts)
	}
	if err := os.WriteFile(p1, []byte("one"), 0o644); err != nil {
		t.Fatalf("seed p1: %v", err)
	}

	// Second call with the same timestamp must not collide with p1.
	p2 := uniqueTimestampedPath(base, ts)
	if p2 == p1 {
		t.Fatalf("second path collided with first: %q", p2)
	}
	if p2 != base+ts+"_1" {
		t.Fatalf("second path = %q, want %q", p2, base+ts+"_1")
	}
	if err := os.WriteFile(p2, []byte("two"), 0o644); err != nil {
		t.Fatalf("seed p2: %v", err)
	}

	// Third call increments again, so no existing backup is ever overwritten.
	p3 := uniqueTimestampedPath(base, ts)
	if p3 != base+ts+"_2" {
		t.Fatalf("third path = %q, want %q", p3, base+ts+"_2")
	}

	// The first two backups are still intact and distinct.
	if b, _ := os.ReadFile(p1); string(b) != "one" {
		t.Errorf("p1 content changed: %q", b)
	}
	if b, _ := os.ReadFile(p2); string(b) != "two" {
		t.Errorf("p2 content changed: %q", b)
	}
}
