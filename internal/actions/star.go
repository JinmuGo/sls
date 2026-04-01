package actions

import (
	"fmt"
	"os"

	"github.com/jinmugo/sls/internal/favorites"
)

// Star toggles the favorite status of a host or container.
func Star(favStore *favorites.Store, alias string) error {
	if favStore.IsFavorite(alias) {
		if err := favStore.Remove(alias); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "✓ Unfavorited \033[36m%s\033[0m\n", alias)
	} else {
		if err := favStore.Add(alias); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "✓ Favorited ⭐︎\033[36m%s\033[0m\n", alias)
	}
	return nil
}
