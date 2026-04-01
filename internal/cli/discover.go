package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/jinmugo/sls/internal/actions"
	"github.com/jinmugo/sls/internal/config"
	"github.com/jinmugo/sls/internal/container"
	"github.com/spf13/cobra"
)

var (
	discoverTimeout     time.Duration
	discoverVerbose     bool
	discoverAllHosts    bool
	discoverConcurrency int
)

var discoverCmd = &cobra.Command{
	Use:   "discover [host-alias]",
	Short: "Discover Docker containers on a remote host",
	Long: `Discover running Docker containers on a remote host via SSH.
Results are cached locally for use in interactive mode and config generation.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cachePath, err := container.DefaultCachePath()
		if err != nil {
			return err
		}
		cache, err := container.LoadCache(cachePath)
		if err != nil {
			return err
		}

		if discoverAllHosts {
			return discoverAll(cache)
		}

		if len(args) == 0 {
			return fmt.Errorf("specify a host alias or use --hosts to discover all")
		}

		count, err := actions.Scan(args[0], cache, discoverTimeout)
		if err != nil {
			return err
		}
		if count == 0 {
			fmt.Fprintf(os.Stderr, "No running containers found on %s\n", args[0])
		}
		return nil
	},
}

func discoverAll(cache *container.Cache) error {
	hosts, err := config.Parse("")
	if err != nil {
		return fmt.Errorf("parse SSH config: %w", err)
	}

	var hostAliases []string
	for _, h := range hosts {
		if len(h.Patterns) > 0 {
			pat := h.Patterns[0].String()
			if pat != "*" {
				hostAliases = append(hostAliases, pat)
			}
		}
	}

	if len(hostAliases) == 0 {
		return fmt.Errorf("no hosts found in SSH config")
	}

	fmt.Fprintf(os.Stderr, "Discovering containers on %d host(s)...\n", len(hostAliases))
	results := container.DiscoverAll(hostAliases, discoverTimeout, discoverConcurrency, discoverVerbose)

	// Remove stale hosts from cache
	cache.RemoveStaleHosts(hostAliases)

	totalContainers := 0
	var errors []string
	for _, r := range results {
		if r.Err != nil {
			errors = append(errors, fmt.Sprintf("  ✗ %s: %v", r.Host, r.Err))
			continue
		}
		cache.MergeUpdate(r.Host, r.Containers)
		totalContainers += len(r.Containers)
		if len(r.Containers) > 0 {
			fmt.Fprintf(os.Stderr, "  ✓ %s: %d container(s)\n", r.Host, len(r.Containers))
		}
	}

	if err := cache.Save(); err != nil {
		return fmt.Errorf("save cache: %w", err)
	}

	if len(errors) > 0 {
		fmt.Fprintln(os.Stderr, "\nFailed hosts:")
		for _, e := range errors {
			fmt.Fprintln(os.Stderr, e)
		}
	}

	fmt.Fprintf(os.Stderr, "\nTotal: %d container(s) discovered\n", totalContainers)
	return nil
}

var DiscoverCmd = discoverCmd

func init() {
	discoverCmd.Flags().DurationVarP(&discoverTimeout, "timeout", "T", 10*time.Second, "SSH connection timeout")
	discoverCmd.Flags().BoolVarP(&discoverVerbose, "verbose", "v", false, "Show debug output")
	discoverCmd.Flags().BoolVar(&discoverAllHosts, "hosts", false, "Discover containers on all SSH config hosts")
	discoverCmd.Flags().IntVar(&discoverConcurrency, "concurrency", 10, "Max concurrent SSH connections for --hosts")
}
