package finder

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jinmugo/sls/internal/hostinfo"
)

func TestTruncatePlain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"truncate", "hello world", 8, "hello w…"},
		{"one char", "hello", 1, "…"},
		{"empty", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncatePlain(tt.input, tt.maxWidth)
			if got != tt.want {
				t.Errorf("truncatePlain(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
			}
		})
	}
}

// makeKeyMsg creates a tea.KeyMsg for testing.
func makeKeyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg(tea.Key{Type: tea.KeyEnter})
	case "esc":
		return tea.KeyMsg(tea.Key{Type: tea.KeyEscape})
	case "ctrl+c":
		return tea.KeyMsg(tea.Key{Type: tea.KeyCtrlC})
	case "ctrl+r":
		return tea.KeyMsg(tea.Key{Type: tea.KeyCtrlR})
	case "ctrl+s":
		return tea.KeyMsg(tea.Key{Type: tea.KeyCtrlS})
	case "ctrl+d":
		return tea.KeyMsg(tea.Key{Type: tea.KeyCtrlD})
	case "ctrl+f":
		return tea.KeyMsg(tea.Key{Type: tea.KeyCtrlF})
	case "up":
		return tea.KeyMsg(tea.Key{Type: tea.KeyUp})
	case "down":
		return tea.KeyMsg(tea.Key{Type: tea.KeyDown})
	case "backspace":
		return tea.KeyMsg(tea.Key{Type: tea.KeyBackspace})
	case "y":
		return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'y'}})
	case "n":
		return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'n'}})
	default:
		if len(key) == 1 {
			return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(key)})
		}
		return tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(key)})
	}
}

func TestModelFilter(t *testing.T) {
	items := []Item{
		{Label: "my-server", Alias: "my-server"},
		{Label: "⭐︎production", Alias: "production"},
		{Label: "staging", Alias: "staging"},
		{Label: "nas-home", Alias: "nas-home"},
	}

	t.Run("empty query returns all", func(t *testing.T) {
		m := newModel(items, SelectOpts{})
		if len(m.filtered) != len(items) {
			t.Errorf("expected %d items, got %d", len(items), len(m.filtered))
		}
	})

	t.Run("query filters items", func(t *testing.T) {
		m := newModel(items, SelectOpts{})
		m.query = "prod"
		m.filter()
		if len(m.filtered) == 0 {
			t.Error("expected at least one match for 'prod'")
		}
		found := false
		for _, fi := range m.filtered {
			if fi.item.Alias == "production" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'production' to match 'prod'")
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		m := newModel(items, SelectOpts{})
		m.query = "zzzzzzz"
		m.filter()
		if len(m.filtered) != 0 {
			t.Errorf("expected 0 matches, got %d", len(m.filtered))
		}
	})

	t.Run("fuzzy matching works", func(t *testing.T) {
		m := newModel(items, SelectOpts{})
		m.query = "msr"
		m.filter()
		found := false
		for _, fi := range m.filtered {
			if fi.item.Alias == "my-server" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'my-server' to fuzzy-match 'msr'")
		}
	})
}

func TestModelCursorRestore(t *testing.T) {
	items := []Item{
		{Label: "alpha", Alias: "alpha"},
		{Label: "beta", Alias: "beta"},
		{Label: "gamma", Alias: "gamma"},
	}

	m := newModel(items, SelectOpts{RestoreAlias: "beta"})
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", m.cursor)
	}
}

func TestModelDeleteConfirmation(t *testing.T) {
	items := []Item{
		{Label: "server", Alias: "server"},
	}

	t.Run("n cancels delete", func(t *testing.T) {
		m := newModel(items, SelectOpts{})
		m.selected = "server"
		m.confirmDelete = true

		updated, _ := m.Update(makeKeyMsg("n"))
		result := updated.(model)
		if result.confirmDelete {
			t.Error("expected confirmDelete to be false after 'n'")
		}
	})

	t.Run("y confirms delete", func(t *testing.T) {
		m := newModel(items, SelectOpts{})
		m.selected = "server"
		m.confirmDelete = true

		updated, _ := m.Update(makeKeyMsg("y"))
		result := updated.(model)
		if result.action != "delete" {
			t.Errorf("expected action 'delete', got %q", result.action)
		}
		if !result.quitting {
			t.Error("expected quitting to be true after delete confirm")
		}
	})
}

