package cli

import (
	"strings"

	"github.com/jinmugo/sls/internal/actions"
	"github.com/jinmugo/sls/internal/container"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect <target>",
	Short: "Connect to a server or container",
	Long: `Connect to a server or container via SSH.

Targets containing "::" are treated as container references (host::container).
Other targets are treated as SSH host aliases.

If a container is no longer running, sls will auto-refresh the cache and retry once.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// Load cache for container refresh-on-miss
		var cache *container.Cache
		if strings.Contains(target, container.KeySep) {
			cachePath, err := container.DefaultCachePath()
			if err == nil {
				cache, _ = container.LoadCache(cachePath)
			}
		}

		return actions.Connect(target, nil, nil, cache)
	},
}

var ConnectCmd = connectCmd
