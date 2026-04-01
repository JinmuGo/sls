package finder

import (
	"strings"
	"testing"
)

func TestMultiModelPreChecked(t *testing.T) {
	items := []Item{
		{Label: "nginx", Alias: "nginx"},
		{Label: "postgres", Alias: "postgres"},
		{Label: "redis", Alias: "redis"},
	}

	m := newMultiModel(items, "select")
	// Simulate what SelectMulti does with preChecked
	m.checked["nginx"] = true
	m.checked["redis"] = true

	// Verify checked state
	if !m.checked["nginx"] {
		t.Error("expected nginx to be pre-checked")
	}
	if m.checked["postgres"] {
		t.Error("expected postgres to NOT be pre-checked")
	}
	if !m.checked["redis"] {
		t.Error("expected redis to be pre-checked")
	}

	// Verify View renders checked items with ◉
	view := m.View()
	lines := strings.Split(view, "\n")

	// Header should show "2/3 selected"
	if !strings.Contains(lines[0], "2/3 selected") {
		t.Errorf("expected header to show '2/3 selected', got: %s", lines[0])
	}

	// Find nginx line — should have ◉
	foundCheckedNginx := false
	foundUncheckedPostgres := false
	for _, line := range lines {
		stripped := stripAnsiForTest(line)
		if strings.Contains(stripped, "nginx") && strings.Contains(stripped, "◉") {
			foundCheckedNginx = true
		}
		if strings.Contains(stripped, "postgres") && strings.Contains(stripped, "○") {
			foundUncheckedPostgres = true
		}
	}
	if !foundCheckedNginx {
		t.Errorf("expected nginx to render with ◉ (checked), view:\n%s", view)
	}
	if !foundUncheckedPostgres {
		t.Errorf("expected postgres to render with ○ (unchecked), view:\n%s", view)
	}
}

func TestMultiToggle(t *testing.T) {
	items := []Item{
		{Label: "nginx", Alias: "nginx"},
		{Label: "postgres", Alias: "postgres"},
	}

	m := newMultiModel(items, "select")

	// Space toggles current item
	updated, _ := m.Update(makeKeyMsg(" "))
	result := updated.(multiModel)
	if !result.checked["nginx"] {
		t.Error("expected nginx to be checked after space")
	}

	// Space again unchecks
	updated, _ = result.Update(makeKeyMsg(" "))
	result = updated.(multiModel)
	if result.checked["nginx"] {
		t.Error("expected nginx to be unchecked after second space")
	}
}

func TestMultiSelectAll(t *testing.T) {
	items := []Item{
		{Label: "a", Alias: "a"},
		{Label: "b", Alias: "b"},
		{Label: "c", Alias: "c"},
	}

	m := newMultiModel(items, "select")

	// ctrl+a selects all
	updated, _ := m.Update(makeKeyMsg("ctrl+a"))
	result := updated.(multiModel)
	if len(result.checked) != 3 {
		t.Errorf("expected 3 checked, got %d", len(result.checked))
	}

	// ctrl+a again deselects all
	updated, _ = result.Update(makeKeyMsg("ctrl+a"))
	result = updated.(multiModel)
	if len(result.checked) != 0 {
		t.Errorf("expected 0 checked, got %d", len(result.checked))
	}
}

func TestMultiEnterReturnsChecked(t *testing.T) {
	items := []Item{
		{Label: "nginx", Alias: "nginx"},
		{Label: "postgres", Alias: "postgres"},
	}

	m := newMultiModel(items, "select")
	m.checked["postgres"] = true

	updated, _ := m.Update(makeKeyMsg("enter"))
	result := updated.(multiModel)
	if result.quitting != true {
		t.Error("expected quitting after enter")
	}
	if !result.checked["postgres"] {
		t.Error("expected postgres to remain checked")
	}
	if result.checked["nginx"] {
		t.Error("expected nginx to remain unchecked")
	}
}

func TestMultiEscCancels(t *testing.T) {
	items := []Item{
		{Label: "nginx", Alias: "nginx"},
	}

	m := newMultiModel(items, "select")
	m.checked["nginx"] = true

	updated, _ := m.Update(makeKeyMsg("esc"))
	result := updated.(multiModel)
	if !result.cancelled {
		t.Error("expected cancelled after esc")
	}
}

// makeKeyMsg is defined in finder_test.go (same package)

// stripAnsiForTest removes ANSI escape codes for assertion matching.
func stripAnsiForTest(s string) string {
	var result strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
