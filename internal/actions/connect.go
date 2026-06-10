package actions

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jinmugo/sls/internal/container"
	"github.com/jinmugo/sls/internal/favorites"
	"github.com/jinmugo/sls/internal/runner"
)

// Connect connects to a host or container. For containers, it tries the cached
// shell first, then detects available shells via "command -v" (bash → sh → ash).
// The working shell path is saved to cache for next time.
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
		// Only re-detect when the shell or connection failed to *start*. A normal
		// interactive session that simply exited non-zero (e.g. the last command
		// failed) must not trigger a surprise reconnect or poison the cache.
		if !shellStartFailed(err) {
			return err
		}
		// Cached shell failed to start — fall through to detection
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

// detectAndConnect finds a shell via "command -v" and connects. Returns the detected shell path.
func detectAndConnect(hostAlias, containerName string) (string, error) {
	// Build a single "command -v" query for all candidates
	// e.g. "command -v bash || command -v sh || command -v ash"
	var parts []string
	for _, name := range container.ShellCandidates {
		parts = append(parts, fmt.Sprintf("command -v %s", name))
	}
	query := strings.Join(parts, " || ")

	// Try detecting via "sh -c". Use Probe (non-interactive, with timeout) and
	// single-quote the inner command so the remote shell doesn't split on "||".
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	out, err := runner.Probe(ctx, hostAlias, fmt.Sprintf("docker exec %s sh -c '%s'", containerName, query))
	cancel()
	if err == nil {
		shellPath := strings.TrimSpace(string(out))
		if shellPath != "" {
			// The probe already confirmed this shell exists, so a non-start
			// failure means the session ran and ended — the shell works.
			connErr := runner.SSHWithCmd(hostAlias, []string{"docker", "exec", "-it", containerName, shellPath})
			if !shellStartFailed(connErr) {
				return shellPath, nil
			}
		}
	}

	// Fallback: probe each well-known path with a timeout, then connect only once confirmed.
	// This avoids hanging on paths that cause docker exec to stall rather than fail fast.
	for _, name := range container.ShellCandidates {
		for _, prefix := range []string{"/bin/", "/usr/bin/", "/usr/local/bin/"} {
			path := prefix + name
			ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
			_, probeErr := runner.Probe(ctx2, hostAlias, fmt.Sprintf("docker exec %s test -x %s", containerName, path))
			cancel2()
			if probeErr != nil {
				continue
			}
			// `test -x` already confirmed the path is executable, so treat a
			// non-start failure as a completed session rather than a missing shell.
			connErr := runner.SSHWithCmd(hostAlias, []string{"docker", "exec", "-it", containerName, path})
			if !shellStartFailed(connErr) {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("no shell available in container %s on %s", containerName, hostAlias)
}

// shellStartFailed reports whether an ssh / docker-exec error means the shell or
// connection failed to start, as opposed to a normal interactive session that
// exited with a non-zero status. Docker exits 125 (daemon/container error), 126
// (command not executable) or 127 (command not found); ssh exits 255 on
// connection failure; -1 means the process could not be started or was signalled.
func shellStartFailed(err error) bool {
	if err == nil {
		return false
	}
	switch runner.ExitCode(err) {
	case 125, 126, 127, 255, -1:
		return true
	default:
		return false
	}
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

	cache.RefreshExisting(hostAlias, containers)
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
