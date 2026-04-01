package actions

import (
	"fmt"
	"os"

	"github.com/jinmugo/sls/internal/config"
	"github.com/jinmugo/sls/internal/container"
	"github.com/jinmugo/sls/internal/favorites"
	"github.com/jinmugo/sls/internal/finder"
	sshconfig "github.com/kevinburke/ssh_config"
)

// RenameHost renames an SSH host alias in the config, favorites, and container cache.
func RenameHost(hostAlias string, favStore *favorites.Store, cache *container.Cache) (string, error) {
	renameItems := []finder.RenameItem{{
		OriginalName: hostAlias,
		DisplayInfo:  "SSH host",
	}}

	nameMap, err := finder.PromptRename(renameItems)
	if err != nil {
		return "", err
	}
	if nameMap == nil {
		return "", nil
	}

	newName, ok := nameMap[hostAlias]
	if !ok || newName == hostAlias {
		return "", nil
	}

	// Rename in SSH config
	cfg, path, err := config.LoadAST("")
	if err != nil {
		return "", fmt.Errorf("load SSH config: %w", err)
	}

	h, _ := config.FindHost(cfg, hostAlias)
	if h == nil {
		return "", fmt.Errorf("host %s not found in SSH config", hostAlias)
	}

	newPattern, err := sshconfig.NewPattern(newName)
	if err != nil {
		return "", fmt.Errorf("invalid alias %q: %w", newName, err)
	}
	h.Patterns[0] = newPattern

	if err := config.SaveAST(cfg, path); err != nil {
		return "", fmt.Errorf("save SSH config: %w", err)
	}

	// Cascade rename to favorites
	if favStore != nil {
		if favStore.IsFavorite(hostAlias) {
			favStore.Remove(hostAlias)
			favStore.Add(newName)
		}
		// Also rename any container favorites
		if cache != nil {
			for _, c := range cache.GetContainers(hostAlias) {
				oldKey := hostAlias + container.KeySep + c.Name
				newKey := newName + container.KeySep + c.Name
				if favStore.IsFavorite(oldKey) {
					favStore.Remove(oldKey)
					favStore.Add(newKey)
				}
			}
		}
	}

	// Cascade rename to container cache
	if cache != nil {
		cache.RenameHost(hostAlias, newName)
		cache.Save()
	}

	fmt.Fprintf(os.Stderr, "✓ Renamed: %s → \033[36m%s\033[0m\n", hostAlias, newName)
	return newName, nil
}

// RenameContainer renames a container's display alias in the cache.
func RenameContainer(cache *container.Cache, hostAlias, containerName string) (string, error) {
	containers := cache.GetContainers(hostAlias)
	if len(containers) == 0 {
		return "", fmt.Errorf("no cached containers for %s", hostAlias)
	}

	var found *container.Container
	var foundIdx int
	for i, c := range containers {
		if c.Name == containerName {
			found = &containers[i]
			foundIdx = i
			break
		}
	}
	if found == nil {
		return "", fmt.Errorf("container %s not found on %s", containerName, hostAlias)
	}

	renameItems := []finder.RenameItem{{
		OriginalName: found.DisplayName(),
		DisplayInfo:  found.Image,
	}}

	nameMap, err := finder.PromptRename(renameItems)
	if err != nil {
		return "", err
	}
	if nameMap == nil {
		return "", nil
	}

	newName := ""
	if n, ok := nameMap[found.DisplayName()]; ok {
		newName = n
		containers[foundIdx].Alias = n
		cache.Update(hostAlias, containers)
		if err := cache.Save(); err != nil {
			return "", fmt.Errorf("save cache: %w", err)
		}
		fmt.Fprintf(os.Stderr, "✓ Renamed: %s → \033[36m%s\033[0m\n", containerName, n)
	}

	return newName, nil
}
