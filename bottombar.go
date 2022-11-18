package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	APIKey Input = iota
	UserID
	DownloadLocation
	APIEndpoint
)

const (
	DownloadAll ButtonId = iota
	SetApiKey
	SetUserId
	SetDownloadLocation
	SetApiEndpoint
)

type bottombarModel struct {
	input         inputModel
	buttons       []buttonModel
	config        *Config
	isActive      bool
	focused       int
	info          string
	buttonsActive bool
}

func (m *bottombarModel) InitModel() {
	m.buttons = []buttonModel{
		{title: "Download All", id: DownloadAll},
		{title: "Set API Key", id: SetApiKey},
		{title: "Set API Endpoint", id: SetApiEndpoint},
		{title: "Set User ID", id: SetUserId},
		{title: "Set DownloadLocation", id: SetDownloadLocation},
	}
	m.buttonsActive = true
}

func (m bottombarModel) Init() tea.Cmd {
	return nil
}

func (m bottombarModel) Update(msg tea.Msg) (bottombarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case infoMsg:
		m.info = msg.info
	case tea.KeyMsg:
		if m.input.isActive {
			return m.callInputUpdate(msg)
		}
		switch msg.String() {
		case "right":
			m.focused++
			if m.focused == len(m.buttons) {
				m.focused = 0
			}
			return m, nil
		case "left":
			m.focused--
			if m.focused == -1 {
				m.focused = len(m.buttons) - 1
			}
			return m, nil
		default:
			return m.callButtonUpdate(msg)
		}
	case buttonPressedMsg:
		switch ButtonId(msg) {
		case SetApiKey:
			m.input = InitInput(APIKey, "API Key", m.config.APIKey, "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
			return m, m.input.Init()
		case SetUserId:
			m.input = InitInput(UserID, "User ID", m.config.UserId, "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
			return m, m.input.Init()
		case SetDownloadLocation:
			m.input = InitInput(DownloadLocation, "Download Location", m.config.DownloadLocation, "/home/user/Jellyfin")
			return m, m.input.Init()
		case SetApiEndpoint:
			m.input = InitInput(APIEndpoint, "API Endpoint", m.config.APIEndpoint, "http://jellyfin")
			return m, m.input.Init()
		}
	case inputDoneMsg:
		m.buttonsActive = true
		var shouldReload bool
		switch inputId := msg.id; inputId {
		case APIKey:
			m.config.APIKey = msg.value
			shouldReload = true
		case UserID:
			m.config.UserId = msg.value
			shouldReload = true
		case DownloadLocation:
			m.config.DownloadLocation = msg.value
		case APIEndpoint:
			if strings.HasSuffix(msg.value, "/") {
				msg.value = msg.value[:len(msg.value)-1]
			}
			m.config.APIEndpoint = msg.value
			shouldReload = true
		}
		writeConfig(*m.config)
		if shouldReload {
			return m, sendMessage(reloadItemsMsg{})
		}
	case incorrectAPIKeyMsg:
		m.input = InitInput(APIKey, "API Key", m.config.APIKey, "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		return m, m.input.Init()
	case incorrectUserIdMsg:
		m.input = InitInput(UserID, "User ID", m.config.UserId, "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		return m, m.input.Init()
	case incorrectAPIEndPointMsg:
		m.input = InitInput(APIEndpoint, "API Endpoint", m.config.APIEndpoint, "http://jellyfin")
		return m, m.input.Init()
	}
	return m, nil
}

func (m bottombarModel) View() string {
	var view string
	if m.buttonsActive {
		for i, button := range m.buttons {
			if i == m.focused && m.isActive {
				button.active = true
			} else {
				button.active = false
			}
			view += button.View()
		}
		view += " "
	}
	if m.input.isActive {
		view += m.input.View()
		view += " "
	}
	if m.info != "" {
		view += m.info
	}

	return view
}

var buttonTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))

var buttonStyle = lipgloss.NewStyle().
	MarginLeft(1).
	PaddingRight(1).
	PaddingLeft(1)
var notActiveButtonStyle = buttonStyle.Copy().Background(lipgloss.Color("8"))
var activeButtonStyle = buttonStyle.Copy().Background(lipgloss.Color("4"))

type ButtonId int

type buttonModel struct {
	id     ButtonId
	title  string
	active bool
}

func (m buttonModel) Init() tea.Cmd {
	return nil
}

func (m buttonModel) Update(msg tea.Msg) (buttonModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, m.sendPressed
		}
	}
	return m, nil
}

func (m buttonModel) View() string {
	view := buttonTextStyle.Render(m.title)
	if m.active {
		view = activeButtonStyle.Render(view)
	} else {
		view = notActiveButtonStyle.Render(view)
	}

	return view
}

type buttonPressedMsg ButtonId

func (m buttonModel) sendPressed() tea.Msg {
	return buttonPressedMsg(m.id)
}
