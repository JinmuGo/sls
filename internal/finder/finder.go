package finder

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jinmugo/sls/internal/hostinfo"
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

// HostInfoFetcher fetches system info for a host. Injected to keep finder
// decoupled from SSH and caching concerns.
type HostInfoFetcher interface {
	// Get returns cached info for a host, or nil if not available.
	Get(host string) *hostinfo.HostInfo
	// FetchAsync starts a background fetch. The result is delivered via tea.Cmd.
	FetchAsync(ctx context.Context, host string) (*hostinfo.HostInfo, error)
}

// SelectOpts configures the finder behavior.
type SelectOpts struct {
	StatusMsg      string           // temporary status message (e.g., "✓ Starred prod")
	RestoreAlias   string           // alias to restore cursor to after rebuild
	HostCount      int              // number of hosts (for header)
	ContainerCount int              // number of containers (for header)
	HasScanned     bool             // true if user has ever scanned (hides first-run hint)
	InfoFetcher    HostInfoFetcher  // optional: provides host info for preview panel
	InfoCache      *hostinfo.Cache  // optional: disk cache for persistence across sessions
}

// Select launches the interactive finder TUI and returns the selected item's alias.
// Returns empty SelectResult if the user cancels (Esc/Ctrl+C).
func Select(items []Item, opts SelectOpts) (SelectResult, error) {
	if len(items) == 0 {
		return SelectResult{}, nil
	}

	m := newModel(items, opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return SelectResult{}, fmt.Errorf("finder: %w", err)
	}

	result := finalModel.(model)

	// Flush in-memory cache to disk on exit
	if result.infoCache != nil && result.infoDirty {
		_ = result.infoCache.Save()
	}

	// Cancel any in-flight fetch
	if result.cancelFetch != nil {
		result.cancelFetch()
	}

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

	// Preview panel
	previewOpen   bool
	infoFetcher   HostInfoFetcher
	infoCache     *hostinfo.Cache
	infoDirty     bool                       // true if cache was updated (needs flush)
	infoMem       map[string]*hostinfo.HostInfo // in-memory cache for current session
	loadingHost   string
	fetchGen      uint64
	cancelFetch   context.CancelFunc
	listWidth     int
	previewWidth  int
	fetchDebounce *time.Timer
	saveDebounce  *time.Timer
	prevCursorHost string // track cursor host to detect changes
}

type filteredItem struct {
	item         Item
	matchIndices []int
}

// clearStatusMsg is sent after the status flash expires.
type clearStatusMsg struct{}

// hostInfoMsg delivers async SSH fetch results.
type hostInfoMsg struct {
	host string
	info *hostinfo.HostInfo
	gen  uint64
}

// fetchDebounceMsg triggers a debounced fetch.
type fetchDebounceMsg struct {
	host string
	gen  uint64
}

// saveDebounceMsg triggers a debounced cache save.
type saveDebounceMsg struct{}

func newModel(items []Item, opts SelectOpts) model {
	m := model{
		items:          items,
		hostCount:      opts.HostCount,
		containerCount: opts.ContainerCount,
		hasScanned:     opts.HasScanned,
		infoFetcher:    opts.InfoFetcher,
		infoCache:      opts.InfoCache,
		infoMem:        make(map[string]*hostinfo.HostInfo),
	}

	// Pre-load disk cache into memory
	if m.infoCache != nil {
		for k, v := range m.infoCache.Hosts {
			m.infoMem[k] = v
		}
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

	case hostInfoMsg:
		if msg.gen != m.fetchGen {
			return m, nil // stale result
		}
		m.loadingHost = ""
		m.infoMem[msg.host] = msg.info
		if m.infoCache != nil {
			m.infoCache.Set(msg.host, msg.info)
			m.infoDirty = true
			// Debounced save: 500ms
			return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return saveDebounceMsg{} })
		}
		return m, nil

	case fetchDebounceMsg:
		if msg.gen != m.fetchGen {
			return m, nil // stale debounce
		}
		return m, m.doFetch(msg.host)

	case saveDebounceMsg:
		if m.infoCache != nil && m.infoDirty {
			_ = m.infoCache.Save()
			m.infoDirty = false
		}
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
		case "tab":
			if m.width >= 100 && m.infoFetcher != nil {
				m.previewOpen = !m.previewOpen
				m.recalcWidths()
				if m.previewOpen {
					return m, m.triggerPreviewFetch()
				}
			}
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
				if !m.filtered[m.cursor].item.IsNested && !strings.Contains(alias, "::") {
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
				if m.previewOpen {
					return m, m.triggerPreviewFetch()
				}
			}
		case "down", "ctrl+n", "ctrl+j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				if m.previewOpen {
					return m, m.triggerPreviewFetch()
				}
			}
		case "backspace":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.filter()
				m.cursor = 0
				if m.previewOpen {
					return m, m.triggerPreviewFetch()
				}
			}
		default:
			if len(msg.String()) == 1 {
				m.query += msg.String()
				m.filter()
				m.cursor = 0
				if m.previewOpen {
					return m, m.triggerPreviewFetch()
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.previewOpen && m.width < 100 {
			m.previewOpen = false
		}
		m.recalcWidths()
	}
	return m, nil
}

// cursorHost returns the SSH host alias for the current cursor position.
// For containers (alias contains "::"), returns the parent host.
func (m model) cursorHost() string {
	if m.cursor >= len(m.filtered) {
		return ""
	}
	alias := m.filtered[m.cursor].item.Alias
	if idx := strings.Index(alias, "::"); idx != -1 {
		return alias[:idx]
	}
	return alias
}

