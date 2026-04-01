package actions

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jinmugo/sls/internal/container"
	"github.com/jinmugo/sls/internal/favorites"
	"github.com/jinmugo/sls/internal/runner"
)

// Connect connects to a host or container. For containers, it tries the cached
// shell first, then falls back through /bin/sh → /bin/bash → /bin/ash.
// The working shell is saved to cache for next time.
func Connect(alias string, extraSSHArgs []string, favStore *favorites.Store, cache *container.Cache) error {
	if strings.Contains(alias, container.KeySep) {
		parts := strings.SplitN(alias, container.KeySep, 2)
		hostAlias, containerName := parts[0], parts[1]
		if favStore != nil {
			favStore.Increment(hostAlias)
		}
		return connectToContainer(hostAlias, containerName, cache)
	}

	if favStore != nil {
		favStore.Increment(alias)
	}
	return runner.SSH(alias, extraSSHArgs)
}

func connectToContainer(hostAlias, containerName string, cache *container.Cache) error {
	// Look up cached shell info
	cachedShell := container.ShellUnknown
	if cache != nil {
		for _, c := range cache.GetContainers(hostAlias) {
			if c.Name == containerName {
				cachedShell = c.Shell
				break
			}
		}
	}

	// If we already know there's no shell, tell the user immediately
	if cachedShell == container.ShellNone {
		fmt.Fprintf(os.Stderr, "Container %q on %s has no shell (previously detected).\n", containerName, hostAlias)
		return fmt.Errorf("no shell available in container %s", containerName)
	}

	// If we have a cached shell, try it directly
	if cachedShell != container.ShellUnknown {
		err := runner.SSHWithCmd(hostAlias, []string{"docker", "exec", "-it", containerName, cachedShell})
		if err == nil {
			return nil
		}
		// Cached shell failed — fall through to detection
		fmt.Fprintf(os.Stderr, "Cached shell %s failed, detecting available shells...\n", cachedShell)
	}

	// Try each shell candidate
	shell, err := detectAndConnect(hostAlias, containerName)
	if err != nil {
		// No shell worked — try refresh-on-miss before giving up
		return refreshAndRetry(hostAlias, containerName, cache)
	}

	// Save detected shell to cache
	saveShellToCache(cache, hostAlias, containerName, shell)
	return nil
}

// detectAndConnect tries each shell candidate. Returns the working shell path.
func detectAndConnect(hostAlias, containerName string) (string, error) {
	for _, shell := range container.ShellCandidates {
		err := runner.SSHWithCmd(hostAlias, []string{"docker", "exec", "-it", containerName, shell})
		if err == nil {
			return shell, nil
		}
	}

	return "", fmt.Errorf("no shell available in container %s on %s", containerName, hostAlias)
}

func refreshAndRetry(hostAlias, containerName string, cache *container.Cache) error {
	fmt.Fprintf(os.Stderr, "Container %s may have stopped or has no shell. Refreshing...\n", containerName)

	if cache == nil {
		return reportNoShell(hostAlias, containerName, cache)
	}

	containers, discErr := container.Discover(hostAlias, 10*time.Second, false)
	if discErr != nil {
		return fmt.Errorf("container %s is not running on %s. Discovery also failed: %w", containerName, hostAlias, discErr)
	}

	cache.MergeUpdate(hostAlias, containers)
	cache.Save()

	// Check if container still exists
	found := false
	for _, c := range containers {
		if c.Name == containerName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("container %s is not running on %s", containerName, hostAlias)
	}

	// Container exists but all shells failed
	return reportNoShell(hostAlias, containerName, cache)
}

func reportNoShell(hostAlias, containerName string, cache *container.Cache) error {
	// Mark as "no shell" in cache so we don't retry every time
	saveShellToCache(cache, hostAlias, containerName, container.ShellNone)

	fmt.Fprintf(os.Stderr, "\n\033[33mNo shell found\033[0m in container %q on %s.\n", containerName, hostAlias)
	fmt.Fprintf(os.Stderr, "This is common with distroless or scratch-based images.\n")
	return fmt.Errorf("no shell available in container %s", containerName)
}

func saveShellToCache(cache *container.Cache, hostAlias, containerName, shell string) {
	if cache == nil {
		return
	}
	containers := cache.GetContainers(hostAlias)
	for i, c := range containers {
		if c.Name == containerName {
			containers[i].Shell = shell
			cache.Update(hostAlias, containers)
			cache.Save()
			return
		}
	}
}
