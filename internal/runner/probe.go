package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Probe runs a non-interactive SSH command on a remote host and returns the output.
// It uses BatchMode to prevent interactive prompts and respects context cancellation.
func Probe(ctx context.Context, host string, remoteCmd string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		host,
		remoteCmd,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("SSH to %s timed out", host)
		}
		if ctx.Err() == context.Canceled {
			return nil, ctx.Err()
		}
		outStr := strings.TrimSpace(string(output))
		if strings.Contains(outStr, "Permission denied") || strings.Contains(outStr, "permission denied") {
			return nil, fmt.Errorf("SSH auth failed for %s", host)
		}
		return nil, fmt.Errorf("SSH to %s failed: %w", host, err)
	}

	return output, nil
}
