package actions

import (
	"fmt"
	"os"
	"time"

	"github.com/jinmugo/sls/internal/container"
	"github.com/jinmugo/sls/internal/finder"
)

// Scan discovers containers on a host, lets the user select and rename them,
// then saves to cache. Returns the number of containers added and any error.
func Scan(hostAlias string, cache *container.Cache, timeout time.Duration) (int, error) {
	fmt.Fprintf(os.Stderr, "Scanning containers on %s...\n", hostAlias)

	containers, err := container.Discover(hostAlias, timeout, false)
	if err != nil {
		return 0, err
	}

	if len(containers) == 0 {
		return 0, nil
	}

	fmt.Fprintf(os.Stderr, "Found %d container(s). Select which ones to add:\n", len(containers))

	// Exclude containers that are already registered for this host
	existing := cache.GetContainers(hostAlias)
	existingSet := make(map[string]bool, len(existing))
	for _, c := range existing {
		existingSet[c.Name] = true
	}

	var selectItems []finder.Item
	for _, c := range containers {
		if existingSet[c.Name] {
			continue
		}
		selectItems = append(selectItems, finder.Item{
			Label: c.Name,
			Alias: c.Name,
		})
	}

	if len(selectItems) == 0 {
		fmt.Fprintln(os.Stderr, "All containers are already registered.")
		return 0, nil
	}

	selected, err := finder.SelectMulti(selectItems, "select containers")
	if err != nil {
		return 0, err
	}
	if selected == nil {
		return 0, nil
	}
	if len(selected) == 0 {
		fmt.Fprintln(os.Stderr, "No containers selected.")
		return 0, nil
	}

	// Filter selected
	selectedSet := make(map[string]bool)
	for _, name := range selected {
		selectedSet[name] = true
	}
	var picked []container.Container
	for _, c := range containers {
		if selectedSet[c.Name] {
			picked = append(picked, c)
		}
	}

	// Rename step
	renameItems := make([]finder.RenameItem, len(picked))
	for i, c := range picked {
		renameItems[i] = finder.RenameItem{
			OriginalName: c.Name,
			DisplayInfo:  c.Image,
		}
	}

	fmt.Fprintf(os.Stderr, "Set names (enter to keep original):\n")
	nameMap, err := finder.PromptRename(renameItems)
	if err != nil {
		return 0, err
	}
	if nameMap == nil {
		return 0, nil
	}

	for i, c := range picked {
		if newName, ok := nameMap[c.Name]; ok && newName != c.Name {
			picked[i].Alias = newName
		}
	}

	// Append new containers to existing ones (don't replace)
	cache.Update(hostAlias, append(cache.GetContainers(hostAlias), picked...))
	if err := cache.Save(); err != nil {
		return 0, fmt.Errorf("save cache: %w", err)
	}

	return len(picked), nil
}
