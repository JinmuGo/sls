package finder

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jinmugo/sls/internal/hostinfo"
)

// previewLines returns the content lines for the preview panel (no border).
// innerW is the usable text width inside the border.
func previewLines(info *hostinfo.HostInfo, loading bool, innerW int) []string {
	if innerW < 10 {
		return nil
	}

	var lines []string

	if loading {
		lines = append(lines, "")
		lines = append(lines, StyleDim.Render("  fetching..."))
		lines = append(lines, "")
		return lines
	}

	if info == nil {
		lines = append(lines, "")
		lines = append(lines, StyleDim.Render("  no data"))
		lines = append(lines, "")
		return lines
	}

	if info.Error != "" {
		lines = append(lines, "")
		lines = append(lines, StyleError.Render("  unreachable"))
		lines = append(lines, StyleDim.Render("  "+truncStr(info.Error, innerW-2)))
		lines = append(lines, "")
		return lines
	}

	// Title
	title := info.Hostname
	if len(title) > innerW {
		title = title[:innerW-1] + "…"
	}
	lines = append(lines, StylePromptBold.Render("  "+title))
	lines = append(lines, StyleDim.Render("  "+strings.Repeat("─", min(innerW-2, 30))))

	// System info
	if info.OS != "" {
		lines = append(lines, renderKV("OS", info.OS, innerW))
	}
	if info.Kernel != "" {
		lines = append(lines, renderKV("Kernel", info.Kernel, innerW))
	}
	if info.Uptime != "" {
		lines = append(lines, renderKV("Uptime", info.Uptime, innerW))
	}

	// Disk
	if info.DiskPct != "" {
		lines = append(lines, StyleDim.Render("  "+strings.Repeat("─", min(innerW-2, 30))))
		lines = append(lines, renderKV("Disk", info.DiskUsed+" / "+info.DiskTotal, innerW))
		lines = append(lines, renderBar(info.DiskPct, innerW))
	}

	// Memory (used / total, same order as Disk)
	if info.MemPct != "" {
		lines = append(lines, StyleDim.Render("  "+strings.Repeat("─", min(innerW-2, 30))))
		lines = append(lines, renderKV("Mem", info.MemUsed+" / "+info.MemTotal, innerW))
		lines = append(lines, renderBar(info.MemPct, innerW))
	} else if info.Memory != "" {
		lines = append(lines, StyleDim.Render("  "+strings.Repeat("─", min(innerW-2, 30))))
		lines = append(lines, renderKV("Mem", info.Memory, innerW))
	}

	// Fetched age
	if !info.FetchedAt.IsZero() {
		lines = append(lines, "")
		d := time.Since(info.FetchedAt)
		var age string
		switch {
		case d < time.Minute:
			age = fmt.Sprintf("%ds ago", int(d.Seconds()))
		case d < time.Hour:
			age = fmt.Sprintf("%dm ago", int(d.Minutes()))
		default:
			age = fmt.Sprintf("%dh ago", int(d.Hours()))
		}
		lines = append(lines, StyleDim.Render("  fetched "+age))
	}

	return lines
}

// renderPreview composes the preview box as a single string (for testing compatibility).
func renderPreview(info *hostinfo.HostInfo, loading bool, width, height int) string {
	if width < 20 {
		return ""
	}
	innerW := width - 4 // 2 border + 2 visual padding
	lines := previewLines(info, loading, innerW)
	return composeBorderedBox(lines, width, height)
}

// composeBorderedBox wraps content lines in a rounded border box.
func composeBorderedBox(contentLines []string, width, height int) string {
	innerW := width - 2 // space between left and right border chars
	contentH := height - 2 // space between top and bottom border

	borderStyle := lipgloss.NewStyle().Foreground(DimColor)
	hBar := strings.Repeat("─", innerW)

	var out strings.Builder
	out.WriteString(borderStyle.Render("╭" + hBar + "╮"))

	for i := 0; i < contentH; i++ {
		out.WriteByte('\n')
		content := ""
		if i < len(contentLines) {
			content = contentLines[i]
		}
		content = padRight(content, innerW)
		out.WriteString(borderStyle.Render("│"))
		out.WriteString(content)
		out.WriteString(borderStyle.Render("│"))
	}

	out.WriteByte('\n')
	out.WriteString(borderStyle.Render("╰" + hBar + "╯"))

	return out.String()
}

// padRight pads (or truncates) a string to exactly the given visible width.
func padRight(s string, width int) string {
	vis := lipgloss.Width(s)
	if vis > width {
		s = lipgloss.NewStyle().MaxWidth(width).Render(s)
		vis = lipgloss.Width(s)
	}
	if vis < width {
		s += strings.Repeat(" ", width-vis)
	}
	return s
}

func renderKV(key, value string, maxW int) string {
	label := fmt.Sprintf("  %-7s ", key)
	remaining := maxW - len(label)
	if remaining < 0 {
		remaining = 0
	}
	val := value
	if len(val) > remaining {
		val = val[:remaining-1] + "…"
	}
	return StyleDim.Render(label) + val
}

func renderBar(pctStr string, maxW int) string {
	pct := 0
	clean := strings.TrimSuffix(pctStr, "%")
	if v, err := strconv.Atoi(clean); err == nil {
		pct = v
	}

	barW := maxW - 14 // "  ████░░ XX%  "
	if barW < 10 {
		barW = 10
	}

	filled := pct * barW / 100
	if filled > barW {
		filled = barW
	}
	empty := barW - filled

	barStyle := lipgloss.NewStyle().Foreground(AccentColor)
	if pct >= 80 {
		barStyle = lipgloss.NewStyle().Foreground(ErrorColor)
	}

	bar := barStyle.Render(strings.Repeat("█", filled)) +
		StyleDim.Render(strings.Repeat("░", empty))

	return fmt.Sprintf("  %s %3d%%", bar, pct)
}

func truncStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return s[:max-1] + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
