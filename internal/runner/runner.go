package runner

import (
    "os"
    "os/exec"
    "syscall"
)

// SSH runs an interactive SSH session. Exit code 255 is suppressed because
// it commonly indicates the user closed the session normally.
func SSH(host string, args []string) error {
    cmd := exec.Command("ssh", append([]string{host}, args...)...)
    return runSSH(cmd, true)
}

// SSHWithCmd runs ssh with a remote command (e.g., docker exec -it container /bin/sh).
// Uses -t to force TTY allocation, required for interactive docker exec.
// Exit code 255 is NOT suppressed here because it indicates a real failure
// (auth failure, host unreachable, command not found).
func SSHWithCmd(host string, remoteCmd []string) error {
    args := append([]string{"-t", host}, remoteCmd...)
    cmd := exec.Command("ssh", args...)
    return runSSH(cmd, false)
}

func runSSH(cmd *exec.Cmd, suppressExit255 bool) error {
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err := cmd.Run()
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
                if suppressExit255 && status.ExitStatus() == 255 {
                    return nil
                }
            }
        }
        return err
    }
    return nil
}
