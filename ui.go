// SPDX-License-Identifier: MIT
// Copyright 2025 Tom F. (https://github.com/tomtom215)

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// listItem represents a USB sound card in the UI list
type listItem struct {
	card *USBSoundCard
}

func (i listItem) Title() string {
	title := fmt.Sprintf("Card %s: %s %s", i.card.CardNumber, i.card.Vendor, i.card.Product)
	if i.card.Serial != "" {
		title += fmt.Sprintf(" (S/N: %s)", i.card.Serial)
	}
	if i.card.IsVirtual {
		title += " [Virtual]"
	}
	return title
}

func (i listItem) Description() string {
	desc := fmt.Sprintf("VID:PID %s:%s", i.card.VendorID, i.card.ProductID)
	if i.card.PhysicalPort != "" {
		desc += fmt.Sprintf(", Port: %s", i.card.PhysicalPort)
	}
	if i.card.ValidationErr != nil {
		desc += fmt.Sprintf(" [Warning: %s]", i.card.ValidationErr)
	}
	return desc
}

func (i listItem) FilterValue() string {
	return i.Title()
}

// viewState represents the current UI state
type viewState int

const (
	stateCardSelect viewState = iota
	stateNameInput
	stateConfirmation
	stateError
	stateSuccess
)

// UI styling
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#43BF6D")).
			Padding(0, 1)

	activeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#43BF6D"))

	inactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF0000"))

	warningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFA500"))

	docStyle = lipgloss.NewStyle().
			Margin(1, 2)

	highlightStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#874BFD"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#43BF6D"))
)

// Suppress unused variable warnings for styles used only in views
var _ = subtitleStyle

// UI-related key mappings
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Back    key.Binding
	Quit    key.Binding
	Edit    key.Binding
	Confirm key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "confirm"),
	),
}

// uiModel represents the UI state
type uiModel struct {
	cards           []USBSoundCard
	list            list.Model
	textInput       textinput.Model
	state           viewState
	selectedCard    USBSoundCard
	customName      string
	config          *Config
	executor        *CommandExecutor
	fileAccess      *SafeFileAccess
	error           string
	warning         string
	width           int
	height          int
	successMessage  string
	ctx             context.Context
	cancel          context.CancelFunc
	resourceTracker *ResourceTracker
	operationLock   *sync.Mutex
	uiClosed        bool
}

// initialUIModel creates the initial UI model
func initialUIModel(cards []USBSoundCard, config *Config, executor *CommandExecutor, fileAccess *SafeFileAccess, resourceTracker *ResourceTracker) uiModel {
	ctx, cancel := context.WithCancel(context.Background())

	items := make([]list.Item, len(cards))
	for i := range cards {
		items[i] = listItem{card: &cards[i]}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select USB Sound Card to Map"
	l.SetFilteringEnabled(false)
	l.SetShowHelp(true)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)

	ti := textinput.New()
	ti.Placeholder = "Enter custom name for the device"
	ti.CharLimit = 64
	ti.Width = 40
	ti.Prompt = "› "

	return uiModel{
		cards:           cards,
		list:            l,
		textInput:       ti,
		state:           stateCardSelect,
		config:          config,
		executor:        executor,
		fileAccess:      fileAccess,
		ctx:             ctx,
		cancel:          cancel,
		resourceTracker: resourceTracker,
		operationLock:   &sync.Mutex{},
	}
}

// Init implements tea.Model
func (m uiModel) Init() tea.Cmd {
	return nil
}

// Custom messages for handling background operations
type successMsg struct {
	message string
}

type errMsg struct {
	err error
}

// safelyPerformBackgroundOperation executes a background operation with proper error handling
func (m *uiModel) safelyPerformBackgroundOperation(operation func() (string, error)) tea.Cmd {
	return func() tea.Msg {
		m.operationLock.Lock()
		defer m.operationLock.Unlock()

		if m.uiClosed {
			return nil
		}

		if m.ctx.Err() != nil {
			return errMsg{err: m.ctx.Err()}
		}

		result, err := operation()
		if err != nil {
			return errMsg{err: err}
		}

		return successMsg{message: result}
	}
}

