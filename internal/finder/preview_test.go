package finder

import (
	"strings"
	"testing"
	"time"

	"github.com/jinmugo/sls/internal/hostinfo"
)

func TestRenderPreviewNormal(t *testing.T) {
	info := &hostinfo.HostInfo{
		Hostname:  "prod-server",
		OS:        "Ubuntu 22.04",
		Kernel:    "6.5.0-44-generic",
		Uptime:    "23 days, 4 hours",
		DiskTotal: "50G",
		DiskUsed:  "32G",
		DiskPct:   "64%",
		Memory:    "2.1Gi / 3.8Gi",
		FetchedAt: time.Now(),
	}

	result := renderPreview(info, false, 40, 20)

	if !strings.Contains(result, "prod-server") {
		t.Error("expected hostname in output")
	}
	if !strings.Contains(result, "Ubuntu 22.04") {
		t.Error("expected OS in output")
	}
	if !strings.Contains(result, "64%") {
		t.Error("expected disk percentage in output")
	}
	if !strings.Contains(result, "█") {
		t.Error("expected disk bar in output")
	}
}

func TestRenderPreviewLoading(t *testing.T) {
	result := renderPreview(nil, true, 40, 20)

	if !strings.Contains(result, "fetching") {
		t.Error("expected 'fetching' in loading state")
	}
}

func TestRenderPreviewError(t *testing.T) {
	info := &hostinfo.HostInfo{
		Hostname:  "bad-host",
		Error:     "SSH auth failed for bad-host",
		FetchedAt: time.Now(),
	}

	result := renderPreview(info, false, 40, 20)

	if !strings.Contains(result, "unreachable") {
		t.Error("expected 'unreachable' in error state")
	}
}

func TestRenderPreviewNil(t *testing.T) {
	result := renderPreview(nil, false, 40, 20)

	if !strings.Contains(result, "no data") {
		t.Error("expected 'no data' for nil info")
	}
}

func TestRenderPreviewMissingMemory(t *testing.T) {
	info := &hostinfo.HostInfo{
		Hostname:  "mac-server",
		OS:        "",
		Kernel:    "23.4.0",
		DiskTotal: "460Gi",
		DiskUsed:  "245Gi",
		DiskPct:   "54%",
		FetchedAt: time.Now(),
	}

	result := renderPreview(info, false, 40, 20)

	if !strings.Contains(result, "23.4.0") {
		t.Error("expected kernel in output")
	}
	// Memory should not appear since it's empty
	if strings.Contains(result, "Mem") {
		t.Error("expected no memory section for empty memory")
	}
}

func TestRenderPreviewDiskWarning(t *testing.T) {
	info := &hostinfo.HostInfo{
		Hostname:  "full-disk",
		DiskTotal: "50G",
		DiskUsed:  "45G",
		DiskPct:   "90%",
		FetchedAt: time.Now(),
	}

	result := renderPreview(info, false, 40, 20)

	if !strings.Contains(result, "90%") {
		t.Error("expected 90% in output")
	}
}

func TestRenderPreviewNarrowWidth(t *testing.T) {
	result := renderPreview(nil, false, 15, 20)
	if result != "" {
		t.Error("expected empty output for width < 20")
	}
}

func TestRenderDiskBar(t *testing.T) {
	tests := []struct {
		pct  string
		want string // substring to check
	}{
		{"64%", "64%"},
		{"90%", "90%"},
		{"0%", "0%"},
		{"100%", "100%"},
	}

	for _, tt := range tests {
		result := renderBar(tt.pct, 40)
		if !strings.Contains(result, tt.want) {
			t.Errorf("renderBar(%q) missing %q, got %q", tt.pct, tt.want, result)
		}
	}
}
