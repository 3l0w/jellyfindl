package main

// void F(void **p, int a) { *p = &a; }
import "C"

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("160"))

func checkError(err error) {
	if err != nil {
		program.Kill()
		fmt.Println("Error while running program:", errorStyle.Render(err.Error()))
		//os.Exit(1)
	}
}

func isInside(items []*list.Model, pos int) bool {
	if pos >= len(items) || pos < 0 {
		return false
	}

	return true
}

func isGoodContent(items1 []list.Item, items2 []list.Item) bool {
	if len(items1) != len(items2) {
		return false
	}

	for i, value := range items1 {
		if value != items2[i] {
			return false
		}
	}

	return true
}

func printStruct(val interface{}) {
	fmt.Printf("%+v\n", val)
}

func formatStruct(val interface{}) string {
	return fmt.Sprintf("%+v\n", val)
}

func remove(slice []*list.Model, s int) []*list.Model {
	return append(slice[:s], slice[s+1:]...)
}

func getListWidth(l list.Model) int {
	var max int
	for _, e := range l.Items() {
		if length := lipgloss.Width(e.(item).title); max < length {
			max = length
		}
	}
	if length := lipgloss.Width(l.Title); max < length {
		max = length
	}

	return max + 3
}

func min(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func contains(m []string, value string) bool {
	for _, v := range m {
		if v == value {
			return true
		}
	}
	return false
}

func sendMessage(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func (m model) callJellyfinUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.jellyfinViewModel, cmd = m.jellyfinViewModel.Update(msg)
	return m, cmd
}

func (m model) callBottombarUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.bottombarModel, cmd = m.bottombarModel.Update(msg)
	return m, cmd
}

func (m bottombarModel) callInputUpdate(msg tea.Msg) (bottombarModel, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m bottombarModel) callButtonUpdate(msg tea.Msg) (bottombarModel, tea.Cmd) {
	var cmd tea.Cmd
	m.buttons[m.focused], cmd = m.buttons[m.focused].Update(msg)
	return m, cmd
}

func (m model) callDownloadUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.downloadModel, cmd = m.downloadModel.Update(msg)
	return m, cmd
}

func (m downloadModel) getItem(id string) downloadItem {
	for _, v := range m.list.Items() {
		if v.(downloadItem).id == id {
			return v.(downloadItem)
		}
	}

	panic("Item not found")
}

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

type downloadItemSorter []list.Item

func (s downloadItemSorter) Len() int      { return len(s) }
func (s downloadItemSorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s downloadItemSorter) Less(i, j int) bool {
	item1 := s[i].(downloadItem)
	item2 := s[j].(downloadItem)

	if (item1.fail != "") != (item2.fail != "") {
		if item1.fail != "" {
			return true
		}
		if item2.fail != "" {
			return false
		}
	}

	if item1.downloadCompleted != item2.downloadCompleted {
		if item1.downloadCompleted {
			return false
		}
		if item2.downloadCompleted {
			return true
		}
	}

	if item1.downloadStarted != item2.downloadStarted {
		if item1.downloadStarted {
			return true
		}
		if item2.downloadStarted {
			return false
		}
	}

	if (item1.jellyfinItem.SeriesName == "") != (item2.jellyfinItem.SeriesName == "") {
		if item1.jellyfinItem.SeriesName == "" {
			return true
		}
		if item2.jellyfinItem.SeriesName == "" {
			return false
		}
	}

	if item1.jellyfinItem.SeriesName == item2.jellyfinItem.SeriesName {
		if item1.jellyfinItem.SeasonNumber == item2.jellyfinItem.SeasonNumber {
			return item1.jellyfinItem.EpisodeNumber < item2.jellyfinItem.EpisodeNumber
		}
		return item1.jellyfinItem.SeasonNumber < item2.jellyfinItem.SeasonNumber
	}
	return item1.title < item2.title
}

func sortList(items []list.Item) {
	sort.Stable(downloadItemSorter(items))
}

func (m downloadModel) getNext() (string, string) {
	for _, i := range m.list.Items() {
		item := i.(downloadItem)
		if !item.downloadCompleted {
			return item.id, getDownloadLocation(item.jellyfinItem)
		}
	}
	return "", ""
}

func (m downloadModel) updateItem(item downloadItem, msg tea.Msg) (downloadModel, tea.Cmd) {
	items := m.list.Items()
	var index int
	for i, v := range items {
		if v.(downloadItem).id == item.id {
			index = i
			break
		}
	}

	nItem, cmd := item.Update(msg)

	items[index] = nItem
	sortList(items)
	lCmd := m.list.SetItems(items)

	return m, tea.Batch(cmd, lCmd)
}
func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}

func getMapKeys[V any](m map[string]V) []string {
	var values []string
	for k, _ := range m {
		values = append(values, k)
	}
	return values
}
