// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"errors"
	"testing"
)

func TestTransaction_ExecuteSuccess(t *testing.T) {
	txn := NewTransaction()

	executed := []int{}
	txn.AddOperation(
		func() error { executed = append(executed, 1); return nil },
		func() error { return nil },
	)
	txn.AddOperation(
		func() error { executed = append(executed, 2); return nil },
		func() error { return nil },
	)

	err := txn.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(executed) != 2 || executed[0] != 1 || executed[1] != 2 {
		t.Errorf("expected operations [1, 2], got %v", executed)
	}
}

func TestTransaction_ExecuteFailureAndRollback(t *testing.T) {
	txn := NewTransaction()

	rollbackOrder := []int{}

	txn.AddOperation(
		func() error { return nil },
		func() error { rollbackOrder = append(rollbackOrder, 1); return nil },
	)
	txn.AddOperation(
		func() error { return nil },
		func() error { rollbackOrder = append(rollbackOrder, 2); return nil },
	)
	txn.AddOperation(
		func() error { return errors.New("operation 3 failed") },
		func() error { rollbackOrder = append(rollbackOrder, 3); return nil },
	)

	err := txn.Execute()
	if err == nil {
		t.Fatal("expected error from failed operation")
	}

	// Rollback should be in reverse order: 2, 1 (not 3, since op 3 failed)
	if len(rollbackOrder) != 2 {
		t.Fatalf("expected 2 rollbacks, got %d: %v", len(rollbackOrder), rollbackOrder)
	}
	if rollbackOrder[0] != 2 || rollbackOrder[1] != 1 {
		t.Errorf("expected rollback order [2, 1], got %v", rollbackOrder)
	}
}

func TestTransaction_Commit(t *testing.T) {
	txn := NewTransaction()

	txn.AddOperation(
		func() error { return nil },
		func() error { return nil },
	)

	err := txn.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	txn.Commit()

	// Rollback after commit should do nothing
	errs := txn.Rollback()
	if len(errs) != 0 {
		t.Errorf("expected no rollback errors after commit, got %v", errs)
	}
}

func TestTransaction_RollbackWithoutCommit(t *testing.T) {
	txn := NewTransaction()

	rolledBack := false
	txn.AddOperation(
		func() error { return nil },
		func() error { rolledBack = true; return nil },
	)

	// Rollback without Execute or Commit
	errs := txn.Rollback()
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}

	if !rolledBack {
		t.Fatal("expected rollback to be called")
	}
}

func TestTransaction_RollbackErrors(t *testing.T) {
	txn := NewTransaction()

	txn.AddOperation(
		func() error { return nil },
		func() error { return errors.New("rollback 1 failed") },
	)
	txn.AddOperation(
		func() error { return nil },
		func() error { return errors.New("rollback 2 failed") },
	)

	errs := txn.Rollback()
	if len(errs) != 2 {
		t.Fatalf("expected 2 rollback errors, got %d", len(errs))
	}
}

func TestTransaction_EmptyTransaction(t *testing.T) {
	txn := NewTransaction()

	err := txn.Execute()
	if err != nil {
		t.Fatalf("expected no error for empty transaction, got %v", err)
	}
}

func TestTransaction_FirstOperationFails(t *testing.T) {
	txn := NewTransaction()

	txn.AddOperation(
		func() error { return errors.New("first op failed") },
		func() error { return nil },
	)
	txn.AddOperation(
		func() error { t.Fatal("second operation should not run"); return nil },
		func() error { return nil },
	)

	err := txn.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
}
