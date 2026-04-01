package finder

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestPreCheckedSurvivesModelCopy verifies that pre-checked state
// survives value-type model copying (Bubbletea uses value receivers).
func TestPreCheckedSurvivesModelCopy(t *testing.T) {
	items := []Item{
		{Label: "nginx", Alias: "nginx"},
		{Label: "postgres", Alias: "postgres"},
	}

	m := newMultiModel(items, "select")
	m.checked["nginx"] = true

	// tea.Model is an interface — storing multiModel (value type) in it makes a copy.
	// But map is a reference type, so checked should be shared.
	var iface tea.Model = m

	result := iface.(multiModel)
	if !result.checked["nginx"] {
		t.Error("pre-checked state lost when stored as tea.Model interface")
	}

	// Simulate Update cycle (enter to quit) — Update returns a copy
	updated, _ := result.Update(makeKeyMsg("enter"))
	final := updated.(multiModel)
	if !final.checked["nginx"] {
		t.Error("pre-checked state lost after Update")
	}
}

// TestPreCheckedViewRendering verifies the View output shows ◉ for pre-checked items.
func TestPreCheckedViewRendering(t *testing.T) {
	items := []Item{
		{Label: "nginx", Alias: "nginx"},
		{Label: "postgres", Alias: "postgres"},
		{Label: "redis", Alias: "redis"},
	}

	m := newMultiModel(items, "select")
	m.checked["nginx"] = true
	m.checked["redis"] = true

	view := m.View()

	// Check that nginx renders with ◉
	if !viewHasCheckedItem(view, "nginx") {
		t.Errorf("nginx should be rendered as checked (◉) in view:\n%s", view)
	}

	// Check that postgres renders with ○
	if viewHasCheckedItem(view, "postgres") {
		t.Errorf("postgres should be rendered as unchecked (○) in view:\n%s", view)
	}

	// Check that redis renders with ◉
	if !viewHasCheckedItem(view, "redis") {
		t.Errorf("redis should be rendered as checked (◉) in view:\n%s", view)
	}
}

// TestPreCheckedMatchesSelectMultiAPI tests the exact API pattern used by scan.go
func TestPreCheckedMatchesSelectMultiAPI(t *testing.T) {
	// This tests the exact code path:
	//   m := newMultiModel(items, prompt)
	//   for _, alias := range preChecked { m.checked[alias] = true }
	// Verifying the model is correct before tea.NewProgram gets it.

	items := []Item{
		{Label: "nginx", Alias: "nginx"},
		{Label: "postgres", Alias: "postgres"},
		{Label: "redis", Alias: "redis"},
	}
	preChecked := []string{"nginx", "redis"}

	m := newMultiModel(items, "select")
	for _, alias := range preChecked {
		m.checked[alias] = true
	}

	// Verify the model state
	if len(m.checked) != 2 {
		t.Fatalf("expected 2 checked items, got %d", len(m.checked))
	}

	// Verify View renders correctly
	view := m.View()
	if !strings.Contains(view, "2/3 selected") {
		t.Errorf("header should show '2/3 selected', view:\n%s", view)
	}

	// Simulate the user pressing enter immediately (accepting pre-selection)
	updated, _ := m.Update(makeKeyMsg("enter"))
	result := updated.(multiModel)

	// The returned checked map should still have nginx and redis
	if !result.checked["nginx"] || !result.checked["redis"] {
		t.Error("pre-checked items should survive enter")
	}
	if result.checked["postgres"] {
		t.Error("postgres should not be checked")
	}
}

// TestPreCheckedThroughTeaProgram runs a real tea.Program with pre-checked items
// and verifies the final model retains the checked state.
func TestPreCheckedThroughTeaProgram(t *testing.T) {
	items := []Item{
		{Label: "nginx", Alias: "nginx"},
		{Label: "postgres", Alias: "postgres"},
		{Label: "redis", Alias: "redis"},
	}

	m := newMultiModel(items, "select")
	m.checked["nginx"] = true
	m.checked["redis"] = true

	// Run through real tea.Program with immediate quit via a custom Cmd
	p := tea.NewProgram(m, tea.WithInput(nil), tea.WithoutRenderer())

	// Send enter key to quit immediately, preserving checked state
	go func() {
		p.Send(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	}()

	finalModel, err := p.Run()
	if err != nil {
		t.Fatalf("tea.Program.Run: %v", err)
	}

	result := finalModel.(multiModel)
	if !result.checked["nginx"] {
		t.Error("nginx should be checked after tea.Program run")
	}
	if result.checked["postgres"] {
		t.Error("postgres should NOT be checked after tea.Program run")
	}
	if !result.checked["redis"] {
		t.Error("redis should be checked after tea.Program run")
	}
}

func viewHasCheckedItem(view, itemName string) bool {
	for _, line := range strings.Split(view, "\n") {
		stripped := stripAnsiForTest(line)
		if strings.Contains(stripped, itemName) && strings.Contains(stripped, "◉") {
			return true
		}
	}
	return false
}
