package main

import (
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type downloadState int

const (
	downloading downloadState = iota
	queued
	finished
)

var downloadStarted = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Render("Downloading....")

type downloadItem struct {
	title, id                   string
	jellyfinItem                JellyfinItem
	seasonNumber, episodeNumber int
	downloadStarted             bool
	downloadCompleted           bool
	resp                        *grab.Response
	spinner                     spinner.Model
	progress                    progress.Model
	fail                        string
}

func (i downloadItem) Title() string { return i.title }

func (i downloadItem) Description() string {
	if i.fail != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Render("❌ " + i.fail)
	}

	if i.downloadCompleted {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✔ Downloaded !")
	}

	if !i.downloadStarted {
		return "Waiting...."
	}

	if i.resp == nil {
		return i.spinner.View() + " " + downloadStarted
	}

	var eta string
	var bytesPerSecond string = ByteCountSI(int64(i.resp.BytesPerSecond()))
	if i.resp.BytesPerSecond() == 0 {
		eta = "∞"
	} else {
		eta = i.resp.ETA().Sub(time.Now()).Round(time.Second).String()
	}

	return i.spinner.View() + " " + i.progress.View() + " " + bytesPerSecond + "/s " + eta
}

func (i downloadItem) FilterValue() string { return i.title }

func (i downloadItem) Update(msg tea.Msg) (downloadItem, tea.Cmd) {
	switch msg := msg.(type) {
	case startDownloadingItemMsg:
		i.downloadStarted = true
		i.spinner = spinner.NewModel()
		i.spinner.Spinner = spinner.Moon
		i.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		i.progress = progress.New(progress.WithDefaultGradient())
		return i, tea.Batch(i.spinner.Tick)
	case downloadStartedMsg:
		i.fail = ""
		if !i.downloadStarted {
			if i.resp != nil {
				msg.resp.Cancel()
			} else {
				return i, nil
			}
		}
		i.resp = msg.resp
		return i, nil
	case progress.FrameMsg:
		progressModel, cmd := i.progress.Update(msg)
		i.progress = progressModel.(progress.Model)
		return i, cmd
	case tickMsg:
		if i.resp != nil {
			cmd := i.progress.SetPercent(i.resp.Progress())
			return i, cmd
		}
	case downloadCompletedMsg:
		i.downloadCompleted = true
	case downloadFailedMsg:
		if i.downloadStarted {
			i.fail = msg.Reason
		}
		i.downloadStarted = false
		i.downloadCompleted = false
		return i, nil

	default:
		var cmd tea.Cmd
		i.spinner, cmd = i.spinner.Update(msg)
		return i, cmd
	}
	return i, nil
}

func (i *downloadItem) Cancel() {
	if i.resp != nil {
		i.resp.Cancel()
	}

	i.resp = nil
	i.downloadCompleted = false
	i.downloadStarted = false
}

type downloadModel struct {
	config        *Config
	list          list.Model
	info          string
	width, height int
	items         map[string]JellyfinItem
	downloading   map[string]*grab.Response
}

func (m *downloadModel) InitModel() {
	m.info = "Download screen"
	m.list = *createList(make([]list.Item, 0), false)
	m.downloading = make(map[string]*grab.Response)
}

func (m downloadModel) Init() tea.Cmd {
	return tea.Batch(m.filterItems, tickCmd())
}

