package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jessevdk/go-flags"
	"github.com/muesli/termenv"
)

type focus int

const (
	jellyfin focus = iota
	bottombar
)

type screen int

const (
	mainScreen screen = iota
	downloadScreen
)

type model struct {
	jellyfinViewModel jellyfinViewModel
	bottombarModel    bottombarModel
	downloadModel     downloadModel
	focus             focus
	currentScreen     screen
	width             int
	height            int
	config            *Config
	lastRequest       int
}

type infoMsg struct {
	info string
}

func (m *model) InitModel() {
	m.config = getConfig()
	m.jellyfinViewModel.config = m.config
	m.jellyfinViewModel.InitModel()
	m.jellyfinViewModel.isActive = true
	m.bottombarModel.InitModel()
	m.bottombarModel.config = m.config

	if args.Download {
		m.currentScreen = downloadScreen
		m.focus = jellyfin
		m.downloadModel = downloadModel{width: m.width, height: m.height, config: m.config}
		m.downloadModel.InitModel()
	}
}

func (m model) Init() tea.Cmd {
	if args.APIKey {
		return tea.Batch(m.jellyfinViewModel.Init(), sendMessage(incorrectAPIKeyMsg("")))
	}
	if args.APIEndPoint {
		return tea.Batch(m.jellyfinViewModel.Init(), sendMessage(incorrectAPIEndPointMsg("")))
	}
	if args.UserId {
		return tea.Batch(m.jellyfinViewModel.Init(), sendMessage(incorrectUserIdMsg("")))
	}
	if args.Download {
		return tea.Batch(m.jellyfinViewModel.Init(), m.downloadModel.Init())
	}
	return m.jellyfinViewModel.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.currentScreen == mainScreen {
		var cmds = make([]tea.Cmd, 0)
		shouldBePassed := true
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "tab":
				if m.focus == jellyfin {
					m.focus = bottombar
					m.jellyfinViewModel.isActive = false
					m.bottombarModel.isActive = true
				} else {
					m.jellyfinViewModel.isActive = true
					m.bottombarModel.isActive = false
					m.focus = jellyfin
				}
				return m, nil
			default:
				switch m.focus {
				case jellyfin:
					//var cmd tea.Cmd
					//_ /*m.jellyfinViewModel*/, cmd := m.jellyfinViewModel.Update(msg)
					return m.callJellyfinUpdate(msg)
				case bottombar:
					return m.callBottombarUpdate(msg)
				}
			}
		case tea.WindowSizeMsg:
			//h, v := docStyle.GetFrameSize()
			m.width = msg.Width
			m.height = msg.Height
			jModel, jCmds := m.jellyfinViewModel.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height - 2})
			cmds = append(cmds, jCmds)
			m.jellyfinViewModel = jModel

			shouldBePassed = false

		case infoMsg:
			return m.callBottombarUpdate(msg)
		case incorrectAPIEndPointMsg, incorrectUserIdMsg, incorrectAPIKeyMsg:
			var str string
			switch msg := msg.(type) {
			case incorrectAPIEndPointMsg:
				str = string(msg)
			case incorrectUserIdMsg:
				str = string(msg)
			case incorrectAPIKeyMsg:
				str = string(msg)
			}
			m.jellyfinViewModel.loadingMsg = str

			m.focus = bottombar
			m.bottombarModel.isActive = true
			m.bottombarModel.buttonsActive = false
			m.jellyfinViewModel.isActive = false
			return m.callBottombarUpdate(msg)
		case reloadItemsMsg:
			m.focus = jellyfin
			m.bottombarModel.isActive = false
			m.jellyfinViewModel.isActive = true
			m.callJellyfinUpdate(msg)
		default:
			if m.focus == jellyfin {
				return m.callJellyfinUpdate(msg)
			} else {
				return m.callBottombarUpdate(msg)
			}
		case inputDoneMsg:
			if msg.id != DownloadLocation {
				m.focus = jellyfin
			}
			return m.callBottombarUpdate(msg)
		case inputCancelMsg:
			if len(m.jellyfinViewModel.lists) == 0 {
				m.bottombarModel.buttonsActive = true
				return m, sendMessage(reloadItemsMsg{})
			}

		case buttonPressedMsg:
			switch ButtonId(msg) {
			case DownloadAll:
				m.currentScreen = downloadScreen
				m.focus = jellyfin
				m.downloadModel = downloadModel{width: m.width, height: m.height, config: m.config}
				m.downloadModel.InitModel()
				return m, m.downloadModel.Init()
			default:
				return m.callBottombarUpdate(msg)
			}
		}

		if shouldBePassed {
			jModel, jCmds := m.jellyfinViewModel.Update(msg)
			cmds = append(cmds, jCmds)
			m.jellyfinViewModel = jModel
		}

		return m, tea.Batch(cmds...)
	} else {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "esc":
				m.downloadModel.CancelAll()
				m.currentScreen = mainScreen
				m.jellyfinViewModel.isActive = true
				m.bottombarModel.isActive = false
				return m, m.jellyfinViewModel.UpdateItems
			default:
				return m.callDownloadUpdate(msg)
			}
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.jellyfinViewModel, _ = m.jellyfinViewModel.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height - 2})
			return m.callDownloadUpdate(msg)
		default:
			return m.callDownloadUpdate(msg)
		}
	}
}

func (m model) View() string {
	if m.currentScreen == mainScreen {
		jellyfinView := m.jellyfinViewModel.View()
		bottombarView := m.bottombarModel.View()

		return lipgloss.JoinVertical(lipgloss.Left, jellyfinView, bottombarView)
	} else {
		return m.downloadModel.View()
	}
}

var program *tea.Program

type ProgramArgs struct {
	Download    bool `short:"d" long:"download" description:"Start downloading selected items"`
	APIKey      bool `short:"k" long:"apikey" description:"Ask for API key"`
	UserId      bool `short:"u" long:"userid" description:"Ask for UserId"`
	APIEndPoint bool `short:"e" long:"endpoint" description:"Ask for API Endpoint"`
}

var args ProgramArgs = ProgramArgs{}

func main() {
	lipgloss.SetHasDarkBackground(termenv.HasDarkBackground())
	_, err := flags.Parse(&args)
	if err != nil {
		if flags.WroteHelp(err) {
			os.Exit(0)
		} else {
			panic(err)
		}
	}

	m := model{}
	m.InitModel()

	program = tea.NewProgram(m, tea.WithAltScreen())

	if err := program.Start(); err != nil {
		fmt.Println("Error running program:", err)
		program.Kill()
	}
}