// Update handles user input and state transitions
func (m uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case successMsg:
		m.successMessage = msg.message
		m.state = stateSuccess
		return m, nil

	case errMsg:
		if errors.Is(msg.err, context.Canceled) {
			m.cancel()
			m.uiClosed = true
			return m, tea.Quit
		}

		m.error = msg.err.Error()
		m.state = stateError
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			slog.Debug("User quit application")
			m.cancel()
			m.uiClosed = true
			return m, tea.Quit
		}

		switch m.state {
		case stateCardSelect:
			switch {
			case key.Matches(msg, keys.Enter):
				selectedItem, ok := m.list.SelectedItem().(listItem)
				if !ok || len(m.cards) == 0 {
					slog.Error("No card selected or no cards available")
					m.error = "No card selected or no cards available"
					m.state = stateError
					return m, nil
				}

				m.selectedCard = *selectedItem.card

				if m.selectedCard.IsVirtual {
					m.warning = "This appears to be a virtual audio device. Continue with caution."
				} else {
					m.warning = ""
				}

				if m.selectedCard.ValidationErr != nil {
					if m.warning != "" {
						m.warning += "\n"
					}
					m.warning += fmt.Sprintf("Validation warning: %s", m.selectedCard.ValidationErr)
				}

				m.customName = m.selectedCard.FriendlyName
				m.textInput.SetValue(m.customName)
				m.textInput.Focus()
				m.state = stateNameInput
				return m, textinput.Blink
			}

			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)

		case stateNameInput:
			switch {
			case key.Matches(msg, keys.Enter):
				customName := m.textInput.Value()
				if customName == "" {
					m.error = "Device name cannot be empty"
					return m, nil
				}

				cleanedName := cleanupName(customName)
				if cleanedName != customName {
					m.customName = cleanedName
					m.textInput.SetValue(cleanedName)
					return m, nil
				}

				m.customName = cleanedName
				m.state = stateConfirmation
				return m, nil

			case key.Matches(msg, keys.Back):
				m.textInput.Blur()
				m.warning = ""
				m.state = stateCardSelect
				return m, nil
			}

			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)

		case stateConfirmation:
			switch {
			case key.Matches(msg, keys.Confirm):
				installCmd := m.safelyPerformBackgroundOperation(func() (string, error) {
					return performInstallation(m.ctx, &m.selectedCard, m.customName, m.config, m.executor, m.fileAccess)
				})
				return m, installCmd

			case key.Matches(msg, keys.Back):
				m.state = stateNameInput
				return m, textinput.Blink
			}

		case stateError:
			m.error = ""
			m.state = stateCardSelect
			return m, nil

		case stateSuccess:
			m.cancel()
			m.uiClosed = true
			return m, tea.Quit
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI based on current state
func (m uiModel) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(fmt.Sprintf(" %s v%s ", AppName, AppVersion)) + "\n\n")

	switch m.state {
	case stateCardSelect:
		sb.WriteString(activeStyle.Render("Step 1: Select a USB sound card") + "\n\n")
		sb.WriteString(m.list.View() + "\n\n")
		sb.WriteString(inactiveStyle.Render("Step 2: Enter custom name") + "\n")

	case stateNameInput:
		sb.WriteString(inactiveStyle.Render("Step 1: Select a USB sound card") + "\n")
		sb.WriteString(fmt.Sprintf("Selected: %s\n\n", highlightStyle.Render(m.selectedCard.Vendor+" "+m.selectedCard.Product)))

		if m.warning != "" {
			sb.WriteString(warningStyle.Render("Warning: "+m.warning) + "\n\n")
		}

		sb.WriteString(activeStyle.Render("Step 2: Enter custom name for this device") + "\n\n")
		sb.WriteString(m.textInput.View() + "\n\n")
		sb.WriteString("This name will be used to identify the device in ALSA.\n")
		sb.WriteString("Press Enter to confirm or Esc to go back.\n")

		if m.error != "" {
			sb.WriteString("\n" + errorStyle.Render(m.error) + "\n")
		}

	case stateConfirmation:
		sb.WriteString("Please confirm the following configuration:\n\n")
		sb.WriteString(fmt.Sprintf("Device: %s\n", highlightStyle.Render(m.selectedCard.Vendor+" "+m.selectedCard.Product)))
		sb.WriteString(fmt.Sprintf("Card Number: %s\n", m.selectedCard.CardNumber))
		sb.WriteString(fmt.Sprintf("VID:PID: %s:%s\n", m.selectedCard.VendorID, m.selectedCard.ProductID))

		if m.selectedCard.Serial != "" {
			sb.WriteString(fmt.Sprintf("Serial: %s\n", m.selectedCard.Serial))
		}

		if m.selectedCard.PhysicalPort != "" {
			sb.WriteString(fmt.Sprintf("Physical Port: %s\n", m.selectedCard.PhysicalPort))
		}

		if m.selectedCard.IsVirtual {
			sb.WriteString("Type: Virtual Device\n")
		}

		sb.WriteString(fmt.Sprintf("\nCustom Name: %s\n\n", highlightStyle.Render(m.customName)))

		if m.warning != "" {
			sb.WriteString(warningStyle.Render("Warning: "+m.warning) + "\n\n")
		}

		sb.WriteString("Press 'y' to confirm or Esc to go back.")

	case stateError:
		sb.WriteString(errorStyle.Render("Error:") + "\n\n")
		sb.WriteString(m.error + "\n\n")
		sb.WriteString("Press any key to return to device selection...")

	case stateSuccess:
		sb.WriteString(infoStyle.Render("Success!") + "\n\n")
		sb.WriteString(m.successMessage + "\n\n")

		rulePath := filepath.Join(m.config.UdevRulesPath,
			fmt.Sprintf("89-usb-soundcard-%s-%s.rules", m.selectedCard.VendorID, m.selectedCard.ProductID))
		sb.WriteString(fmt.Sprintf("Rule file created at: %s\n\n", rulePath))

		sb.WriteString("Important: For the changes to take full effect, please:\n")
		sb.WriteString("1. Disconnect and reconnect the USB sound device, or\n")
		sb.WriteString("2. Reboot your system\n\n")

		sb.WriteString("For immediate application of rules without rebooting, run:\n")
		sb.WriteString("sudo udevadm control --reload-rules && sudo udevadm trigger --action=add --subsystem-match=sound\n\n")

		sb.WriteString("Press any key to exit...")
	}

	return docStyle.Render(sb.String())
}

// runUI starts the terminal UI for interactive mode
func runUI(ctx context.Context, cards []USBSoundCard, config *Config, executor *CommandExecutor, fileAccess *SafeFileAccess, resourceTracker *ResourceTracker) (string, error) {
	if len(cards) == 0 {
		return "", ErrNoUSBSoundCards
	}

	model := initialUIModel(cards, config, executor, fileAccess, resourceTracker)

	p := tea.NewProgram(model, tea.WithAltScreen())

	cancelCh := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			p.Send(errMsg{err: ctx.Err()})
		case <-cancelCh:
			return
		}
	}()

	finalModel, err := p.Run()
	close(cancelCh)

	if err != nil {
		return "", fmt.Errorf("UI error: %w", err)
	}

	m, ok := finalModel.(uiModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type returned from UI")
	}

	if m.successMessage != "" {
		return m.successMessage, nil
	}

	return "", ErrOperationCancelled
}
