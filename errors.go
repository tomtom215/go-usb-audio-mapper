package main

import "errors"

// Sentinel errors for specific failure cases
var (
	ErrNoUSBSoundCards      = errors.New("no USB sound cards found")
	ErrInsufficientPrivs    = errors.New("insufficient privileges")
	ErrUdevSystemFailure    = errors.New("udev system test failed")
	ErrCommandNotFound      = errors.New("required command not found")
	ErrOperationCancelled   = errors.New("operation cancelled by user")
	ErrDeviceNameEmpty      = errors.New("device name cannot be empty")
	ErrInvalidDeviceParams  = errors.New("invalid device parameters")
	ErrDeviceDisconnected   = errors.New("device disconnected during operation")
	ErrFileLockFailed       = errors.New("failed to acquire file lock")
	ErrRuleVerificationFail = errors.New("rule verification failed")
	ErrVirtualDevice        = errors.New("virtual audio device detected")
	ErrResourceExhausted    = errors.New("resource limits exhausted")
	ErrInvalidPath          = errors.New("invalid path provided")
	ErrTransactionFailed    = errors.New("transaction failed, rollback completed")
	ErrUnsafeArgument       = errors.New("unsafe command argument detected")
)
