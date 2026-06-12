package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jinmugo/sls/internal/updater"
	"github.com/spf13/cobra"
)

// Version is the running sls version, injected by the cmd package before the
// command tree executes. Used by `sls update` to compare against the latest
// release.
var Version = "dev"

var (
	updateCheckOnly bool
	updateYes       bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update sls to the latest version",
	Long: `Update sls to the latest released version.

sls detects how it was installed (Homebrew, the Linux package repo, or
go install) and runs the matching upgrade command, asking for confirmation
first. Manually-downloaded binaries are pointed at the releases page instead.

Use --check to only report whether an update is available.`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

// UpdateCmd is the exported handle registered on the root command.
var UpdateCmd = updateCmd

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "Only check for an update; don't install")
	updateCmd.Flags().BoolVarP(&updateYes, "yes", "y", false, "Skip the confirmation prompt")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	current := Version

	if updateCheckOnly {
		latest, ok := updater.LatestVersion(current)
		fmt.Println(formatCheck(current, latest, ok))
		return nil
	}

	// Avoid an unnecessary package-manager call when already current.
	if latest, ok := updater.LatestVersion(current); ok && !updater.Outdated(current, latest) {
		fmt.Printf("sls %s is already the latest version.\n", current)
		return nil
	}

	method := updater.DetectMethod()
	argv, needsSudo, ok := updater.UpgradeCommand(method)
	if !ok {
		fmt.Println(manualInstructions(method))
		return nil
	}

	fmt.Printf("Detected install method: %s\n", method)
	fmt.Printf("Will run: %s\n", strings.Join(argv, " "))
	if needsSudo {
		fmt.Println("This command needs sudo and may prompt for your password.")
	}

	if !updateYes && !confirm("Continue?") {
		fmt.Println("Aborted.")
		return nil
	}

	if err := runCommand(argv); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	fmt.Println("✓ Update complete. Run 'sls version' to confirm.")
	return nil
}

// formatCheck renders the result of `sls update --check`.
func formatCheck(current, latest string, ok bool) string {
	if !ok {
		return "Could not determine the latest version (network error?)."
	}
	if updater.Outdated(current, latest) {
		return fmt.Sprintf("Update available: sls %s → %s. Run 'sls update' to upgrade.", current, latest)
	}
	return fmt.Sprintf("sls %s is up to date.", current)
}

// manualInstructions returns guidance for install methods with no safe automatic
// upgrade command.
func manualInstructions(m updater.Method) string {
	var b strings.Builder
	if m == updater.MethodUnknown {
		b.WriteString("Could not detect how sls was installed.\n")
	} else {
		b.WriteString("sls was installed as a standalone binary.\n")
	}
	b.WriteString("Download the latest release for your platform from:\n  ")
	b.WriteString(updater.ReleasesURL)
	return b.String()
}

// confirm prompts the user for a yes/no answer, defaulting to yes on empty input.
func confirm(prompt string) bool {
	fmt.Printf("%s [Y/n] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "" || answer == "y" || answer == "yes"
}

// runCommand executes argv, wiring through the current terminal so package
// managers can prompt (e.g. for a sudo password) and stream their output.
func runCommand(argv []string) error {
	c := exec.Command(argv[0], argv[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
