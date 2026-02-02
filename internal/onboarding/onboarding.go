package onboarding

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jinmugo/sls/internal/config"
)

// HandleMissingConfig handles SSH config missing scenarios with user-friendly prompts.
// Returns:
//   - shouldRetry: whether the caller should retry parsing the config
//   - addAlias: if non-empty, the alias to add as the first host
//   - err: any error that occurred during the process
func HandleMissingConfig(err error) (shouldRetry bool, addAlias string, retErr error) {
	reader := bufio.NewReader(os.Stdin)

	if errors.Is(err, config.ErrSSHDirNotExist) {
		fmt.Println("~/.ssh directory not found.")
		if !promptYesNo(reader, "Create ~/.ssh directory?") {
			fmt.Println("To add a host, run: sls config add <alias>")
			return false, "", nil
		}
	}

	if errors.Is(err, config.ErrSSHDirNotExist) || errors.Is(err, config.ErrSSHConfigNotExist) {
		if errors.Is(err, config.ErrSSHConfigNotExist) {
			fmt.Println("SSH config file not found.")
			if !promptYesNo(reader, "Create ~/.ssh/config?") {
				fmt.Println("To add a host, run: sls config add <alias>")
				return false, "", nil
			}
		}

		// Create the SSH config file
		if _, createErr := config.EnsureSSHConfig(); createErr != nil {
			return false, "", createErr
		}

		// Prompt to add first host
		return promptAddFirstHost(reader)
	}

	if errors.Is(err, config.ErrSSHConfigEmpty) {
		fmt.Println("No SSH hosts configured.")
		return promptAddFirstHost(reader)
	}

	return false, "", err
}

// HandleEmptyConfig handles the case when SSH config exists but has no hosts.
func HandleEmptyConfig() (shouldRetry bool, addAlias string, retErr error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("No SSH hosts configured.")
	return promptAddFirstHost(reader)
}

func promptAddFirstHost(reader *bufio.Reader) (shouldRetry bool, addAlias string, retErr error) {
	if !promptYesNo(reader, "Add your first host?") {
		fmt.Println("To add a host, run: sls config add <alias>")
		return false, "", nil
	}

	fmt.Print("Enter host alias: ")
	alias, err := reader.ReadString('\n')
	if err != nil {
		return false, "", fmt.Errorf("failed to read alias: %w", err)
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		fmt.Println("No alias provided. To add a host, run: sls config add <alias>")
		return false, "", nil
	}

	return true, alias, nil
}

func promptYesNo(reader *bufio.Reader, question string) bool {
	fmt.Printf("%s (y/n): ", question)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}
