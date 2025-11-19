package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jinmugo/sls/internal/config"
	"github.com/jinmugo/sls/internal/favorites"
	"github.com/jinmugo/sls/internal/validator"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage host tags",
}

func init() {
	tagCmd.AddCommand(tagAddCmd, tagRemoveCmd, tagListCmd, tagShowAllCmd)
}

var tagAddCmd = &cobra.Command{
	Use:   "add <host> <tag>",
	Short: "Add a tag to a host",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate that host exists in SSH config
		if err := validator.ValidateHostExists(args[0]); err != nil {
			return err
		}

		store, err := favorites.DefaultStore()
		if err != nil {
			return err
		}
		if err := store.AddTag(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Added tag %q to host %q\n", args[1], args[0])
		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// First argument: host name
			hosts, err := config.Parse("")
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var candidates []string
			for _, h := range hosts {
				if len(h.Patterns) > 0 {
					pat := h.Patterns[0].String()
					if pat == "*" {
						continue
					}
					if toComplete == "" || strings.HasPrefix(pat, toComplete) {
						candidates = append(candidates, pat)
					}
				}
			}
			return candidates, cobra.ShellCompDirectiveNoFileComp
		}
		// Second argument: tag name (no completion)
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:   "remove <host> <tag>",
	Short: "Remove a tag from a host",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate that host exists in SSH config
		if err := validator.ValidateHostExists(args[0]); err != nil {
			return err
		}

		store, err := favorites.DefaultStore()
		if err != nil {
			return err
		}
		if err := store.RemoveTag(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Removed tag %q from host %q\n", args[1], args[0])
		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// First argument: host name
			hosts, err := config.Parse("")
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var candidates []string
			for _, h := range hosts {
				if len(h.Patterns) > 0 {
					pat := h.Patterns[0].String()
					if pat == "*" {
						continue
					}
					if toComplete == "" || strings.HasPrefix(pat, toComplete) {
						candidates = append(candidates, pat)
					}
				}
			}
			return candidates, cobra.ShellCompDirectiveNoFileComp
		} else if len(args) == 1 {
			// Second argument: existing tags for the host
			store, err := favorites.DefaultStore()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			tags := store.GetTags(args[0])
			var candidates []string
			for _, tag := range tags {
				if toComplete == "" || strings.HasPrefix(tag, toComplete) {
					candidates = append(candidates, tag)
				}
			}
			return candidates, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

var tagListCmd = &cobra.Command{
	Use:   "list <host>",
	Short: "List tags for a host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate that host exists in SSH config
		if err := validator.ValidateHostExists(args[0]); err != nil {
			return err
		}

		store, err := favorites.DefaultStore()
		if err != nil {
			return err
		}
		tags := store.GetTags(args[0])
		if len(tags) == 0 {
			fmt.Printf("No tags for host %q\n", args[0])
			return nil
		}
		fmt.Printf("Tags for %q:\n", args[0])
		for _, tag := range tags {
			fmt.Printf("  - %s\n", tag)
		}
		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		hosts, err := config.Parse("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var candidates []string
		for _, h := range hosts {
			if len(h.Patterns) > 0 {
				pat := h.Patterns[0].String()
				if pat == "*" {
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

var tagShowAllCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all tags and their hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := favorites.DefaultStore()
		if err != nil {
			return err
		}

		// Build a map of tag -> hosts
		tagHosts := make(map[string][]string)
		for host, entry := range store.Data() {
			for _, tag := range entry.Tags {
				tagHosts[tag] = append(tagHosts[tag], host)
			}
		}

		if len(tagHosts) == 0 {
			fmt.Println("No tags defined")
			return nil
		}

		// Sort tags alphabetically
		tags := make([]string, 0, len(tagHosts))
		for tag := range tagHosts {
			tags = append(tags, tag)
		}
		sort.Strings(tags)

		fmt.Println("All tags:")
		for _, tag := range tags {
			hosts := tagHosts[tag]
			sort.Strings(hosts)
			fmt.Printf("  %s: %s\n", tag, strings.Join(hosts, ", "))
		}
		return nil
	},
}

var TagCmd = tagCmd
