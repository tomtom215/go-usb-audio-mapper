package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// CommandExecutor runs system commands safely with retries
type CommandExecutor struct {
	DefaultTimeout  time.Duration
	MaxRetries      int
	RetryInterval   time.Duration
	ResourceTracker *ResourceTracker
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(config Config, resourceTracker *ResourceTracker) *CommandExecutor {
	return &CommandExecutor{
		DefaultTimeout:  config.Timeouts.CommandExecution,
		MaxRetries:      config.MaxRetries,
		RetryInterval:   config.Timeouts.RetryInterval,
		ResourceTracker: resourceTracker,
	}
}

// ExecuteCommand executes a command with the default timeout and retries
func (ce *CommandExecutor) ExecuteCommand(ctx context.Context, command string, args ...string) (string, error) {
	if err := validateCommandArgs(command, args...); err != nil {
		return "", err
	}

	return ce.ExecuteCommandWithTimeoutAndRetry(ctx, ce.DefaultTimeout, ce.MaxRetries, command, args...)
}

// validateCommandArgs checks command and arguments for safety
func validateCommandArgs(command string, args ...string) error {
	if !fileNameRegex.MatchString(command) {
		return fmt.Errorf("unsafe command name: %s: %w", command, ErrUnsafeArgument)
	}

	for i, arg := range args {
		if strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "./") || strings.HasPrefix(arg, "../") {
			if !pathSafeRegex.MatchString(arg) {
				return fmt.Errorf("unsafe path argument at position %d: %s: %w", i, arg, ErrUnsafeArgument)
			}
		} else if strings.Contains(arg, "&&") || strings.Contains(arg, "||") ||
			strings.Contains(arg, ";") || strings.Contains(arg, "`") {
			return fmt.Errorf("potentially unsafe argument at position %d: %s: %w", i, arg, ErrUnsafeArgument)
		}
	}

	return nil
}

// ExecuteCommandWithTimeoutAndRetry executes a command with specific timeout and retries
func (ce *CommandExecutor) ExecuteCommandWithTimeoutAndRetry(
	ctx context.Context,
	timeout time.Duration,
	maxRetries int,
	command string,
	args ...string,
) (string, error) {
	var (
		output     string
		err        error
		retryCount int
	)

	cmdID := fmt.Sprintf("cmd_%s_%d", command, time.Now().UnixNano())

	for retryCount = 0; retryCount <= maxRetries; retryCount++ {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		output, err = ce.executeCommandOnce(ctx, timeout, cmdID, command, args...)

		if err == nil ||
			errors.Is(err, ErrCommandNotFound) ||
			errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, ErrUnsafeArgument) {
			return output, err
		}

		if retryCount < maxRetries {
			slog.Debug("Command failed, retrying",
				"command", command,
				"args", args,
				"error", err,
				"retry", retryCount+1)

			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(ce.RetryInterval):
			}
		}
	}

	return output, fmt.Errorf("command failed after %d retries: %w", retryCount-1, err)
}

// executeCommandOnce executes a command once with timeout
func (ce *CommandExecutor) executeCommandOnce(
	ctx context.Context,
	timeout time.Duration,
	cmdID string,
	command string,
	args ...string,
) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)

	ce.ResourceTracker.AddResource(cmdID, func() error {
		cancel()
		return nil
	})

	cmdPath, err := exec.LookPath(command)
	if err != nil {
		ce.ResourceTracker.ReleaseResource(cmdID)
		return "", fmt.Errorf("command not found %s: %w", command, ErrCommandNotFound)
	}

	slog.Debug("Executing command", "command", command, "args", args)
	cmd := exec.CommandContext(execCtx, cmdPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	processID := fmt.Sprintf("process_%s_%d", command, time.Now().UnixNano())
	ce.ResourceTracker.AddResource(processID, func() error {
		if cmd.Process != nil {
			err := cmd.Process.Kill()
			if err != nil && !strings.Contains(err.Error(), "process already finished") {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
		return nil
	})

	err = cmd.Run()

	ce.ResourceTracker.ReleaseResource(cmdID)
	ce.ResourceTracker.ReleaseResource(processID)

	if execCtx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after %s: %s %v", timeout, command, args)
	}

	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return "", fmt.Errorf("command '%s %v' failed with exit code %d: %s",
				command, args, exitError.ExitCode(), stderr.String())
		}
		return "", fmt.Errorf("command '%s %v' failed: %s", command, args, stderr.String())
	}

	return stdout.String(), nil
}

// CheckCommands verifies that all required system commands are available
func CheckCommands() error {
	requiredCommands := []string{"lsusb", "aplay", "udevadm"}

	var missingCommands []string
	for _, cmd := range requiredCommands {
		_, err := exec.LookPath(cmd)
		if err != nil {
			missingCommands = append(missingCommands, cmd)
		}
	}

	if len(missingCommands) > 0 {
		return fmt.Errorf("required commands not found: %s: %w", strings.Join(missingCommands, ", "), ErrCommandNotFound)
	}

	return nil
}
