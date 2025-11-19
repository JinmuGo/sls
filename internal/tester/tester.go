package tester

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// TestConnection tests SSH connectivity to a host with a timeout
func TestConnection(host string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Use SSH with a simple command to test connectivity
	// -o BatchMode=yes prevents interactive password prompts
	// -o ConnectTimeout sets connection timeout
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=no",
		host,
		"echo",
		"ok",
	)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("connection timeout after %v", timeout)
		}
		return fmt.Errorf("connection failed: %w", err)
	}

	return nil
}
