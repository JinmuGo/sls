package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jinmugo/sls/internal/actions"
	"github.com/jinmugo/sls/internal/cli"
	"github.com/jinmugo/sls/internal/config"
	"github.com/jinmugo/sls/internal/container"
	"github.com/jinmugo/sls/internal/favorites"
	"github.com/jinmugo/sls/internal/finder"
	"github.com/jinmugo/sls/internal/hostinfo"
	"github.com/jinmugo/sls/internal/onboarding"
	"github.com/jinmugo/sls/internal/pulse"
	"github.com/jinmugo/sls/internal/updater"
	sshconfig "github.com/kevinburke/ssh_config"
	"github.com/spf13/cobra"
)

var filterTag string

var rootCmd = &cobra.Command{
	Use:   "sls [flags] [-- extra-ssh-args...]",
	Short: "Smart fuzzy CLI selector for SSH config hosts",
	Long:  "sls is an interactive CLI tool for selecting and connecting to SSH hosts defined in ~/.ssh/config.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// `sls --version` / `sls -v` should print version info, not launch the TUI.
		if ok, _ := cmd.Flags().GetBool("version"); ok {
			versionCmd.Run(cmd, args)
			return nil
		}
		return runInteractive(args)
	},
}

func runInteractive(extraSSHArgs []string) error {
	// Onboarding check (runs once)
	hosts, err := config.Parse("")
	if err != nil {
		if errors.Is(err, config.ErrSSHDirNotExist) || errors.Is(err, config.ErrSSHConfigNotExist) {
			retry, alias, onboardErr := onboarding.HandleMissingConfig(err)
			if onboardErr != nil {
				return onboardErr
			}
			if alias != "" {
				if addErr := cli.RunConfigAdd(alias); addErr != nil {
					return addErr
				}
			}
			if !retry {
				return nil
			}
		} else {
			return fmt.Errorf("parse ssh_config: %w", err)
		}
	}

	if len(hosts) == 0 {
		retry, alias, onboardErr := onboarding.HandleEmptyConfig()
		if onboardErr != nil {
			return onboardErr
		}
		if alias != "" {
			if addErr := cli.RunConfigAdd(alias); addErr != nil {
				return addErr
			}
		}
		if !retry {
			return nil
		}
	}

	// Load persistent stores once
	favStore, err := favorites.DefaultStore()
	if err != nil {
		return fmt.Errorf("load favorites: %w", err)
	}

	cachePath, _ := container.DefaultCachePath()
	cache, cacheErr := container.LoadCache(cachePath)
	var cacheWarning string
	if cacheErr != nil {
		cacheWarning = "⚠ Container cache unreadable"
		cache = &container.Cache{Hosts: make(map[string]container.HostCache)}
	}

	// Load host info cache for preview panel
	infoCachePath, _ := hostinfo.DefaultCachePath()
	infoCache, _ := hostinfo.LoadCache(infoCachePath)
	infoFetcher := hostinfo.NewFetcher(infoCache)

	// Track what needs reloading
	needReloadHosts := true
	needReloadFavs := false
	needReloadCache := false

	var statusMsg string
	var restoreAlias string
	hasScanned := len(cache.Hosts) > 0

	// Update check: build a banner from the last cached result, then refresh the
	// cache in the background for next time. No-op when disabled, on dev builds,
	// or when not running interactively.
	var updateBanner string
	if notice := updater.Cached(version); notice != nil {
		updateBanner = fmt.Sprintf("⬆ sls %s available · run 'sls update'", notice.Latest)
	}
	updater.RefreshAsync(version)

	// Main loop
	for {
		if needReloadHosts {
			hosts, err = config.Parse("")
			if err != nil {
				return fmt.Errorf("parse ssh_config: %w", err)
			}
			needReloadHosts = false
		}
		if needReloadFavs {
			// Keep the previous store on error: overwriting favStore with a nil
			// pointer would panic the TUI loop on the next buildItems call.
			if fs, ferr := favorites.DefaultStore(); ferr == nil {
				favStore = fs
			}
			needReloadFavs = false
		}
		if needReloadCache {
			// Likewise, only replace the cache when the reload succeeds.
			if c, cerr := container.LoadCache(cachePath); cerr == nil {
				cache = c
			}
			needReloadCache = false
		}

		items, hostCount, containerCount := buildItems(hosts, favStore, cache)

		opts := finder.SelectOpts{
			StatusMsg:      statusMsg,
			RestoreAlias:   restoreAlias,
			HostCount:      hostCount,
			ContainerCount: containerCount,
			HasScanned:     hasScanned,
			InfoFetcher:    infoFetcher,
			InfoCache:      infoCache,
			UpdateBanner:   updateBanner,
		}
		if cacheWarning != "" {
			opts.StatusMsg = cacheWarning
			cacheWarning = "" // show once
		}

		result, err := finder.Select(items, opts)
		if err != nil {
			return err
		}
		if result.Alias == "" {
			return nil // esc/ctrl+c
		}

		statusMsg = ""
		restoreAlias = result.Alias

		switch result.Action {
		case "connect":
			pulse.Track("command_run", pulse.Props{"command": "connect"})
			return actions.Connect(result.Alias, extraSSHArgs, favStore, cache)

		case "rename":
			if strings.Contains(result.Alias, container.KeySep) {
				parts := strings.SplitN(result.Alias, container.KeySep, 2)
				newName, renameErr := actions.RenameContainer(cache, parts[0], parts[1])
				if renameErr != nil {
					statusMsg = finder.StyleError.Render("  ✗ " + renameErr.Error())
				} else if newName != "" {
					statusMsg = finder.StyleSuccess.Render("  ✓ Renamed → " + newName)
					needReloadCache = true
				}
			} else {
				newName, renameErr := actions.RenameHost(result.Alias, favStore, cache, infoCache)
				if renameErr != nil {
					statusMsg = finder.StyleError.Render("  ✗ " + renameErr.Error())
				} else if newName != "" {
					statusMsg = finder.StyleSuccess.Render("  ✓ Renamed → " + newName)
					restoreAlias = newName
					needReloadHosts = true
					needReloadFavs = true
					needReloadCache = true
				}
			}

		case "scan":
			pulse.Track("command_run", pulse.Props{"command": "scan"})
			count, scanErr := actions.Scan(result.Alias, cache, 10*time.Second)
			if scanErr != nil {
				statusMsg = finder.StyleError.Render("  ✗ Scan failed: " + scanErr.Error())
			} else if count == 0 {
				statusMsg = finder.StyleDim.Render("  ○ No containers on " + result.Alias)
			} else {
				statusMsg = finder.StyleSuccess.Render(fmt.Sprintf("  ✓ Added %d container(s)", count))
				hasScanned = true
			}
			needReloadCache = true

		case "delete":
			if strings.Contains(result.Alias, container.KeySep) {
				parts := strings.SplitN(result.Alias, container.KeySep, 2)
				if delErr := actions.DeleteContainer(cache, parts[0], parts[1], favStore); delErr != nil {
					statusMsg = finder.StyleError.Render("  ✗ " + delErr.Error())
				} else {
					statusMsg = finder.StyleSuccess.Render("  ✓ Deleted " + parts[1])
					restoreAlias = parts[0]
					needReloadCache = true
				}
			} else {
				if delErr := actions.DeleteHost(result.Alias, favStore, cache, infoCache); delErr != nil {
					statusMsg = finder.StyleError.Render("  ✗ " + delErr.Error())
				} else {
					statusMsg = finder.StyleSuccess.Render("  ✓ Deleted " + result.Alias)
					restoreAlias = ""
					needReloadHosts = true
					needReloadFavs = true
				}
			}

		case "star":
			if starErr := actions.Star(favStore, result.Alias); starErr != nil {
				statusMsg = finder.StyleError.Render("  ✗ " + starErr.Error())
			} else {
				if favStore.IsFavorite(result.Alias) {
					statusMsg = finder.StyleSuccess.Render("  ✓ Starred " + result.Alias)
				} else {
					statusMsg = finder.StyleSuccess.Render("  ✓ Unstarred " + result.Alias)
				}
				needReloadFavs = true
			}
		}
	}
}