// triggerPreviewFetch starts a debounced fetch for the current cursor's host.
func (m *model) triggerPreviewFetch() tea.Cmd {
	host := m.cursorHost()
	if host == "" {
		return nil
	}

	// Check in-memory cache first (no debounce needed)
	if info, ok := m.infoMem[host]; ok {
		ttl := hostinfo.DefaultTTL
		if info.Error != "" {
			ttl = hostinfo.ErrorTTL
		}
		if time.Since(info.FetchedAt) <= ttl {
			m.loadingHost = ""
			m.prevCursorHost = host
			return nil
		}
	}

	// Same host as before — already fetching
	if host == m.prevCursorHost && m.loadingHost == host {
		return nil
	}
	m.prevCursorHost = host

	// Cancel previous fetch
	if m.cancelFetch != nil {
		m.cancelFetch()
		m.cancelFetch = nil
	}

	m.fetchGen++
	m.loadingHost = host
	gen := m.fetchGen

	// 200ms debounce
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return fetchDebounceMsg{host: host, gen: gen}
	})
}

// doFetch starts the actual SSH fetch.
func (m *model) doFetch(host string) tea.Cmd {
	if m.infoFetcher == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	m.cancelFetch = cancel
	gen := m.fetchGen

	return func() tea.Msg {
		info, _ := m.infoFetcher.FetchAsync(ctx, host)
		if info == nil {
			info = &hostinfo.HostInfo{
				Hostname:  host,
				Error:     "fetch failed",
				FetchedAt: time.Now(),
			}
		}
		return hostInfoMsg{host: host, info: info, gen: gen}
	}
}

func (m *model) recalcWidths() {
	if m.previewOpen && m.width >= 100 {
		m.listWidth = m.width * 60 / 100
		m.previewWidth = m.width - m.listWidth
	} else {
		m.listWidth = m.width
		m.previewWidth = 0
	}
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

	effectiveWidth := m.width
	if m.previewOpen && m.listWidth > 0 {
		effectiveWidth = m.listWidth
	}

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
	if effectiveWidth > 0 {
		maxLabel = effectiveWidth - 4
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

	listContent := header + "\n" + strings.Join(lines, "\n") + hint

	// Split-pane with preview (fzf-style line-by-line rendering)
	if m.previewOpen && m.previewWidth > 0 {
		host := m.cursorHost()
		var info *hostinfo.HostInfo
		loading := false
		if host != "" {
			if cached, ok := m.infoMem[host]; ok {
				info = cached
			}
			if info == nil && m.loadingHost == host {
				loading = true
			}
		}

		paneH := m.height
		if paneH <= 0 {
			paneH = 24
		}

		innerW := m.previewWidth - 4 // 2 border + 2 visual padding
		if innerW < 1 {
			innerW = 1
		}
		pvLines := previewLines(info, loading, innerW)

		listLines := strings.Split(listContent, "\n")

		borderStyle := lipgloss.NewStyle().Foreground(DimColor)
		hBar := strings.Repeat("─", m.previewWidth-2)
		borderL := borderStyle.Render("│")
		borderR := borderStyle.Render("│")

		// Column where the preview pane starts (1-based for ANSI CHA)
		previewCol := m.listWidth + 1

		var out strings.Builder
		for i := 0; i < paneH; i++ {
			if i > 0 {
				out.WriteByte('\n')
			}

			// Left: list content (no padding — cursor positioning handles alignment)
			if i < len(listLines) {
				out.WriteString(listLines[i])
			}

			// Clear stale content between left end and preview border
			out.WriteString("\x1b[K")

			// Move cursor to preview column
			fmt.Fprintf(&out, "\x1b[%dG", previewCol)

			// Right: preview pane with border
			switch {
			case i == 0:
				out.WriteString(borderStyle.Render("╭" + hBar + "╮"))
			case i == paneH-1:
				out.WriteString(borderStyle.Render("╰" + hBar + "╯"))
			default:
				contentIdx := i - 1
				content := ""
				if contentIdx < len(pvLines) {
					content = pvLines[contentIdx]
				}
				out.WriteString(borderL)
				out.WriteString(padRight(content, m.previewWidth-2))
				out.WriteString(borderR)
			}
		}

		return out.String()
	}

	return listContent
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
	if m.previewOpen && m.listWidth > 0 {
		w = m.listWidth
	}
	if w <= 0 {
		w = 80
	}

	// Hide hint bar on very short terminals
	if m.height > 0 && m.height < 15 {
		return ""
	}

	isContainer := false
	if m.cursor < len(m.filtered) {
		isContainer = m.filtered[m.cursor].item.IsNested ||
			strings.Contains(m.filtered[m.cursor].item.Alias, "::")
	}

	canPreview := m.infoFetcher != nil && w >= 100

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

	previewHint := ""
	if canPreview {
		if m.previewOpen {
			previewHint = " · tab close"
		} else {
			previewHint = " · tab preview"
		}
	}

	// Full hints
	if isContainer {
		return "\n" + StyleDim.Render("  ^j/k up/down · enter connect · ^r rename · ^f star · ^d delete" + previewHint + " · esc quit")
	}
	return "\n" + StyleDim.Render("  ^j/k up/down · enter connect · ^r rename · ^s scan · ^f star · ^d delete" + previewHint + " · esc quit")
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
