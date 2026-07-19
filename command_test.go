// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestExecutor() *CommandExecutor {
	rt := NewResourceTracker()
	return &CommandExecutor{
		DefaultTimeout:  5 * time.Second,
		MaxRetries:      0,
		RetryInterval:   100 * time.Millisecond,
		ResourceTracker: rt,
	}
}

func TestValidateCommandArgs_SafeCommand(t *testing.T) {
	err := validateCommandArgs("ls")
	if err != nil {
		t.Fatalf("expected no error for safe command, got %v", err)
	}
}

func TestValidateCommandArgs_UnsafeCommand(t *testing.T) {
	err := validateCommandArgs("ls;rm")
	if err == nil {
		t.Fatal("expected error for unsafe command name")
	}
	if !errors.Is(err, ErrUnsafeArgument) {
		t.Errorf("expected ErrUnsafeArgument, got %v", err)
	}
}

func TestValidateCommandArgs_SafePathArg(t *testing.T) {
	err := validateCommandArgs("cat", "/etc/udev/rules.d/test.rules")
	if err != nil {
		t.Fatalf("expected no error for safe path arg, got %v", err)
	}
}

func TestValidateCommandArgs_UnsafePathArg(t *testing.T) {
	err := validateCommandArgs("cat", "/etc/udev/rules.d/test rules")
	if err == nil {
		t.Fatal("expected error for path with space")
	}
}

func TestValidateCommandArgs_ShellInjection(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"double ampersand", []string{"--flag", "value && rm -rf /"}},
		{"double pipe", []string{"--flag", "value || evil"}},
		{"semicolon", []string{"--flag", "value; evil"}},
		{"backtick", []string{"--flag", "`evil`"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommandArgs("echo", tt.args...)
			if err == nil {
				t.Fatal("expected error for shell injection attempt")
			}
		})
	}
}

func TestValidateCommandArgs_SafeNonPathArgs(t *testing.T) {
	err := validateCommandArgs("udevadm", "info", "--attribute-walk", "--path")
	if err != nil {
		t.Fatalf("expected no error for safe args, got %v", err)
	}
}

func TestCommandExecutor_ExecuteCommand_Echo(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	output, err := executor.ExecuteCommand(ctx, "echo", "hello")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output != "hello\n" {
		t.Errorf("expected 'hello\\n', got %q", output)
	}
}

func TestCommandExecutor_ExecuteCommand_NotFound(t *testing.T) {
	executor := newTestExecutor()
	ctx := context.Background()

	_, err := executor.ExecuteCommand(ctx, "nonexistent_command_12345")
	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}
	if !errors.Is(err, ErrCommandNotFound) {
		t.Errorf("expected ErrCommandNotFound, got %v", err)
	}
}

func TestCommandExecutor_ExecuteCommand_ContextCanceled(t *testing.T) {
	executor := newTestExecutor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := executor.ExecuteCommand(ctx, "echo", "hello")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestCommandExecutor_ExecuteCommand_Timeout(t *testing.T) {
	rt := NewResourceTracker()
	executor := &CommandExecutor{
		DefaultTimeout:  50 * time.Millisecond,
		MaxRetries:      0,
		RetryInterval:   10 * time.Millisecond,
		ResourceTracker: rt,
	}
	ctx := context.Background()

	_, err := executor.ExecuteCommand(ctx, "sleep", "10")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestCommandExecutor_ExecuteCommand_WithRetries(t *testing.T) {
	rt := NewResourceTracker()
	executor := &CommandExecutor{
		DefaultTimeout:  5 * time.Second,
		MaxRetries:      2,
		RetryInterval:   10 * time.Millisecond,
		ResourceTracker: rt,
	}
	ctx := context.Background()

	// This command will fail but should retry
	_, err := executor.ExecuteCommand(ctx, "false")
	if err == nil {
		t.Fatal("expected error from 'false' command")
	}
}

func TestCommandExecutor_NegativeRetriesStillExecutesOnce(t *testing.T) {
	// A negative retry budget must be clamped so the command is still attempted
	// once and the returned error wraps the real failure, not a nil cause.
	rt := NewResourceTracker()
	executor := &CommandExecutor{
		DefaultTimeout:  5 * time.Second,
		MaxRetries:      -3,
		RetryInterval:   10 * time.Millisecond,
		ResourceTracker: rt,
	}

	_, err := executor.ExecuteCommand(context.Background(), "false")
	if err == nil {
		t.Fatal("expected a non-nil error from 'false' with negative retries")
	}
	if strings.Contains(err.Error(), "%!w(<nil>)") {
		t.Errorf("error wraps a nil cause, indicating the command never ran: %v", err)
	}
}

func TestCommandExecutor_ZeroRetriesRunsOnce(t *testing.T) {
	rt := NewResourceTracker()
	executor := &CommandExecutor{
		DefaultTimeout:  5 * time.Second,
		MaxRetries:      0,
		RetryInterval:   10 * time.Millisecond,
		ResourceTracker: rt,
	}

	out, err := executor.ExecuteCommand(context.Background(), "echo", "ok")
	if err != nil {
		t.Fatalf("expected success with zero retries, got %v", err)
	}
	if out != "ok\n" {
		t.Errorf("output = %q, want %q", out, "ok\n")
	}
}

func TestCheckCommands_AllPresent(t *testing.T) {
	// This test only works if the required commands are installed
	// We test with standard system commands that should always be available
	err := CheckCommands()
	// We can't guarantee these are installed in CI, so just verify the function runs
	if err != nil {
		t.Skipf("required commands not found (expected in CI): %v", err)
	}
}
