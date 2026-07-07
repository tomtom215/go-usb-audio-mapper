// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestNewCommandExecutor(t *testing.T) {
	cfg := &Config{
		Timeouts:   ConfigurableTimeouts{CommandExecution: 3 * time.Second, RetryInterval: 250 * time.Millisecond},
		MaxRetries: 4,
	}
	rt := NewResourceTracker()
	ce := NewCommandExecutor(cfg, rt)

	if ce.DefaultTimeout != 3*time.Second {
		t.Errorf("DefaultTimeout = %v, want 3s", ce.DefaultTimeout)
	}
	if ce.MaxRetries != 4 {
		t.Errorf("MaxRetries = %d, want 4", ce.MaxRetries)
	}
	if ce.ResourceTracker != rt {
		t.Error("ResourceTracker not wired through")
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns what was
// written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	_ = w.Close()

	buf := make([]byte, 64*1024)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func TestShowDeviceList(t *testing.T) {
	// Empty list path.
	if out := captureStdout(t, func() { showDeviceList(nil) }); !strings.Contains(out, "No USB sound cards found") {
		t.Errorf("empty list output = %q", out)
	}

	card := sampleUSBCard()
	virtual := sampleUSBCard()
	virtual.CardNumber = "2"
	virtual.IsVirtual = true
	virtual.Serial = ""
	virtual.PhysicalPort = ""
	virtual.ValidationErr = os.ErrInvalid

	out := captureStdout(t, func() { showDeviceList([]USBSoundCard{card, virtual}) })
	for _, want := range []string{"Scarlett 2i2", "1234:5678", "Serial: SN123456ABC", "Virtual Device", "Suggested Name"} {
		if !strings.Contains(out, want) {
			t.Errorf("device list missing %q in:\n%s", want, out)
		}
	}
}

func TestSetupSignalHandling_CtxDoneReturns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := &Config{Timeouts: ConfigurableTimeouts{GracefulShutdown: time.Second}}
	setupSignalHandling(ctx, cancel, NewResourceTracker(), cfg)
	defer signal.Reset(os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGABRT)

	// Canceling the context makes the watcher goroutine take its <-ctx.Done()
	// branch and return without touching signals.
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestSetupSignalHandling_SignalRunsCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer signal.Reset(os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGABRT)

	rt := NewResourceTracker()
	cleaned := make(chan struct{})
	rt.AddResource("probe", func() error { close(cleaned); return nil })

	// A generous graceful timeout guarantees the cleanup path wins the race
	// against the force-exit timer, so os.Exit is never reached in the test.
	cfg := &Config{Timeouts: ConfigurableTimeouts{GracefulShutdown: 10 * time.Second}}
	setupSignalHandling(ctx, cancel, rt, cfg)

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM to self: %v", err)
	}

	select {
	case <-cleaned:
		// Signal handler invoked CleanupAll -> signal path exercised.
	case <-time.After(3 * time.Second):
		t.Fatal("signal handler did not run resource cleanup")
	}
}
