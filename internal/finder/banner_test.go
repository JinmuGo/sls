package finder

import (
	"strings"
	"testing"
)

func TestUpdateBannerRendered(t *testing.T) {
	items := []Item{
		{Label: "alpha", Alias: "alpha"},
		{Label: "beta", Alias: "beta"},
	}

	banner := "⬆ sls v1.1.2 available · run 'sls update'"

	t.Run("shown when set", func(t *testing.T) {
		m := newModel(items, SelectOpts{UpdateBanner: banner})
		m.width = 80
		m.height = 24
		if !strings.Contains(m.View(), "v1.1.2 available") {
			t.Errorf("View() did not include the update banner:\n%s", m.View())
		}
	})

	t.Run("absent when empty", func(t *testing.T) {
		m := newModel(items, SelectOpts{})
		m.width = 80
		m.height = 24
		if strings.Contains(m.View(), "available") {
			t.Errorf("View() unexpectedly showed an update banner:\n%s", m.View())
		}
	})
}