func TestModelCtrlSOnlyOnHosts(t *testing.T) {
	items := []Item{
		{Label: "server", Alias: "server"},
		{Label: "nginx 🐳", Alias: "server::nginx", IsNested: true},
	}

	t.Run("ctrl+s on host triggers scan", func(t *testing.T) {
		m := newModel(items, SelectOpts{})
		m.cursor = 0

		updated, _ := m.Update(makeKeyMsg("ctrl+s"))
		result := updated.(model)
		if result.action != "scan" {
			t.Errorf("expected action 'scan' on host, got %q", result.action)
		}
	})

	t.Run("ctrl+s on container is ignored", func(t *testing.T) {
		m := newModel(items, SelectOpts{})
		m.cursor = 1

		updated, _ := m.Update(makeKeyMsg("ctrl+s"))
		result := updated.(model)
		if result.action == "scan" {
			t.Error("ctrl+s should not trigger scan on container items")
		}
	})
}

func TestModelCursorBounds(t *testing.T) {
	items := []Item{
		{Label: "a", Alias: "a"},
		{Label: "b", Alias: "b"},
	}
	m := newModel(items, SelectOpts{})

	// Can't go above 0
	updated, _ := m.Update(makeKeyMsg("up"))
	result := updated.(model)
	if result.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", result.cursor)
	}

	// Move down
	updated, _ = result.Update(makeKeyMsg("down"))
	result = updated.(model)
	if result.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", result.cursor)
	}

	// Can't go below len-1
	updated, _ = result.Update(makeKeyMsg("down"))
	result = updated.(model)
	if result.cursor != 1 {
		t.Errorf("expected cursor 1 (clamped), got %d", result.cursor)
	}
}

func TestModelEnterAction(t *testing.T) {
	items := []Item{
		{Label: "server", Alias: "server"},
	}
	m := newModel(items, SelectOpts{})

	updated, _ := m.Update(makeKeyMsg("enter"))
	result := updated.(model)
	if result.action != "connect" {
		t.Errorf("expected action 'connect', got %q", result.action)
	}
	if result.selected != "server" {
		t.Errorf("expected selected 'server', got %q", result.selected)
	}
}

func TestModelEsc(t *testing.T) {
	items := []Item{
		{Label: "server", Alias: "server"},
	}
	m := newModel(items, SelectOpts{})

	updated, _ := m.Update(makeKeyMsg("esc"))
	result := updated.(model)
	if !result.cancelled {
		t.Error("expected cancelled after esc")
	}
}

func TestModelTabToggle(t *testing.T) {
	items := []Item{
		{Label: "server", Alias: "server"},
	}

	t.Run("tab ignored without fetcher", func(t *testing.T) {
		m := newModel(items, SelectOpts{})
		m.width = 120
		m.height = 30

		updated, _ := m.Update(makeKeyMsg("tab"))
		result := updated.(model)
		if result.previewOpen {
			t.Error("preview should not open without fetcher")
		}
	})

	t.Run("tab ignored on narrow terminal", func(t *testing.T) {
		m := newModel(items, SelectOpts{InfoFetcher: &mockFetcher{}})
		m.width = 80
		m.height = 30

		updated, _ := m.Update(makeKeyMsg("tab"))
		result := updated.(model)
		if result.previewOpen {
			t.Error("preview should not open on narrow terminal")
		}
	})

	t.Run("tab toggles on wide terminal with fetcher", func(t *testing.T) {
		m := newModel(items, SelectOpts{InfoFetcher: &mockFetcher{}})
		m.width = 120
		m.height = 30

		updated, _ := m.Update(makeKeyMsg("tab"))
		result := updated.(model)
		if !result.previewOpen {
			t.Error("expected preview to open")
		}

		updated2, _ := result.Update(makeKeyMsg("tab"))
		result2 := updated2.(model)
		if result2.previewOpen {
			t.Error("expected preview to close on second tab")
		}
	})
}

