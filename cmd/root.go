package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/jinmugo/sls/internal/cli"
	"github.com/jinmugo/sls/internal/config"
	"github.com/jinmugo/sls/internal/favorites"
	"github.com/jinmugo/sls/internal/runner"
	"github.com/spf13/cobra"
)

var (
	filterTag string
)

var rootCmd = &cobra.Command{
	Use:   "sls",
	Short: "ssh ls with fuzzyfinder",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle --version flag
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			cmd.Root().SetArgs([]string{"version"})
			return cmd.Root().Execute()
		}
		return runInteractive(args)
	},
}

func runInteractive(extraSSHArgs []string) error {
	hosts, err := config.Parse("")
	if err != nil {
		return fmt.Errorf("parse ssh_config: %w", err)
	}
	if len(hosts) == 0 {
		return fmt.Errorf("no Host entries found in ssh_config")
	}

	favStore, err := favorites.DefaultStore()
	if err != nil {
		return fmt.Errorf("load favorites: %w", err)
	}
	favMap := map[string]struct{}{}
	favCounts := map[string]int{}
	for _, h := range favStore.List() {
		favMap[h] = struct{}{}
		favCounts[h] = favStore.Count(h)
	}

	var favAliases []string
	var normalAliases []struct {
		Alias string
		Count int
	}
	for _, h := range hosts {
		if len(h.Patterns) > 0 {
			pat := h.Patterns[0].String()
			if pat == "*" {
				continue
			}
			// Filter by tag if specified
			if filterTag != "" && !favStore.HasTag(pat, filterTag) {
				continue
			}
			if favStore.IsFavorite(pat) {
				favAliases = append(favAliases, "⭐︎"+pat)
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

	var aliases []string
	aliases = append(aliases, favAliases...)
	for _, n := range normalAliases {
		aliases = append(aliases, n.Alias)
	}

	// Check if fzf is available
	if _, err := exec.LookPath("fzf"); err != nil {
		return fmt.Errorf("fzf not found in PATH. Please install fzf: https://github.com/junegunn/fzf")
	}

	fzfArgs := []string{"--prompt", "sls > ", "--preview", "sls preview {}"}
	if os.Getenv("FZF_DEFAULT_OPTS") == "" {
		fzfArgs = append(fzfArgs,
			"--height", "~50%",
			"--layout", "reverse",
		)
	}

	fzf := exec.Command("fzf", fzfArgs...)
	fzf.Stdin = strings.NewReader(strings.Join(aliases, "\n"))

	var out bytes.Buffer
	fzf.Stdout = &out
	fzf.Stderr = os.Stderr

	if err := fzf.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() > 0 {
			return nil
		}
		return fmt.Errorf("fzf: %w", err)
	}

	choice := strings.TrimSpace(out.String())
	if choice == "" {
		return nil
	}
	choice = strings.TrimPrefix(choice, "⭐︎")

	if err := favStore.Increment(choice); err != nil {
		// Don't fail on increment error, just log to stderr
		fmt.Fprintf(os.Stderr, "Warning: failed to update usage count: %v\n", err)
	}

	return runner.SSH(choice, extraSSHArgs)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(cli.ConfigCmd)
	rootCmd.AddCommand(cli.FavCmd)
	rootCmd.AddCommand(cli.TagCmd)
	rootCmd.AddCommand(cli.TestCmd)
	rootCmd.AddCommand(cli.CompletionCmd)
	rootCmd.AddCommand(cli.PreviewCmd)

	// Add --tag flag for filtering hosts by tag
	rootCmd.Flags().StringVarP(&filterTag, "tag", "t", "", "Filter hosts by tag")
}
