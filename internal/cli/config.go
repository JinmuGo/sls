package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jinmugo/sls/internal/config"
	"github.com/jinmugo/sls/internal/consts"
	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

var (
	flagHostName string
	flagUser     string
	flagPort     int
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ssh_config entries",
}

var ConfigCmd = configCmd

func init() {
	cfgListCmd := &cobra.Command{
		Use:   "list",
		Short: "List Host entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			hosts, err := config.Parse("")
			if err != nil {
				return err
			}
			for _, h := range hosts {
				if len(h.Patterns) > 0 {
					pat := h.Patterns[0].String()
					if pat == "*" {
						continue
					}
					fmt.Println(pat)
				}
			}
			return nil
		},
	}

	cfgAddCmd := &cobra.Command{
		Use:   "add <alias>",
		Short: "Add a Host entry (interactive)",
		Args:  cobra.ExactArgs(1),
		RunE:  runCfgUpsert,
	}

	cfgEditCmd := &cobra.Command{
		Use:   "edit <alias>",
		Short: "Edit fields of an existing Host entry",
		Args:  cobra.ExactArgs(1),
		RunE:  runCfgUpsert,
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

	for _, c := range []*cobra.Command{cfgAddCmd, cfgEditCmd} {
		c.Flags().StringVar(&flagHostName, "hostname", "", "HostName value")
		c.Flags().StringVar(&flagUser, "user", "", "User value")
		c.Flags().IntVar(&flagPort, "port", 0, "Port value")
	}

	cfgRemoveCmd := &cobra.Command{
		Use:   "remove <alias>",
		Short: "Remove a Host entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := config.LoadAST("")
			if err != nil {
				return err
			}
			if !config.DeleteHost(cfg, args[0]) {
				return fmt.Errorf("host %q not found", args[0])
			}
			return config.SaveAST(cfg, path)
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

	configCmd.AddCommand(cfgListCmd, cfgAddCmd, cfgEditCmd, cfgRemoveCmd)
}

func runCfgUpsert(cmd *cobra.Command, args []string) error {
	alias := args[0]
	reader := bufio.NewReader(os.Stdin)

	if cmd.Name() == "edit" {
		hosts, err := config.Parse("")
		if err != nil {
			return err
		}
		found := false
		for _, h := range hosts {
			if len(h.Patterns) > 0 && h.Patterns[0].String() == alias {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("host %q not found in ~/.ssh/config", alias)
		}
	}

	cfg, path, err := config.LoadAST("")
	if err != nil {
		return err
	}
	h, _ := config.FindHost(cfg, alias)
	curHostName, curUser, curPort := "", "", 0
	if h != nil {
		curHostName = config.GetKV(h, consts.SSHConfigHostName)
		curUser = config.GetKV(h, consts.SSHConfigUser)
		curPortStr := config.GetKV(h, consts.SSHConfigPort)
		if curPortStr != "" {
			if p, err := strconv.Atoi(curPortStr); err == nil {
				curPort = p
			}
		}
	}

	if flagHostName == "" {
		prompt := consts.SSHConfigHostName
		if curHostName != "" {
			prompt += fmt.Sprintf(" (%s)", curHostName)
		}
		fmt.Printf("%s: ", prompt)
		in, _ := reader.ReadString('\n')
		in = strings.TrimSpace(in)
		if in != "" {
			flagHostName = in
		} else if curHostName != "" {
			flagHostName = curHostName
		}
	}

	if flagUser == "" {
		prompt := consts.SSHConfigUser
		if curUser != "" {
			prompt += fmt.Sprintf(" (%s)", curUser)
		}
		fmt.Printf("%s: ", prompt)
		in, _ := reader.ReadString('\n')
		in = strings.TrimSpace(in)
		if in != "" {
			flagUser = in
		} else if curUser != "" {
			flagUser = curUser
		}
	}

	if flagPort == 0 {
		prompt := consts.SSHConfigPort
		if curPort != 0 {
			prompt += fmt.Sprintf(" (%d)", curPort)
		}
		fmt.Printf("%s: ", prompt)
		in, _ := reader.ReadString('\n')
		in = strings.TrimSpace(in)
		if in != "" {
			if p, err := strconv.Atoi(in); err == nil {
				flagPort = p
			}
		} else if curPort != 0 {
			flagPort = curPort
		}
	}

	extra := map[string]string{}
	allOpts := []string{}
	for k := range consts.AllSSHConfigOptions {
		allOpts = append(allOpts, k)
	}
	for {
		fmt.Print("Add a new option? (y/n): ")
		yn, _ := reader.ReadString('\n')
		yn = strings.ToLower(strings.TrimSpace(yn))
		if yn != "y" {
			break
		}
		idx, err := fuzzyfinder.Find(
			allOpts,
			func(i int) string { return allOpts[i] },
		)
		if err != nil {
			fmt.Println("cancelled")
			continue
		}
		key := allOpts[idx]
		fmt.Printf("%s: ", key)
		val, _ := reader.ReadString('\n')
		extra[key] = strings.TrimSpace(val)
	}

	h = config.UpsertHost(cfg, alias, flagHostName, flagUser, flagPort)
	for k, v := range extra {
		config.SetKV(h, k, v)
	}
	if err := config.SaveAST(cfg, path); err != nil {
		return err
	}
	fmt.Printf("%s updated\n", alias)
	return nil
}
