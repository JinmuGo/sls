package container

import "testing"

func TestDisplayName(t *testing.T) {
	t.Run("returns alias when set", func(t *testing.T) {
		c := Container{Name: "nginx", Alias: "web"}
		if c.DisplayName() != "web" {
			t.Errorf("expected 'web', got %q", c.DisplayName())
		}
	})

	t.Run("returns name when alias empty", func(t *testing.T) {
		c := Container{Name: "nginx"}
		if c.DisplayName() != "nginx" {
			t.Errorf("expected 'nginx', got %q", c.DisplayName())
		}
	})
}

func TestKey(t *testing.T) {
	c := Container{Name: "nginx", Host: "prod"}
	expected := "prod" + KeySep + "nginx"
	if c.Key() != expected {
		t.Errorf("expected %q, got %q", expected, c.Key())
	}
}
