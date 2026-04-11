// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"errors"
	"testing"
	"time"
)

func TestResourceTracker_AddAndRelease(t *testing.T) {
	rt := NewResourceTracker()

	cleaned := false
	rt.AddResource("test-1", func() error {
		cleaned = true
		return nil
	})

	err := rt.ReleaseResource("test-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !cleaned {
		t.Fatal("expected cleanup function to be called")
	}
}

func TestResourceTracker_ReleaseNonExistent(t *testing.T) {
	rt := NewResourceTracker()

	err := rt.ReleaseResource("nonexistent")
	if err != nil {
		t.Fatalf("expected no error releasing nonexistent resource, got %v", err)
	}
}

func TestResourceTracker_CleanupAll(t *testing.T) {
	rt := NewResourceTracker()

	count := 0
	rt.AddResource("r1", func() error {
		count++
		return nil
	})
	rt.AddResource("r2", func() error {
		count++
		return nil
	})
	rt.AddResource("r3", func() error {
		count++
		return nil
	})

	errs := rt.CleanupAll()
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}

	if count != 3 {
		t.Fatalf("expected 3 cleanups, got %d", count)
	}
}

func TestResourceTracker_CleanupAllWithErrors(t *testing.T) {
	rt := NewResourceTracker()

	rt.AddResource("good", func() error {
		return nil
	})
	rt.AddResource("bad", func() error {
		return errors.New("cleanup failed")
	})

	errs := rt.CleanupAll()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestResourceTracker_WaitForCompletion_NoTasks(t *testing.T) {
	rt := NewResourceTracker()

	err := rt.WaitForCompletion(1 * time.Second)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestResourceTracker_WaitForCompletion_Timeout(t *testing.T) {
	rt := NewResourceTracker()

	rt.wg.Add(1)
	// Never call wg.Done() to simulate stuck task

	err := rt.WaitForCompletion(50 * time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	// Clean up to avoid goroutine leak
	rt.wg.Done()
}

func TestResourceTracker_OverwriteResource(t *testing.T) {
	rt := NewResourceTracker()

	first := false
	second := false

	rt.AddResource("key", func() error {
		first = true
		return nil
	})

	rt.AddResource("key", func() error {
		second = true
		return nil
	})

	rt.CleanupAll()

	if first {
		t.Error("first cleanup should not have been called (overwritten)")
	}
	if !second {
		t.Error("second cleanup should have been called")
	}
}
