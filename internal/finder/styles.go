package finder

import "github.com/charmbracelet/lipgloss"

// Color palette for the sls TUI.
var (
	// Accent — cursor bar, active selection
	AccentColor = lipgloss.Color("#E9672B")
	// Prompt — input prompt text
	PromptColor = lipgloss.Color("6")
	// Match — fuzzy match highlights
	MatchColor = lipgloss.Color("3")
	// Dim — hints, counts, secondary info
	DimColor = lipgloss.Color("8")
	// Success — ✓ confirmations, checked items
	SuccessColor = lipgloss.Color("2")
	// Error — ✗ errors, delete warnings
	ErrorColor = lipgloss.Color("1")
	// Warning — ⚠ warnings
	WarningColor = lipgloss.Color("3")
	// CursorBg — subtle dark background for selected row
	CursorBgColor = lipgloss.Color("237")
)

// Shared styles used across all TUI components.
var (
	StylePrompt    = lipgloss.NewStyle().Foreground(PromptColor)
	StylePromptBold = lipgloss.NewStyle().Foreground(PromptColor).Bold(true)
	StyleCursorBar = lipgloss.NewStyle().Background(AccentColor)
	StyleCursorLabel = lipgloss.NewStyle().Background(CursorBgColor)
	StyleMatch     = lipgloss.NewStyle().Foreground(MatchColor)
	StyleDim       = lipgloss.NewStyle().Foreground(DimColor)
	StyleSuccess   = lipgloss.NewStyle().Foreground(SuccessColor)
	StyleError     = lipgloss.NewStyle().Foreground(ErrorColor)
	StyleCheck     = lipgloss.NewStyle().Foreground(SuccessColor)
	StyleUncheck   = lipgloss.NewStyle().Foreground(DimColor)
	StyleInput     = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	StylePreviewBorder = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(DimColor)
)
