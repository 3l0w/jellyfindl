package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Input int

type inputModel struct {
	textInput textinput.Model
	isActive  bool
	id        Input
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (inputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	if cmd != nil {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			m.isActive = false
			return m, m.doneInput
		case "esc":
			m.isActive = false
			return m, m.cancelInput
		}
	}

	return m, nil
}

func (m inputModel) View() string {
	return m.textInput.View()
}

func InitInput(id Input, input, de, placeholder string) inputModel {
	ti := textinput.New()
	ti.Prompt = input + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(" ? ")
	ti.Placeholder = placeholder
	ti.SetValue(de)
	ti.Focus()
	ti.CharLimit = 33
	ti.Width = 33

	return inputModel{
		id:        id,
		textInput: ti,
		isActive:  true,
	}
}

type inputDoneMsg struct {
	id    Input
	value string
}

func (m inputModel) doneInput() tea.Msg {
	return inputDoneMsg{m.id, m.textInput.Value()}
}

type inputCancelMsg struct{}

func (m inputModel) cancelInput() tea.Msg {
	return inputCancelMsg{}
}
