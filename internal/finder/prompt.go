package finder

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jinmugo/sls/internal/container"
)

// PromptRename asks the user to rename each item. Returns a map of original alias → new name.
// If the user leaves the input empty or unchanged, the original name is kept.
// Returns nil if cancelled.
func PromptRename(items []RenameItem) (map[string]string, error) {
	if len(items) == 0 {
		return nil, nil
	}

	m := newRenameModel(items)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt: %w", err)
	}

	fmt.Fprint(os.Stderr, "\033[A\033[K")

	result := finalModel.(renameModel)
	if result.cancelled {
		return nil, nil
	}
	return result.results, nil
}

// RenameItem represents one item to rename.
type RenameItem struct {
	OriginalName string // Container name from Docker
	DisplayInfo  string // Extra info shown next to the prompt (e.g., image name)
}

type renameModel struct {
	items     []RenameItem
	current   int
	input     string
	results   map[string]string
	cancelled bool
	done      bool
	errMsg    string // validation error
}

func newRenameModel(items []RenameItem) renameModel {
	m := renameModel{
		items:   items,
		results: make(map[string]string),
		input:   items[0].OriginalName,
	}
	return m
}

func (m renameModel) Init() tea.Cmd {
	return nil
}

func (m renameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "enter":
			name := strings.TrimSpace(m.input)
			if name == "" {
				name = m.items[m.current].OriginalName
			}

			// Validate alias
			if !container.ValidateName(name) {
				m.errMsg = "only letters, numbers, dots, hyphens, underscores"
				return m, nil
			}

			m.errMsg = ""
			m.results[m.items[m.current].OriginalName] = name

			m.current++
			if m.current >= len(m.items) {
				m.done = true
				return m, tea.Quit
			}
			// Pre-fill with next item's original name
			m.input = m.items[m.current].OriginalName
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
				m.errMsg = ""
			}
		case "ctrl+u":
			m.input = ""
			m.errMsg = ""
		default:
			if len(msg.String()) == 1 {
				m.input += msg.String()
				m.errMsg = ""
			}
		}
	}
	return m, nil
}

func (m renameModel) View() string {
	if m.done {
		return ""
	}

	item := m.items[m.current]
	progress := StyleDim.Render(fmt.Sprintf("[%d/%d]", m.current+1, len(m.items)))
	info := StyleDim.Render(item.DisplayInfo)

	label := fmt.Sprintf("%s %s %s", progress, info, StylePromptBold.Render("name:"))
	input := StyleInput.Render(m.input) + "█"

	result := label + " " + input

	if m.errMsg != "" {
		result += "\n" + StyleError.Render("  ✗ "+m.errMsg)
	}

	help := StyleDim.Render("  enter confirm · ctrl+u clear · esc cancel")

	return result + "\n" + help
}
