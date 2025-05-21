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

var rootCmd = &cobra.Command{
	Use:   "sls",
	Short: "ssh ls with fuzzyfinder",
	RunE: func(cmd *cobra.Command, args []string) error {
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

	favStore := favorites.DefaultStore()
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

	fzf := exec.Command("fzf",
		"--prompt", "sls> ",
		"--height", "~50%",
		"--layout", "reverse",
		"--preview", "sls preview {}",
	)
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

	favStore.Increment(choice)

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
	rootCmd.AddCommand(cli.CompletionCmd)
	rootCmd.AddCommand(cli.PreviewCmd)
}
