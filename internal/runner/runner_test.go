package runner

import (
	"errors"
	"os/exec"
	"testing"
)

func TestExitCode(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Errorf("ExitCode(nil) = %d, want 0", got)
	}

	// A plain (non-exec) error has no exit code.
	if got := ExitCode(errors.New("boom")); got != -1 {
		t.Errorf("ExitCode(plain err) = %d, want -1", got)
	}

	// Real process exit codes are surfaced exactly.
	for _, code := range []int{1, 2, 3, 42, 126, 127} {
		err := exec.Command("sh", "-c", "exit "+itoa(code)).Run()
		if got := ExitCode(err); got != code {
			t.Errorf("ExitCode(exit %d) = %d, want %d", code, got, code)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
