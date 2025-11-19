package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/jinmugo/sls/internal/config"
	"github.com/jinmugo/sls/internal/tester"
	"github.com/jinmugo/sls/internal/validator"
	"github.com/spf13/cobra"
)

var (
	testTimeout int
)

var testCmd = &cobra.Command{
	Use:     "test <host>",
	Aliases: []string{"ping"},
	Short:   "Test SSH connectivity to a host",
	Long: `Test SSH connectivity to a host with a timeout.
This command attempts to connect via SSH and run a simple echo command.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]

		// Validate that host exists in SSH config
		if err := validator.ValidateHostExists(host); err != nil {
			return err
		}

		timeout := time.Duration(testTimeout) * time.Second

		fmt.Printf("Testing connection to %s (timeout: %v)...\n", host, timeout)

		err := tester.TestConnection(host, timeout)
		if err != nil {
			fmt.Printf("\033[31m✗ Connection failed:\033[0m %v\n", err)
			return err
		}

		fmt.Printf("\033[32m✓ Connection successful\033[0m\n")
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

func init() {
	testCmd.Flags().IntVarP(&testTimeout, "timeout", "T", 10, "Connection timeout in seconds")
}

var TestCmd = testCmd
