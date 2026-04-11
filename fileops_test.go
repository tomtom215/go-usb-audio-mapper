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

	err := atomicWriteFile(target, []byte("hello world"), 0644, fa, 2*time.Second)
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

	if info.Mode().Perm() != 0644 {
		t.Errorf("expected permissions 0644, got %o", info.Mode().Perm())
	}
}

func TestAtomicWriteFile_Overwrite(t *testing.T) {
	dir := t.TempDir()
	rt := NewResourceTracker()
	fa := NewSafeFileAccess(rt)
	target := filepath.Join(dir, "testfile.txt")

	// Write initial content
	err := atomicWriteFile(target, []byte("initial"), 0644, fa, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Overwrite
	err = atomicWriteFile(target, []byte("updated"), 0644, fa, 2*time.Second)
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

	err := atomicWriteFile("/tmp/test file", []byte("data"), 0644, fa, 2*time.Second)
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

	err := atomicWriteFile(target, bigData, 0644, fa, 2*time.Second)
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

	fa.LockFile(lockFile1, 2*time.Second)
	fa.LockFile(lockFile2, 2*time.Second)

	// Should not panic or error
	fa.CleanupAllLocks()
}
