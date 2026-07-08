// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

// Tests for the Bubble Tea model. Update and View are exercised directly as
// pure functions over the model state — no terminal or running program.

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestUIModel(t *testing.T) uiModel {
	t.Helper()
	cards := []USBSoundCard{sampleUSBCard()}
	cfg := testConfig(t)
	m := initialUIModel(cards, cfg, newTestExecutor(), newFileAccess(), NewResourceTracker())
	// Give the list a size so its view renders.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return updated.(uiModel)
}

func TestListItem_Rendering(t *testing.T) {
	card := sampleUSBCard()
	card.IsVirtual = true
	card.ValidationErr = errors.New("boom")
	item := listItem{card: &card}

	if !strings.Contains(item.Title(), "Scarlett 2i2") || !strings.Contains(item.Title(), "[Virtual]") {
		t.Errorf("unexpected Title(): %q", item.Title())
	}
	if !strings.Contains(item.Description(), "1234:5678") || !strings.Contains(item.Description(), "boom") {
		t.Errorf("unexpected Description(): %q", item.Description())
	}
	if item.FilterValue() != item.Title() {
		t.Error("FilterValue should equal Title")
	}
}

func TestInitialUIModel(t *testing.T) {
	m := newTestUIModel(t)
	if m.state != stateCardSelect {
		t.Errorf("initial state = %v, want stateCardSelect", m.state)
	}
	if len(m.cards) != 1 {
		t.Errorf("expected 1 card in model, got %d", len(m.cards))
	}
	if m.Init() != nil {
		t.Error("Init() should return nil")
	}
}

func TestUIView_AllStates(t *testing.T) {
	states := []struct {
		state viewState
		want  string
	}{
		{stateCardSelect, "Select a USB sound card"},
		{stateNameInput, "Enter custom name"},
		{stateConfirmation, "confirm"},
		{stateError, "Error"},
		{stateSuccess, "Success"},
	}
	for _, s := range states {
		m := newTestUIModel(t)
		m.state = s.state
		m.selectedCard = sampleUSBCard()
		m.customName = "my_audio"
		m.error = "some error"
		m.successMessage = "done"
		out := m.View()
		if !strings.Contains(out, s.want) {
			t.Errorf("View() in state %v missing %q:\n%s", s.state, s.want, out)
		}
	}
}

func TestUIUpdate_SuccessAndError(t *testing.T) {
	m := newTestUIModel(t)

	updated, _ := m.Update(successMsg{message: "installed"})
	m = updated.(uiModel)
	if m.state != stateSuccess || m.successMessage != "installed" {
		t.Errorf("successMsg did not transition to success state")
	}

	m2 := newTestUIModel(t)
	updated, _ = m2.Update(errMsg{err: errors.New("kaboom")})
	m2 = updated.(uiModel)
	if m2.state != stateError || m2.error != "kaboom" {
		t.Errorf("errMsg did not transition to error state")
	}
}

func TestUIUpdate_CardSelectToNameInput(t *testing.T) {
	m := newTestUIModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(uiModel)
	if m.state != stateNameInput {
		t.Fatalf("Enter on card select should go to name input, got %v", m.state)
	}
	if m.customName != sampleUSBCard().FriendlyName {
		t.Errorf("name input should preload the friendly name, got %q", m.customName)
	}
}

func TestUIUpdate_NameInputValidation(t *testing.T) {
	m := newTestUIModel(t)
	m.state = stateNameInput
	m.textInput.SetValue("")

	// Empty name -> error message, stays on name input.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(uiModel)
	if m.error == "" || m.state != stateNameInput {
		t.Errorf("empty name should set an error and stay on name input")
	}

	// A clean name -> confirmation.
	m.textInput.SetValue("clean_name")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(uiModel)
	if m.state != stateConfirmation || m.customName != "clean_name" {
		t.Errorf("clean name should advance to confirmation, got state %v name %q", m.state, m.customName)
	}
}

func TestUIUpdate_ErrorStateReturnsToSelect(t *testing.T) {
	m := newTestUIModel(t)
	m.state = stateError
	m.error = "boom"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(uiModel)
	if m.state != stateCardSelect || m.error != "" {
		t.Errorf("any key in error state should clear error and return to select")
	}
}

func TestSafelyPerformBackgroundOperation(t *testing.T) {
	m := newTestUIModel(t)

	// Success path.
	msg := m.safelyPerformBackgroundOperation(func() (string, error) {
		return "ok", nil
	})()
	if sm, ok := msg.(successMsg); !ok || sm.message != "ok" {
		t.Errorf("expected successMsg{ok}, got %#v", msg)
	}

	// Error path.
	msg = m.safelyPerformBackgroundOperation(func() (string, error) {
		return "", errors.New("fail")
	})()
	if _, ok := msg.(errMsg); !ok {
		t.Errorf("expected errMsg, got %#v", msg)
	}

	// Closed UI short-circuits to nil.
	m.uiClosed = true
	if got := m.safelyPerformBackgroundOperation(func() (string, error) { return "x", nil })(); got != nil {
		t.Errorf("closed UI should return nil msg, got %#v", got)
	}
	m.uiClosed = false

	// Canceled context yields an errMsg.
	m.cancel()
	if _, ok := m.safelyPerformBackgroundOperation(func() (string, error) { return "x", nil })().(errMsg); !ok {
		t.Errorf("canceled ctx should yield errMsg")
	}
}

func TestRunUI_NoCards(t *testing.T) {
	_, err := runUI(context.Background(), nil, testConfig(t), newTestExecutor(), newFileAccess(), NewResourceTracker())
	if !errors.Is(err, ErrNoUSBSoundCards) {
		t.Fatalf("runUI with no cards should return ErrNoUSBSoundCards, got %v", err)
	}
}
