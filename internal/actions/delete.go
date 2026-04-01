package actions

import (
	"fmt"
	"os"

	"github.com/jinmugo/sls/internal/config"
	"github.com/jinmugo/sls/internal/container"
	"github.com/jinmugo/sls/internal/favorites"
)

// DeleteHost removes a host from SSH config, cleaning up favorites.
// Returns an error if the host has registered containers (must be removed first).
func DeleteHost(hostAlias string, favStore *favorites.Store, cache *container.Cache) error {
	// Check if host has containers
	if cache != nil {
		containers := cache.GetContainers(hostAlias)
		if len(containers) > 0 {
			fmt.Fprintf(os.Stderr, "\033[33mCannot delete %s\033[0m — %d container(s) are registered:\n", hostAlias, len(containers))
			for _, c := range containers {
				fmt.Fprintf(os.Stderr, "  • %s\n", c.DisplayName())
			}
			fmt.Fprintf(os.Stderr, "Remove all containers first (ctrl+d on each), then delete the host.\n")
			return nil
		}
	}

	// Delete from SSH config
	cfg, path, err := config.LoadAST("")
	if err != nil {
		return fmt.Errorf("load SSH config: %w", err)
	}

	if !config.DeleteHost(cfg, hostAlias) {
		return fmt.Errorf("host %s not found in SSH config", hostAlias)
	}

	if err := config.SaveAST(cfg, path); err != nil {
		return fmt.Errorf("save SSH config: %w", err)
	}

	// Clean up favorites
	if favStore != nil {
		favStore.Remove(hostAlias)
	}

	fmt.Fprintf(os.Stderr, "✓ Deleted host \033[31m%s\033[0m from SSH config\n", hostAlias)
	return nil
}

// DeleteContainer removes a container from the cache.
func DeleteContainer(cache *container.Cache, hostAlias, containerName string, favStore *favorites.Store) error {
	containers := cache.GetContainers(hostAlias)
	var remaining []container.Container
	var deletedName string
	for _, c := range containers {
		if c.Name == containerName {
			deletedName = c.DisplayName()
			continue
		}
		remaining = append(remaining, c)
	}

	if deletedName == "" {
		return fmt.Errorf("container %s not found on %s", containerName, hostAlias)
	}

	cache.Update(hostAlias, remaining)
	if err := cache.Save(); err != nil {
		return fmt.Errorf("save cache: %w", err)
	}

	// Clean up favorites for this container
	if favStore != nil {
		key := hostAlias + container.KeySep + containerName
		favStore.Remove(key)
	}

	fmt.Fprintf(os.Stderr, "✓ Removed container \033[31m%s\033[0m from %s\n", deletedName, hostAlias)
	return nil
}
