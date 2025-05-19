package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jinmugo/sls/internal/config"
	"github.com/jinmugo/sls/internal/consts"
	"github.com/spf13/cobra"
)

var previewCmd = &cobra.Command{
	Use:    "preview <host>",
	Short:  "Print details of a given Host (for use in fzf preview)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := config.LoadAST("")
		if err != nil {
			return err
		}
		alias := strings.TrimSpace(strings.TrimPrefix(args[0], "⋆"))
		h, _ := config.FindHost(cfg, alias)
		if h == nil {
			fmt.Println("(no such host)")
			return nil
		}

		var keys []string
		for key := range consts.RequiredSSHConfigOptions {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		var lines []string
		for _, key := range keys {
			val := config.GetKV(h, key)
			if val == "" && key != consts.SSHConfigHost {
				lines = append(lines, fmt.Sprintf("\033[33m%s:\033[0m \033[90m(not set)\033[0m", key))
			} else {
				if key == consts.SSHConfigHost {
					lines = append(lines, fmt.Sprintf("\033[36m%s:\033[0m %s", key, h.Patterns[0]))
				} else {
					lines = append(lines, fmt.Sprintf("\033[36m%s:\033[0m %s", key, val))
				}
			}
		}
		fmt.Println(strings.Join(lines, "\n"))
		return nil
	},
}

var PreviewCmd = previewCmd
