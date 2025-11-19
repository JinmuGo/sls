package runner

import (
    "os"
    "os/exec"
    "syscall"
)

func SSH(host string, args []string) error {
    cmd := exec.Command("ssh", append([]string{host}, args...)...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err := cmd.Run()
    if err != nil {
        // SSH exit code 255 is a normal connection termination by remote host
        // Don't treat it as an error
        if exitErr, ok := err.(*exec.ExitError); ok {
            if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
                if status.ExitStatus() == 255 {
                    return nil
                }
            }
        }
        return err
    }
    return nil
}
