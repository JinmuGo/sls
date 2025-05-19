package runner

import (
    "os"
    "os/exec"
)

func SSH(host string, args []string) error {
    cmd := exec.Command("ssh", append([]string{host}, args...)...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