func TestCursorHost(t *testing.T) {
	items := []Item{
		{Label: "server", Alias: "server"},
		{Label: "⭐︎nginx", Alias: "server::nginx"},
		{Label: "staging", Alias: "staging"},
	}
	m := newModel(items, SelectOpts{})

	m.cursor = 0
	if got := m.cursorHost(); got != "server" {
		t.Errorf("cursor 0: got %q, want server", got)
	}

	m.cursor = 1
	if got := m.cursorHost(); got != "server" {
		t.Errorf("cursor 1: got %q, want server (parent of container)", got)
	}

	m.cursor = 2
	if got := m.cursorHost(); got != "staging" {
		t.Errorf("cursor 2: got %q, want staging", got)
	}
}

func TestCtrlSBlockedOnContainer(t *testing.T) {
	items := []Item{
		{Label: "⭐︎nginx", Alias: "server::nginx"}, // favorited container, IsNested=false
	}
	m := newModel(items, SelectOpts{})
	m.cursor = 0

	updated, _ := m.Update(makeKeyMsg("ctrl+s"))
	result := updated.(model)
	if result.action == "scan" {
		t.Error("ctrl+s should not trigger scan on container (:: check)")
	}
}

// mockFetcher implements HostInfoFetcher for testing.
type mockFetcher struct{}

func (f *mockFetcher) Get(host string) *hostinfo.HostInfo { return nil }
func (f *mockFetcher) FetchAsync(ctx context.Context, host string) (*hostinfo.HostInfo, error) {
	return &hostinfo.HostInfo{Hostname: host}, nil
}

func TestPreviewViewConsistency(t *testing.T) {
	items := []Item{
		{Label: "⭐︎ jgopi", Alias: "jgopi"},
		{Label: "proxmox", Alias: "proxmox"},
		{Label: "frontend 🐢 (stale)", Alias: "jgopi::frontend", IsNested: true},
		{Label: "oci_atlas_1", Alias: "oci_atlas_1"},
		{Label: "backend 🐢 (stale)", Alias: "oci_atlas_1::backend", IsNested: true, IsLast: true},
		{Label: "RockyLinux", Alias: "RockyLinux"},
		{Label: "HomeServer", Alias: "HomeServer"},
		{Label: "readup-old", Alias: "readup-old"},
	}

	widths := []int{100, 120, 150, 200}

	for _, w := range widths {
		m := newModel(items, SelectOpts{
			HostCount:      5,
			ContainerCount: 2,
			InfoFetcher:    &mockFetcher{},
		})
		m.width = w
		m.height = 30
		m.previewOpen = true
		m.recalcWidths()

		// Verify consistent line count across all cursor positions
		var prevLineCount int
		for cursor := 0; cursor < len(items); cursor++ {
			m.cursor = cursor
			output := m.View()
			lines := strings.Split(output, "\n")

			if cursor == 0 {
				prevLineCount = len(lines)
			}

			if len(lines) != prevLineCount {
				t.Errorf("width=%d cursor=%d: line count changed from %d to %d",
					w, cursor, prevLineCount, len(lines))
			}

			if len(lines) != 30 {
				t.Errorf("width=%d cursor=%d: expected 30 lines, got %d",
					w, cursor, len(lines))
			}
		}
	}
}

func TestPreviewViewStructure(t *testing.T) {
	items := []Item{
		{Label: "server", Alias: "server"},
		{Label: "container", Alias: "server::container", IsNested: true, IsLast: true},
	}

	m := newModel(items, SelectOpts{
		HostCount:   1,
		InfoFetcher: &mockFetcher{},
	})
	m.width = 120
	m.height = 20
	m.previewOpen = true
	m.recalcWidths()

	output := m.View()
	lines := strings.Split(output, "\n")

	// Line count must match paneH
	if len(lines) != 20 {
		t.Errorf("expected 20 lines, got %d", len(lines))
	}

	// First line must contain top border
	if !strings.Contains(lines[0], "╭") || !strings.Contains(lines[0], "╮") {
		t.Error("first line missing top border ╭...╮")
	}

	// Last line must contain bottom border
	if !strings.Contains(lines[len(lines)-1], "╰") || !strings.Contains(lines[len(lines)-1], "╯") {
		t.Error("last line missing bottom border ╰...╯")
	}

	// Middle lines must contain side borders
	for i := 1; i < len(lines)-1; i++ {
		if !strings.Contains(lines[i], "│") {
			t.Errorf("line %d missing side border │", i)
		}
	}
}
