package runner

import (
    "errors"
    "os"
    "os/exec"
    "syscall"
)

// ExitCode returns the process exit code carried by err, or -1 if err is not an
// *exec.ExitError (e.g. the command could not be started or was killed by a
// signal). Returns 0 when err is nil.
func ExitCode(err error) int {
    if err == nil {
        return 0
    }
    var exitErr *exec.ExitError
    if errors.As(err, &exitErr) {
        return exitErr.ExitCode()
    }
    return -1
}

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
