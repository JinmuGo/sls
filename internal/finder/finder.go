package finder

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// Item represents a selectable entry in the finder.
type Item struct {
	Label    string // Display text (e.g., "⭐︎my-server")
	Alias    string // Actual host alias (e.g., "my-server")
	IsLast   bool   // true if this is the last container under a host (for └─)
	IsNested bool   // true if this is a container nested under a host
}

// SelectResult holds the result of a Select call.
type SelectResult struct {
	Alias  string // selected item alias
	Action string // "connect", "rename", "scan", "delete", "star"
}

// SelectOpts configures the finder behavior.
type SelectOpts struct {
	StatusMsg   string // temporary status message (e.g., "✓ Starred prod")
	RestoreAlias string // alias to restore cursor to after rebuild
	HostCount   int    // number of hosts (for header)
	ContainerCount int // number of containers (for header)
	HasScanned  bool   // true if user has ever scanned (hides first-run hint)
}

// Select launches the interactive finder TUI and returns the selected item's alias.
// Returns empty SelectResult if the user cancels (Esc/Ctrl+C).
func Select(items []Item, opts SelectOpts) (SelectResult, error) {
	if len(items) == 0 {
		return SelectResult{}, nil
	}

	m := newModel(items, opts)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return SelectResult{}, fmt.Errorf("finder: %w", err)
	}

	fmt.Fprint(os.Stderr, "\033[A\033[K")

	result := finalModel.(model)
	if result.cancelled {
		return SelectResult{}, nil
	}
	return SelectResult{Alias: result.selected, Action: result.action}, nil
}

type model struct {
	items          []Item
	filtered       []filteredItem
	query          string
	cursor         int
	selected       string
	action         string
	cancelled      bool
	quitting       bool
	width          int
	height         int
	statusMsg      string
	statusExpiry   time.Time
	confirmDelete  bool // true when showing delete confirmation
	hostCount      int
	containerCount int
	hasScanned     bool
}

type filteredItem struct {
	item         Item
	matchIndices []int
}

// clearStatusMsg is sent after the status flash expires.
type clearStatusMsg struct{}

func newModel(items []Item, opts SelectOpts) model {
	m := model{
		items:          items,
		hostCount:      opts.HostCount,
		containerCount: opts.ContainerCount,
		hasScanned:     opts.HasScanned,
	}
	if opts.StatusMsg != "" {
		m.statusMsg = opts.StatusMsg
		m.statusExpiry = time.Now().Add(2 * time.Second)
	}
	m.filter()

	// Restore cursor position
	if opts.RestoreAlias != "" {
		for i, fi := range m.filtered {
			if fi.item.Alias == opts.RestoreAlias {
				m.cursor = i
				break
			}
		}
	}
	return m
}

func (m *model) filter() {
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

func (m model) Init() tea.Cmd {
	if !m.statusExpiry.IsZero() {
		d := time.Until(m.statusExpiry)
		if d > 0 {
			return tea.Tick(d, func(time.Time) tea.Msg { return clearStatusMsg{} })
		}
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		// Delete confirmation mode — only y/n/esc accepted
		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y":
				m.action = "delete"
				m.quitting = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmDelete = false
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = m.filtered[m.cursor].item.Alias
				m.action = "connect"
			}
			m.quitting = true
			return m, tea.Quit
		case "ctrl+r":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = m.filtered[m.cursor].item.Alias
				m.action = "rename"
				m.quitting = true
				return m, tea.Quit
			}
		case "ctrl+s":
			// Scan: only on SSH hosts (no ::)
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				alias := m.filtered[m.cursor].item.Alias
				if !m.filtered[m.cursor].item.IsNested {
					m.selected = alias
					m.action = "scan"
					m.quitting = true
					return m, tea.Quit
				}
			}
		case "ctrl+d":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = m.filtered[m.cursor].item.Alias
				m.confirmDelete = true
			}
		case "ctrl+f":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = m.filtered[m.cursor].item.Alias
				m.action = "star"
				m.quitting = true
				return m, tea.Quit
			}
		case "up", "ctrl+p", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n", "ctrl+j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
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

