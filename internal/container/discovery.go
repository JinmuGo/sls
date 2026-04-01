package container

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// DiscoverResult holds the result of discovering containers on a single host.
type DiscoverResult struct {
	Host       string
	Containers []Container
	Err        error
}

// Discover runs `docker ps` on a remote host via SSH and returns the list of running containers.
func Discover(hostAlias string, timeout time.Duration, verbose bool) ([]Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		hostAlias,
		`docker ps --format "{{.ID}}|||{{.Names}}|||{{.Image}}|||{{.Status}}"`,
	)

	if verbose {
		fmt.Fprintf(verboseWriter, "[discover] ssh %s docker ps --format ...\n", hostAlias)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("SSH connection to %s timed out after %v", hostAlias, timeout)
		}
		outStr := strings.TrimSpace(string(output))
		if strings.Contains(outStr, "command not found") || strings.Contains(outStr, "not found") {
			return nil, fmt.Errorf("Docker not found on %s", hostAlias)
		}
		if strings.Contains(outStr, "permission denied") || strings.Contains(outStr, "Permission denied") {
			return nil, fmt.Errorf("Docker permission denied on %s. Add your user to the docker group: sudo usermod -aG docker $USER", hostAlias)
		}
		return nil, fmt.Errorf("SSH connection to %s failed: %w", hostAlias, err)
	}

	return parseDockerOutput(hostAlias, string(output), verbose)
}

func parseDockerOutput(hostAlias, output string, verbose bool) ([]Container, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var containers []Container

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.SplitN(line, "|||", 4)
		if len(fields) != 4 {
			if verbose {
				fmt.Fprintf(verboseWriter, "[discover] skipping malformed line: %q\n", line)
			}
			continue
		}

		name := strings.TrimSpace(fields[1])
		if !ValidateName(name) {
			if verbose {
				fmt.Fprintf(verboseWriter, "[discover] skipping container with unsafe name: %q\n", name)
			}
			continue
		}

		containers = append(containers, Container{
			ID:     strings.TrimSpace(fields[0]),
			Name:   name,
			Image:  strings.TrimSpace(fields[2]),
			Status: strings.TrimSpace(fields[3]),
			Host:   hostAlias,
		})
	}

	return containers, nil
}

// DiscoverAll runs discovery on multiple hosts concurrently with a concurrency limit.
func DiscoverAll(hosts []string, timeout time.Duration, concurrency int, verbose bool) []DiscoverResult {
	if concurrency <= 0 {
		concurrency = 10
	}

	var mu sync.Mutex
	results := make([]DiscoverResult, 0, len(hosts))

	g := new(errgroup.Group)
	g.SetLimit(concurrency)

	for _, host := range hosts {
		host := host
		g.Go(func() error {
			containers, err := Discover(host, timeout, verbose)
			mu.Lock()
			results = append(results, DiscoverResult{
				Host:       host,
				Containers: containers,
				Err:        err,
			})
			mu.Unlock()
			return nil // never fail the group; errors are per-host
		})
	}

	g.Wait()
	return results
}
