package hostinfo

import (
	"testing"
)

func TestParseOutputFull(t *testing.T) {
	raw := `---OS---
Ubuntu 22.04.3 LTS
---KERNEL---
6.5.0-44-generic
---UPTIME---
up 23 days, 4 hours
---DISK---
/dev/sda1       50G   32G   18G  64% /
---MEMORY---
Mem:           3.8Gi  2.1Gi  1.2Gi  128Mi  1.2Gi  3.2Gi`

	info := &HostInfo{}
	parseOutput(info, raw)

	if info.OS != "Ubuntu 22.04.3 LTS" {
		t.Errorf("OS = %q, want Ubuntu 22.04.3 LTS", info.OS)
	}
	if info.Kernel != "6.5.0-44-generic" {
		t.Errorf("Kernel = %q, want 6.5.0-44-generic", info.Kernel)
	}
	if info.Uptime != "23 days, 4 hours" {
		t.Errorf("Uptime = %q, want 23 days, 4 hours", info.Uptime)
	}
	if info.DiskTotal != "50G" {
		t.Errorf("DiskTotal = %q, want 50G", info.DiskTotal)
	}
	if info.DiskUsed != "32G" {
		t.Errorf("DiskUsed = %q, want 32G", info.DiskUsed)
	}
	if info.DiskPct != "64%" {
		t.Errorf("DiskPct = %q, want 64%%", info.DiskPct)
	}
	if info.Memory != "3.8Gi / 2.1Gi" {
		t.Errorf("Memory = %q, want 3.8Gi / 2.1Gi", info.Memory)
	}
}

func TestParseOutputMacOS(t *testing.T) {
	raw := `---OS---
N/A
---KERNEL---
23.4.0
---UPTIME---
13:02  up 23 days,  4:02, 2 users, load averages: 1.52 1.58 1.63
---DISK---
/dev/disk3s1s1  460Gi  245Gi  215Gi  54% /
---MEMORY---
N/A`

	info := &HostInfo{}
	parseOutput(info, raw)

	if info.OS != "" {
		t.Errorf("OS = %q, want empty for macOS N/A", info.OS)
	}
	if info.Kernel != "23.4.0" {
		t.Errorf("Kernel = %q, want 23.4.0", info.Kernel)
	}
	if info.Uptime != "23 days,  4:02" {
		t.Errorf("Uptime = %q, want '23 days,  4:02'", info.Uptime)
	}
	if info.DiskPct != "54%" {
		t.Errorf("DiskPct = %q, want 54%%", info.DiskPct)
	}
	if info.Memory != "" {
		t.Errorf("Memory = %q, want empty for N/A", info.Memory)
	}
}

func TestParseOutputPartialMissing(t *testing.T) {
	raw := `---OS---
Debian GNU/Linux 12
---KERNEL---
6.1.0-18-amd64
---UPTIME---
---DISK---
---MEMORY---`

	info := &HostInfo{}
	parseOutput(info, raw)

	if info.OS != "Debian GNU/Linux 12" {
		t.Errorf("OS = %q, want Debian GNU/Linux 12", info.OS)
	}
	if info.Uptime != "" {
		t.Errorf("Uptime = %q, want empty", info.Uptime)
	}
	if info.DiskPct != "" {
		t.Errorf("DiskPct = %q, want empty", info.DiskPct)
	}
}

func TestParseOutputEmpty(t *testing.T) {
	info := &HostInfo{}
	parseOutput(info, "")

	if info.OS != "" || info.Kernel != "" || info.Uptime != "" {
		t.Error("expected all empty fields for empty output")
	}
}

func TestParseOutputGarbled(t *testing.T) {
	raw := `some random output
that doesn't have markers
at all`

	info := &HostInfo{}
	parseOutput(info, raw)

	if info.OS != "" || info.Kernel != "" {
		t.Error("expected empty fields for garbled output")
	}
}

func TestParseUptimePFormat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"up 23 days, 4 hours", "23 days, 4 hours"},
		{"up 1 day, 2 hours, 30 minutes", "1 day, 2 hours, 30 minutes"},
		{"up 5 minutes", "5 minutes"},
	}

	for _, tt := range tests {
		got := parseUptime(tt.input)
		if got != tt.want {
			t.Errorf("parseUptime(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseUptimeStandardFormat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"13:02:15 up 23 days, 4:02, 2 users, load average: 0.15, 0.10, 0.09", "23 days, 4:02"},
		{"10:30  up 1 day,  3:15, 1 user, load averages: 1.52 1.58 1.63", "1 day,  3:15"},
	}

	for _, tt := range tests {
		got := parseUptime(tt.input)
		if got != tt.want {
			t.Errorf("parseUptime(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