func (m model) View() string {
	if m.quitting {
		return ""
	}

	// Header
	header := StylePrompt.Render("sls > ") + m.query + "█"
	countStr := fmt.Sprintf("  %d hosts · %d containers", m.hostCount, m.containerCount)
	if m.query != "" {
		countStr = fmt.Sprintf("  %d/%d", len(m.filtered), len(m.items))
	}
	header += "\n" + StyleDim.Render(countStr)

	// Status flash
	if m.statusMsg != "" {
		header += "\n" + m.statusMsg
	}

	// Calculate available height
	headerLines := 2
	if m.statusMsg != "" {
		headerLines++
	}
	hintLines := 1

	listHeight := len(m.filtered)
	if m.height > 0 && listHeight > m.height-headerLines-hintLines {
		listHeight = m.height - headerLines - hintLines
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
		maxLabel = m.width - 4
	}

	var lines []string
	if len(m.filtered) == 0 && m.query != "" {
		// Empty state
		lines = append(lines, "")
		lines = append(lines, StyleDim.Render(fmt.Sprintf("  No matches for %q", m.query)))
		lines = append(lines, "")
	} else {
		for i := start; i < end; i++ {
			fi := m.filtered[i]
			label := fi.item.Label

			// Add tree characters for nested containers
			if fi.item.IsNested && m.query == "" {
				if fi.item.IsLast {
					label = "  └─ " + label
				} else {
					label = "  ├─ " + label
				}
			}

			label = truncatePlain(label, maxLabel)
			if i == m.cursor {
				lines = append(lines, StyleCursorBar.Render(" ")+" "+StyleCursorLabel.Render(label))
			} else {
				lines = append(lines, "  "+label)
			}
		}
	}

	// First-run hint for hosts without containers
	if !m.hasScanned && m.containerCount == 0 && m.query == "" && len(m.filtered) > 0 {
		lines = append(lines, "")
		lines = append(lines, StyleDim.Render("  tip: press ctrl+s on a host to discover containers"))
	}

	// Hint bar
	hint := m.buildHintBar()

	return header + "\n" + strings.Join(lines, "\n") + hint
}

func (m model) buildHintBar() string {
	if m.confirmDelete {
		name := m.selected
		return "\n" + StyleError.Render(fmt.Sprintf("  Delete %s? [y/n]", name))
	}

	if len(m.filtered) == 0 {
		return "\n" + StyleDim.Render("  esc quit")
	}

	w := m.width
	if w <= 0 {
		w = 80
	}

	// Hide hint bar on very short terminals
	if m.height > 0 && m.height < 15 {
		return ""
	}

	isContainer := false
	if m.cursor < len(m.filtered) {
		isContainer = m.filtered[m.cursor].item.IsNested
	}

	if w < 60 {
		// Minimal hints for narrow terminals
		return ""
	}

	if w < 80 {
		// Short hints
		if isContainer {
			return "\n" + StyleDim.Render("  ⏎ connect · ^r rename · ^f star · ^d delete · esc")
		}
		return "\n" + StyleDim.Render("  ⏎ connect · ^s scan · ^r rename · ^f star · esc")
	}

	// Full hints
	if isContainer {
		return "\n" + StyleDim.Render("  ^j/k up/down · enter connect · ^r rename · ^f star · ^d delete · esc quit")
	}
	return "\n" + StyleDim.Render("  ^j/k up/down · enter connect · ^r rename · ^s scan · ^f star · ^d delete · esc quit")
}

// truncatePlain truncates a plain-text string to maxWidth runes.
func truncatePlain(s string, maxWidth int) string {
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	return string(runes[:maxWidth-1]) + "…"
}
