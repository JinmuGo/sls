package cli

import (
	"strings"

	"github.com/jinmugo/sls/internal/config"
	"github.com/jinmugo/sls/internal/favorites"
	"github.com/spf13/cobra"
)

var favCmd = &cobra.Command{
	Use:   "fav",
	Short: "Manage favourite hosts",
}

func init() {
	favCmd.AddCommand(favAddCmd, favListCmd, favRemoveCmd)
}

var favAddCmd = &cobra.Command{
	Use:   "add [host]",
	Short: "Add a host to favourites",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := favorites.DefaultStore()
		return store.Add(args[0])
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		hosts, err := config.Parse("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		store := favorites.DefaultStore()
		favSet := map[string]struct{}{}
		for _, h := range store.List() {
			favSet[h] = struct{}{}
		}
		var candidates []string
		for _, h := range hosts {
			if len(h.Patterns) > 0 {
				pat := h.Patterns[0].String()
				if pat == "*" {
					continue
				}
				if _, ok := favSet[pat]; ok {
					continue
				}
				if toComplete == "" || strings.HasPrefix(pat, toComplete) {
					candidates = append(candidates, pat)
				}
			}
		}
		return candidates, cobra.ShellCompDirectiveNoFileComp
	},
}

var favListCmd = &cobra.Command{
	Use:   "list",
	Short: "List favourites",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := favorites.DefaultStore()
		for _, h := range store.List() {
			cmd.Println(h)
		}
		return nil
	},
}

var favRemoveCmd = &cobra.Command{
	Use:   "remove [host]",
	Short: "Remove a host from favourites",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := favorites.DefaultStore()
		return store.Remove(args[0])
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		store := favorites.DefaultStore()
		var candidates []string
		for _, h := range store.List() {
			if toComplete == "" || strings.HasPrefix(h, toComplete) {
				candidates = append(candidates, h)
			}
		}
		return candidates, cobra.ShellCompDirectiveNoFileComp
	},
}

var FavCmd = favCmd
