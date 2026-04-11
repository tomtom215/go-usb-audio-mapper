// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"fmt"
	"sync"
	"time"
)

// ResourceTracker tracks and manages resources to ensure proper cleanup
type ResourceTracker struct {
	resources map[string]func() error
	mu        sync.Mutex
	wg        sync.WaitGroup
}

// NewResourceTracker creates a new resource tracker
func NewResourceTracker() *ResourceTracker {
	return &ResourceTracker{
		resources: make(map[string]func() error),
	}
}

// AddResource adds a resource with cleanup function
func (rt *ResourceTracker) AddResource(id string, cleanup func() error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.resources[id] = cleanup
}

// ReleaseResource releases a specific resource
func (rt *ResourceTracker) ReleaseResource(id string) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if cleanup, exists := rt.resources[id]; exists {
		err := cleanup()
		delete(rt.resources, id)
		return err
	}

	return nil
}

// CleanupAll releases all tracked resources
func (rt *ResourceTracker) CleanupAll() []error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	var errs []error
	for id, cleanup := range rt.resources {
		if err := cleanup(); err != nil {
			errs = append(errs, fmt.Errorf("failed to clean up resource %s: %w", id, err))
		}
		delete(rt.resources, id)
	}

	return errs
}

// WaitForCompletion waits for all background tasks to complete
func (rt *ResourceTracker) WaitForCompletion(timeout time.Duration) error {
	done := make(chan struct{})

	go func() {
		rt.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for resource cleanup")
	}
}