// buildItems constructs the sorted item list for the finder.
func buildItems(hosts []*sshconfig.Host, favStore *favorites.Store, cache *container.Cache) ([]finder.Item, int, int) {
	var favAliases []string
	var normalAliases []struct {
		Alias string
		Count int
	}

	for _, h := range hosts {
		for _, p := range h.Patterns {
			pat := p.String()
			if pat == "*" {
				continue
			}
			if filterTag != "" && !favStore.HasTag(pat, filterTag) {
				continue
			}
			if favStore.IsFavorite(pat) {
				favAliases = append(favAliases, pat)
			} else {
				normalAliases = append(normalAliases, struct {
					Alias string
					Count int
				}{pat, favStore.Count(pat)})
			}
		}
	}
	sort.SliceStable(normalAliases, func(i, j int) bool {
		return normalAliases[i].Count > normalAliases[j].Count
	})

	hostCount := len(favAliases) + len(normalAliases)
	containerCount := 0

	// Favorited containers at top level
	favContainerSet := make(map[string]bool)
	var favContainerItems []finder.Item
	if cache != nil {
		for hostAlias, hc := range cache.Hosts {
			for _, c := range hc.Containers {
				key := hostAlias + container.KeySep + c.Name
				if favStore.IsFavorite(key) {
					favContainerSet[key] = true
					favContainerItems = append(favContainerItems, finder.Item{
						Label: "⭐︎" + containerLabel(c, ""),
						Alias: key,
					})
					containerCount++
				}
			}
		}
	}

	// Nested containers per host
	containerItems := func(hostAlias string) []finder.Item {
		if cache == nil {
			return nil
		}
		containers := cache.GetContainers(hostAlias)
		if len(containers) == 0 {
			return nil
		}
		staleTag := ""
		if cache.IsStale(hostAlias, 1*time.Hour) {
			staleTag = " (stale)"
		}
		var items []finder.Item
		for _, c := range containers {
			key := hostAlias + container.KeySep + c.Name
			if favContainerSet[key] {
				continue
			}
			items = append(items, finder.Item{
				Label:    containerLabel(c, staleTag),
				Alias:    key,
				IsNested: true,
			})
			containerCount++
		}
		if len(items) > 0 {
			items[len(items)-1].IsLast = true
		}
		return items
	}

	var items []finder.Item
	items = append(items, favContainerItems...)
	for _, alias := range favAliases {
		items = append(items, finder.Item{Label: "⭐︎" + alias, Alias: alias})
		items = append(items, containerItems(alias)...)
	}
	for _, n := range normalAliases {
		items = append(items, finder.Item{Label: n.Alias, Alias: n.Alias})
		items = append(items, containerItems(n.Alias)...)
	}

	return items, hostCount, containerCount
}

// containerLabel builds the display label for a container in the finder list.
// e.g., "nginx 🐳", "nginx 🐳 (bash)", "nginx 🐳 (no shell)", "nginx 🐳 (stale)"
func containerLabel(c container.Container, suffix string) string {
	label := c.DisplayName() + " 🐳"
	if sl := c.ShellLabel(); sl != "" {
		label += " (" + sl + ")"
	}
	label += suffix
	return label
}

func Execute() {
	pulse.Init(version)
	defer pulse.Shutdown()

	// Inject the running version so `sls update` can compare against the latest.
	cli.Version = version

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(cli.ConfigCmd)
	rootCmd.AddCommand(cli.FavCmd)
	rootCmd.AddCommand(cli.TagCmd)
	rootCmd.AddCommand(cli.TestCmd)
	rootCmd.AddCommand(cli.CompletionCmd)
	rootCmd.AddCommand(cli.DiscoverCmd)
	rootCmd.AddCommand(cli.ConnectCmd)
	rootCmd.AddCommand(cli.GenCmd)
	rootCmd.AddCommand(cli.UpdateCmd)

	rootCmd.Flags().StringVarP(&filterTag, "tag", "t", "", "Filter hosts by tag")
}
