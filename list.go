package main

import (
	"os"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2).
	Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
var unSelect = lipgloss.NewStyle().Margin(0, 1).Foreground(lipgloss.Color("220"))
var selectedItem = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
var downloadedItem = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
var classicItem = lipgloss.NewStyle()

type item struct {
	title, desc, id string
	isFolder        bool
}

type reloadItemsMsg struct{}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type jellyfinViewModel struct {
	lists      []*list.Model
	focused    int
	isActive   bool
	loaded     map[string][]JellyfinItem
	width      int
	height     int
	config     *Config
	loadingMsg string
	requestId  int
}

func (m *jellyfinViewModel) InitModel() {
	m.loaded = make(map[string][]JellyfinItem)
	m.loadingMsg = "Loading..."
}

/* Bubble tea */
func (m jellyfinViewModel) Init() tea.Cmd {
	return m.UpdateItems
}

func (m jellyfinViewModel) Update(msg tea.Msg) (jellyfinViewModel, tea.Cmd) {
	var cmds = make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "right":
			m.focused++
			if m.focused == len(m.lists) {
				m.focused = 0
			}
			return m, nil
		case "left":
			m.focused--
			if m.focused == -1 {
				m.focused = len(m.lists) - 1
			}
			return m, nil
		case "up", "down":
			m.requestId++
			cmds = append(cmds, m.UpdateItems)
		case "enter", "space":
			cmds = append(cmds, m.SelectUnSelect)
		case "r":
			id := m.lists[m.focused].SelectedItem().(item).id
			path, ok := m.config.Downloaded[id]
			if ok {
				os.Remove(path)
				delete(m.config.Downloaded, id)
				writeConfig(*m.config)
				return m, m.UpdateItems
			}
		}

	case tea.WindowSizeMsg: //Custom send by the main model
		m.width = msg.Width
		m.height = msg.Height
		setListsSize(m.lists, msg.Width, msg.Height)
		docStyle = docStyle.Height(msg.Height - docStyle.GetVerticalFrameSize())
		docStyle = docStyle.Width(msg.Width / 5)

	case itemsMsg:
		if m.requestId == msg.requestId {
			return m.applyItems(msg.lists)
		}
	case selectedMsg:
		writeConfig(*m.config)
		m.requestId++
		cmds = append(cmds, m.UpdateItems)
	case reloadItemsMsg:
		m.lists = make([]*list.Model, 0)
		m.InitModel()
		return m, m.UpdateItems
	}

	if len(m.lists) != 0 {
		ml, cmd := m.lists[m.focused].Update(msg)
		m.lists[m.focused] = &ml
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m jellyfinViewModel) isFiltering() bool {
	for _, l := range m.lists {
		if l.SettingFilter() {
			return true
		}
	}
	return false
}

func (m jellyfinViewModel) View() string {
	var view string
	if len(m.lists) == 0 {
		view = m.loadingMsg
	} else {
		views := make([]string, len(m.lists))
		for i, l := range m.lists {
			if i != m.focused || !m.isActive {
				l.Styles.Title = unSelect
				l.View()
			} else {
				l.Styles.Title = list.DefaultStyles().Title
			}
			views[i] = docStyle.Width(l.Width()).Render(l.View())
		}
		view = lipgloss.JoinHorizontal(lipgloss.Left, views...)
	}
	return view
}

/* Sizing */
func setListsSize(lists []*list.Model, width int, height int) {
	h, v := docStyle.GetFrameSize()
	var sizeUsed int
	for i, l := range lists {
		var size int
		if i == len(lists)-1 {
			size = width - sizeUsed - h
		} else {
			size = getListWidth(*l) + 3
			size = min(size, int(float32(width)/3.6))
		}

		sizeUsed += size + h
		l.SetSize(size, height-v)
	}
}

/* Jellyfin Item providers & update */
func (m *jellyfinViewModel) fillItems(parentId string) []list.Item {
	collections, ok := m.loaded[parentId]
	if !ok {
		collections = getChilds(parentId, m.config).Items
	}

	items := make([]list.Item, len(collections))
	for i, e := range collections {
		name := e.Name

		if e.SeriesName != "" && !e.IsFolder {
			name = strconv.Itoa(e.EpisodeNumber) + ". " + name
		}

		_, ok := m.config.Downloaded[e.Id]
		if ok {
			name = downloadedItem.Render(name)
		} else if m.config.Selected.Contains(e.Id) {
			name = selectedItem.Render(name)
		} else {
			name = classicItem.Render(name)
		}
		items[i] = item{title: name, id: e.Id, isFolder: e.IsFolder}
	}
	m.loaded[parentId] = collections
	return items
}

type itemsMsg struct {
	requestId int
	lists     [][]list.Item
}

func (m *jellyfinViewModel) UpdateItems() tea.Msg {
	var requestId = m.requestId
	var lastParent string
	var i int
	var lists [][]list.Item
	for {
		items := m.fillItems(lastParent)
		if len(items) == 0 {
			break
		}
		lists = append(lists, items)

		var it item
		if isInside(m.lists, i) {
			cursor := m.lists[i].Index()
			if cursor < len(items) {
				it = items[cursor].(item)
			}
		} else {
			it = items[0].(item)
		}

		lastParent = it.id
		i++

		if !items[0].(item).isFolder {
			break
		}
		if i > 10 {
			break
		}
	}

	return itemsMsg{requestId, lists}
}

var listTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))

func (m jellyfinViewModel) applyItems(lists [][]list.Item) (jellyfinViewModel, tea.Cmd) {
	var cmds []tea.Cmd
	var viewLists []*list.Model
	for i, items := range lists {
		var l *list.Model
		if i >= len(m.lists) {
			l = createList(items, false)
		} else {
			l = m.lists[i]
			if !isGoodContent(m.lists[i].Items(), items) {
				//l = createList(items)
				l.SetItems(items)
			}
		}
		if i != 0 {
			active := viewLists[i-1].SelectedItem()
			if active == nil {
				active = viewLists[i-1].Items()[0]
			}
			l.Title = active.(item).title
		}
		viewLists = append(viewLists, l)
	}
	m.lists = viewLists
	setListsSize(m.lists, m.width, m.height)
	return m, tea.Batch(cmds...)
}

/* Selection Handling */
type selectedMsg struct{}

func (m *jellyfinViewModel) SelectUnSelect() tea.Msg {
	lFocused := m.lists[m.focused]
	it := lFocused.SelectedItem().(item)
	added := m.config.Selected.Toggle(it.id)
	if it.isFolder {
		m.forcedSelectUnSelect(it.id, added)
	}

	return selectedMsg{}
}

func (m *jellyfinViewModel) forcedSelectUnSelect(it string, added bool) {
	collections, ok := m.loaded[it]
	if !ok {
		collections = getChilds(it, m.config).Items
		m.loaded[it] = collections
	}

	for _, child := range collections {
		if added {
			m.config.Selected.Add(child.Id)
		} else {
			m.config.Selected.Remove(child.Id)
		}
		if child.IsFolder {
			m.forcedSelectUnSelect(child.Id, added)
		}
	}
}

/* List model creation */
func createList(items []list.Item, desc bool) *list.Model {
	delegate := list.DefaultDelegate{
		ShowDescription: desc,
		Styles:          list.NewDefaultItemStyles(),
	}
	delegate.SetHeight(2)
	delegate.SetSpacing(1)

	list := list.New(items, delegate, 10, 10)
	list.SetShowHelp(false)
	return &list
}
