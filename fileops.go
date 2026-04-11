// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// SafeFileAccess ensures thread-safe file operations
type SafeFileAccess struct {
	lockMap map[string]*flock.Flock
	mu      sync.Mutex
	tracker *ResourceTracker
}

// NewSafeFileAccess creates a new file access manager
func NewSafeFileAccess(tracker *ResourceTracker) *SafeFileAccess {
	return &SafeFileAccess{
		lockMap: make(map[string]*flock.Flock),
		tracker: tracker,
	}
}

// LockFile acquires a lock on a file with timeout
func (sfa *SafeFileAccess) LockFile(filePath string, timeout time.Duration) (*flock.Flock, error) {
	if !pathSafeRegex.MatchString(filePath) {
		return nil, fmt.Errorf("unsafe file path: %s: %w", filePath, ErrInvalidPath)
	}

	sfa.mu.Lock()
	defer sfa.mu.Unlock()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	lock, exists := sfa.lockMap[absPath]
	if !exists {
		lock = flock.New(absPath)
		sfa.lockMap[absPath] = lock
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	lockID := fmt.Sprintf("filelock_%s", absPath)

	sfa.tracker.AddResource(lockID, func() error {
		if lock.Locked() {
			return lock.Unlock()
		}
		return nil
	})

	success, err := lock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		_ = sfa.tracker.ReleaseResource(lockID)
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !success {
		_ = sfa.tracker.ReleaseResource(lockID)
		return nil, ErrFileLockFailed
	}

	return lock, nil
}

// UnlockFile releases a lock on a file
func (sfa *SafeFileAccess) UnlockFile(filePath string) error {
	if !pathSafeRegex.MatchString(filePath) {
		return fmt.Errorf("unsafe file path: %s: %w", filePath, ErrInvalidPath)
	}

	sfa.mu.Lock()
	defer sfa.mu.Unlock()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	lock, exists := sfa.lockMap[absPath]
	if !exists {
		return nil
	}

	err = lock.Unlock()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	delete(sfa.lockMap, absPath)

	lockID := fmt.Sprintf("filelock_%s", absPath)
	_ = sfa.tracker.ReleaseResource(lockID)

	return nil
}

// CleanupAllLocks releases all locks
func (sfa *SafeFileAccess) CleanupAllLocks() {
	sfa.mu.Lock()
	defer sfa.mu.Unlock()

	for path, lock := range sfa.lockMap {
		if lock.Locked() {
			err := lock.Unlock()
			if err != nil {
				slog.Error("Failed to release lock during cleanup",
					"path", path, "error", err)
			}

			lockID := fmt.Sprintf("filelock_%s", path)
			_ = sfa.tracker.ReleaseResource(lockID)
		}
	}

	sfa.lockMap = make(map[string]*flock.Flock)
}

// atomicWriteFile writes a file atomically using a temporary file and rename
func atomicWriteFile(filename string, data []byte, perm fs.FileMode, fileAccess *SafeFileAccess, lockTimeout time.Duration) error { //nolint:unparam // perm is configurable by design
	if !pathSafeRegex.MatchString(filename) {
		return fmt.Errorf("unsafe file path: %s: %w", filename, ErrInvalidPath)
	}

	if int64(len(data)) > maxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size (%d bytes): %w",
			maxFileSize, ErrResourceExhausted)
	}

	dir := filepath.Dir(filename)

	_, err := fileAccess.LockFile(filename, lockTimeout)
	if err != nil {
		return fmt.Errorf("failed to acquire lock on file: %w", err)
	}
	defer func() { _ = fileAccess.UnlockFile(filename) }()

	tempFile, err := os.CreateTemp(dir, filepath.Base(filename)+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()

	success := false
	defer func() {
		if !success {
			err := os.Remove(tempPath)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				slog.Error("Failed to remove temporary file during cleanup",
					"path", tempPath, "error", err)
			}
		}
	}()

	if _, err = tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	if err = tempFile.Chmod(perm); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to chmod temporary file: %w", err)
	}

	if err = tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err = os.Rename(tempPath, filename); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	success = true
	return nil
}

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) (bool, error) {
	if !pathSafeRegex.MatchString(filename) {
		return false, fmt.Errorf("unsafe file path: %s: %w", filename, ErrInvalidPath)
	}

	info, err := os.Stat(filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("error checking file existence: %w", err)
	}
	return !info.IsDir(), nil
}

// directoryExists checks if a directory exists
func directoryExists(path string) (bool, error) {
	if !pathSafeRegex.MatchString(path) {
		return false, fmt.Errorf("unsafe directory path: %s: %w", path, ErrInvalidPath)
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("error checking directory existence: %w", err)
	}
	return info.IsDir(), nil
}

// pathExists checks if a path exists (file or directory)
func pathExists(path string) (bool, error) {
	if !pathSafeRegex.MatchString(path) {
		return false, fmt.Errorf("unsafe path: %s: %w", path, ErrInvalidPath)
	}

	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("error checking path existence: %w", err)
	}
	return true, nil
}
