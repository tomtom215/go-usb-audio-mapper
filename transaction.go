// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"fmt"
	"log/slog"
	"sync"
)

// Transaction represents an atomic operation with rollback capability
type Transaction struct {
	operations []func() error
	rollbacks  []func() error
	committed  bool
	mu         sync.Mutex
}

// NewTransaction creates a new transaction
func NewTransaction() *Transaction {
	return &Transaction{
		operations: make([]func() error, 0),
		rollbacks:  make([]func() error, 0),
		committed:  false,
	}
}

// AddOperation adds an operation and its rollback function to the transaction
func (t *Transaction) AddOperation(operation, rollback func() error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.operations = append(t.operations, operation)
	t.rollbacks = append(t.rollbacks, rollback)
}

// Execute executes all operations in the transaction
func (t *Transaction) Execute() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, operation := range t.operations {
		if err := operation(); err != nil {
			slog.Error("Transaction operation failed, rolling back", "error", err, "operation", i)

			// Execute rollbacks in reverse order
			for j := i - 1; j >= 0; j-- {
				if rollbackErr := t.rollbacks[j](); rollbackErr != nil {
					slog.Error("Rollback failed", "error", rollbackErr, "operation", j)
				}
			}

			return fmt.Errorf("transaction failed at operation %d: %w", i, err)
		}
	}

	t.committed = true
	return nil
}

// Commit marks the transaction as committed
func (t *Transaction) Commit() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.committed = true
}

// Rollback executes all rollback functions in reverse order
func (t *Transaction) Rollback() []error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.committed {
		return nil
	}

	var errs []error
	for i := len(t.rollbacks) - 1; i >= 0; i-- {
		if err := t.rollbacks[i](); err != nil {
			errs = append(errs, fmt.Errorf("rollback %d failed: %w", i, err))
		}
	}

	return errs
}
