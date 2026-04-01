package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jinmugo/sls/internal/container"
	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate SSH config and related files",
}

var genSSHConfigCmd = &cobra.Command{
	Use:   "ssh-config",
	Short: "Generate SSH config entries for discovered containers",
	Long: `Generate an SSH config include file with Host entries for all discovered containers.

The generated file is written to ~/.config/sls/ssh_config.
Add "Include ~/.config/sls/ssh_config" to your ~/.ssh/config to use the entries
with vanilla ssh (e.g., ssh my-server--nginx).

Note: Generated entries are for shell access only.
Do not use them with scp or rsync (RemoteCommand + RequestTTY will break file transfer).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cachePath, err := container.DefaultCachePath()
		if err != nil {
			return err
		}
		cache, err := container.LoadCache(cachePath)
		if err != nil {
			return err
		}

		if len(cache.Hosts) == 0 {
			return fmt.Errorf("no cached containers. Run `sls discover <host>` first")
		}

		includePath, err := container.DefaultIncludePath()
		if err != nil {
			return err
		}

		if err := container.GenerateIncludeFile(cache, includePath); err != nil {
			return err
		}

		// Count total entries
		total := 0
		for _, hc := range cache.Hosts {
			total += len(hc.Containers)
		}

		fmt.Fprintf(os.Stderr, "✓ Generated %d host entries at %s\n", total, includePath)

		// Offer to add Include line
		home, _ := os.UserHomeDir()
		sshConfigPath := filepath.Join(home, ".ssh", "config")
		if err := container.AddIncludeLine(sshConfigPath, includePath); err != nil {
			fmt.Fprintf(os.Stderr, "\nTo use with vanilla ssh, add this to your ~/.ssh/config:\n")
			fmt.Fprintf(os.Stderr, "  Include %s\n", includePath)
		} else {
			fmt.Fprintf(os.Stderr, "✓ Include directive added to %s\n", sshConfigPath)
		}

		return nil
	},
}

var GenCmd = genCmd

func init() {
	genCmd.AddCommand(genSSHConfigCmd)
}