func (m downloadModel) Update(msg tea.Msg) (downloadModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.list.Items()) != 0 {
				ok := m.list.SelectedItem().(downloadItem).downloadStarted
				if !ok {
					item := m.list.SelectedItem().(downloadItem)
					return m, m.downloadItem(item.id, getDownloadLocation(item.jellyfinItem))
				} else {
					item := m.list.SelectedItem().(downloadItem)
					delete(m.downloading, item.id)
					item.Cancel()
					return m.updateItem(item, msg)
				}
			}
		case "r":
			item := m.list.SelectedItem().(downloadItem)
			if item.downloadCompleted {
				path := m.config.Downloaded[item.id]
				os.Remove(path)
				delete(m.config.Downloaded, item.id)
				writeConfig(*m.config)
				item.Cancel()
				return m.updateItem(item, msg)
			}
		default:
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
	case infoMsg:
		m.info = msg.info
	case itemFilteredMsg:
		m.list = *createList(msg.listItems, true)
		m.list.SetShowTitle(false)
		m.list.SetHeight(m.height - 6)
		m.list.SetWidth(m.width)
		if len(msg.listItems) > 0 {
			return m, m.downloadItem(m.getNext())
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetHeight(m.height - 6)
		m.list.SetWidth(m.width)
	case startDownloadingItemMsg: //When the download request is sent
		return m.updateItem(m.getItem(string(msg)), msg)

	case downloadStartedMsg: //When the downloading starts
		m.downloading[msg.Id] = msg.resp
		return m.updateItem(m.getItem(msg.Id), msg)
	case downloadCompletedMsg: //When download is completed
		delete(m.downloading, msg.Id)
		m2, cmd := m.updateItem(m.getItem(msg.Id), msg)
		m2.config.Downloaded[msg.Id] = msg.File
		writeConfig(*m.config)
		var nextCmd tea.Cmd
		if len(m2.downloading) == 0 {
			nextCmd = m.downloadItem(m.getNext())
		}
		return m2, tea.Batch(cmd, nextCmd)
	case downloadFailedMsg: //When download failed
		delete(m.downloading, msg.Id)
		return m.updateItem(m.getItem(msg.Id), msg)
	case tickMsg:
		cmds = append(cmds, tickCmd())
	}

	for i, v := range m.list.Items() {
		dlItem, cmd := v.(downloadItem).Update(msg)
		cmdList := m.list.SetItem(i, dlItem)
		cmds = append(cmds, cmd, cmdList)
	}
	return m, tea.Batch(cmds...)
}

func (m downloadModel) View() string {
	if m.config.Selected.Size() == 0 {
		return "No items selected"
	}

	if len(m.list.Items()) == 0 {
		return "Loading...\n" + m.info
	}

	padding := lipgloss.NewStyle().Margin(0, 1)

	return lipgloss.JoinVertical(lipgloss.Left, padding.Render(m.list.View()), m.info)
}

type itemFilteredMsg struct {
	listItems []list.Item
	items     map[string]JellyfinItem
}

func (m downloadModel) filterItems() tea.Msg {
	items := getItems(m.config.Selected.Values(), m.config).Items
	downloaded := getItems(getMapKeys(m.config.Downloaded), m.config).Items
	items = append(items, downloaded...)

	msg := itemFilteredMsg{make([]list.Item, 0), make(map[string]JellyfinItem)}
	for _, v := range items {
		if !v.IsFolder {
			_, isDl := m.config.Downloaded[v.Id]
			item := downloadItem{
				title:             getTitle(v),
				id:                v.Id,
				downloadCompleted: isDl,
				jellyfinItem:      v,
			}
			msg.listItems = append(msg.listItems, item)
		}
		msg.items[v.Id] = v
	}
	sortList(msg.listItems)
	return msg
}

func getTitle(item JellyfinItem) string {
	name := strconv.Itoa(item.EpisodeNumber) + ". " + item.Name
	name = lipgloss.NewStyle().Foreground(lipgloss.Color("190")).Render(name)
	dot := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(" • ")
	if item.SeriesName != "" {
		serieStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
		name = serieStyle.Render(item.SeriesName) + dot +
			serieStyle.Render(item.SeasonName) + dot +
			name
	} else {
		name = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("Film") + dot + name
	}
	return name
}

func getDownloadLocation(item JellyfinItem) string {
	if item.SeriesName != "" {
		return filepath.Join("Series", item.SeriesName, item.SeasonName)
	} else {
		return filepath.Join("Film")
	}
}

type downloadFailedMsg struct {
	Id     string
	Reason string
}
type downloadCompletedMsg struct {
	Id   string
	File string
}

func (m downloadModel) downloadItem(item, itemDestination string) tea.Cmd {
	if item == "" {
		return nil
	}

	_, isDl := m.downloading[item]
	if isDl {
		return nil
	}
	dest := m.config.DownloadLocation
	if dest == "" {
		p, err := os.UserHomeDir()
		checkError(err)
		dest = path.Join(p, "Jellyfin")
	}

	dest = path.Join(dest, itemDestination)

	checkError(os.MkdirAll(dest, os.ModePerm))
	return func() tea.Msg {
		file, err := downloadFile(item, dest, m.config)
		if err != nil {
			return downloadFailedMsg{item, err.Error()}
		}
		return downloadCompletedMsg{item, file}
	}
}

func (m downloadModel) CancelAll() {
	for _, r := range m.downloading {
		r.Cancel()
	}
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}
