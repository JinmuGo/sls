package finder

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// SelectMulti launches a multi-select TUI. Returns the aliases of selected items.
// Returns nil if the user cancels (Esc/Ctrl+C).
// preChecked contains aliases that should be checked by default.
func SelectMulti(items []Item, prompt string, preChecked ...string) ([]string, error) {
	if len(items) == 0 {
		return nil, nil
	}

	m := newMultiModel(items, prompt)
	for _, alias := range preChecked {
		m.checked[alias] = true
	}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("finder: %w", err)
	}

	fmt.Fprint(os.Stderr, "\033[A\033[K")

	result := finalModel.(multiModel)
	if result.cancelled {
		return nil, nil
	}

	var selected []string
	for alias := range result.checked {
		selected = append(selected, alias)
	}
	return selected, nil
}

type multiModel struct {
	items     []Item
	filtered  []filteredItem
	query     string
	cursor    int
	checked   map[string]bool
	cancelled bool
	quitting  bool
	prompt    string
	width     int
	height    int
}

func newMultiModel(items []Item, prompt string) multiModel {
	m := multiModel{
		items:   items,
		checked: make(map[string]bool),
		prompt:  prompt,
	}
	m.filter()
	return m
}

func (m *multiModel) filter() {
	if m.query == "" {
		m.filtered = make([]filteredItem, len(m.items))
		for i, item := range m.items {
			m.filtered[i] = filteredItem{item: item}
		}
		return
	}

	labels := make([]string, len(m.items))
	for i, item := range m.items {
		labels[i] = item.Label
	}

	matches := fuzzy.Find(m.query, labels)
	sort.Stable(matches)

	m.filtered = make([]filteredItem, len(matches))
	for i, match := range matches {
		m.filtered[i] = filteredItem{
			item:         m.items[match.Index],
			matchIndices: match.MatchedIndexes,
		}
	}
}

func (m multiModel) Init() tea.Cmd {
	return nil
}

func (m multiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.quitting = true
			return m, tea.Quit
		case "up", "ctrl+p", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n", "ctrl+j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case " ", "tab":
			if m.cursor < len(m.filtered) {
				alias := m.filtered[m.cursor].item.Alias
				if m.checked[alias] {
					delete(m.checked, alias)
				} else {
					m.checked[alias] = true
				}
			}
		case "ctrl+a":
			if len(m.checked) == len(m.items) {
				m.checked = make(map[string]bool)
			} else {
				for _, item := range m.items {
					m.checked[item.Alias] = true
				}
			}
		case "backspace":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.filter()
				m.cursor = 0
			}
		default:
			if len(msg.String()) == 1 {
				m.query += msg.String()
				m.filter()
				m.cursor = 0
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m multiModel) View() string {
	if m.quitting {
		return ""
	}

	// Header
	header := StylePrompt.Render(m.prompt+" > ") + m.query + "█"
	header += StyleDim.Render(fmt.Sprintf("  %d/%d selected", len(m.checked), len(m.items)))

	// Fixed height = number of items
	listHeight := len(m.filtered)
	if m.height > 0 && listHeight > m.height-2 {
		listHeight = m.height - 2
	}
	if listHeight < 1 {
		listHeight = 1
	}

	// Scroll window
	start := 0
	if m.cursor >= start+listHeight {
		start = m.cursor - listHeight + 1
	}
	if m.cursor < start {
		start = m.cursor
	}
	end := start + listHeight
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	// Render lines
	maxLabel := 60
	if m.width > 0 {
		maxLabel = m.width - 6
	}
	var lines []string
	for i := start; i < end; i++ {
		fi := m.filtered[i]
		label := truncatePlain(fi.item.Label, maxLabel)

		var check string
		if m.checked[fi.item.Alias] {
			check = StyleCheck.Render("◉ ")
		} else {
			check = StyleUncheck.Render("○ ")
		}

		if i == m.cursor {
			lines = append(lines, StyleCursorBar.Render(" ")+" "+check+StyleCursorLabel.Render(label))
		} else {
			lines = append(lines, "  "+check+label)
		}
	}

	// Help
	help := StyleDim.Render("  space select · ctrl+a all · enter confirm · esc cancel")

	return header + "\n" + strings.Join(lines, "\n") + "\n" + help
}
