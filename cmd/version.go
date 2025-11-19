package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// These variables are set via ldflags during build
	version   = "dev"
	commit    = "none"
	date      = "unknown"
	builtBy   = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print detailed version information including build time and commit hash",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("sls version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built at: %s\n", date)
		fmt.Printf("  built by: %s\n", builtBy)
		fmt.Printf("  go version: %s\n", runtime.Version())
		fmt.Printf("  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Add --version flag to root command
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
}
