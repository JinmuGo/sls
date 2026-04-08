package hostinfo

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jinmugo/sls/internal/runner"
)

// HostInfo holds cached system information for an SSH host.
type HostInfo struct {
	Hostname   string           `json:"hostname"`
	OS         string           `json:"os"`
	Kernel     string           `json:"kernel"`
	Uptime     string           `json:"uptime"`
	DiskTotal  string           `json:"disk_total"`
	DiskUsed   string           `json:"disk_used"`
	DiskPct    string           `json:"disk_pct"`
	Memory     string           `json:"memory"`
	MemTotal   string           `json:"mem_total,omitempty"`
	MemUsed    string           `json:"mem_used,omitempty"`
	MemPct     string           `json:"mem_pct,omitempty"`
	FetchedAt  time.Time        `json:"fetched_at"`
	Error      string           `json:"error,omitempty"`
}

const fetchCmd = `echo "---OS---" && grep PRETTY_NAME /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"' || echo "N/A" && echo "---KERNEL---" && uname -r 2>/dev/null || echo "N/A" && echo "---UPTIME---" && uptime -p 2>/dev/null || uptime && echo "---DISK---" && df -h / | tail -1 && echo "---MEMORY---" && free -h 2>/dev/null | grep Mem || echo "N/A"`

// Fetch retrieves system information from a remote host via SSH.
func Fetch(ctx context.Context, host string) *HostInfo {
	info := &HostInfo{
		Hostname:  host,
		FetchedAt: time.Now(),
	}

	output, err := runner.Probe(ctx, host, fetchCmd)
	if err != nil {
		info.Error = err.Error()
		return info
	}

	parseOutput(info, string(output))
	info.FillMemoryFields()
	return info
}

var uptimeRe = regexp.MustCompile(`up\s+(.+?)(?:,\s*\d+\s*user|$)`)

func parseOutput(info *HostInfo, raw string) {
	sections := parseSections(raw)

	info.OS = cleanValue(sections["OS"])
	info.Kernel = cleanValue(sections["KERNEL"])
	info.Uptime = parseUptime(sections["UPTIME"])
	parseDisk(info, sections["DISK"])
	info.Memory = parseMemory(sections["MEMORY"])
	parseMemoryFields(info, sections["MEMORY"])
}

func parseSections(raw string) map[string]string {
	sections := make(map[string]string)
	var currentKey string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "---") && strings.HasSuffix(line, "---") {
			currentKey = strings.Trim(line, "-")
			continue
		}
		if currentKey != "" && line != "" {
			if existing, ok := sections[currentKey]; ok {
				sections[currentKey] = existing + "\n" + line
			} else {
				sections[currentKey] = line
			}
		}
	}
	return sections
}

func cleanValue(s string) string {
	s = strings.TrimSpace(s)
	if s == "N/A" || s == "" {
		return ""
	}
	return s
}

func parseUptime(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" {
		return ""
	}

	// uptime -p format: "up 23 days, 4 hours"
	if strings.HasPrefix(s, "up ") {
		return strings.TrimPrefix(s, "up ")
	}

	// standard uptime format: "13:02:15 up 23 days, 4:02, 2 users, ..."
	if m := uptimeRe.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}

	// fallback: return raw but truncated
	if len(s) > 30 {
		return s[:30] + "..."
	}
	return s
}

func parseDisk(info *HostInfo, s string) {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" {
		return
	}
	// df -h output: "Filesystem  Size  Used Avail Use% Mounted"
	// We get the data line: "/dev/sda1  50G  32G  18G  64%  /"
	fields := strings.Fields(s)
	if len(fields) >= 5 {
		info.DiskTotal = fields[1]
		info.DiskUsed = fields[2]
		info.DiskPct = fields[4]
	}
}

func parseMemory(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" {
		return ""
	}
	// free -h output: "Mem:  3.8Gi  2.1Gi  1.2Gi ..."
	fields := strings.Fields(s)
	if len(fields) >= 3 {
		return fields[1] + " / " + fields[2]
	}
	return ""
}

func parseMemoryFields(info *HostInfo, s string) {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" {
		return
	}
	fields := strings.Fields(s)
	if len(fields) >= 3 {
		info.MemTotal = fields[1]
		info.MemUsed = fields[2]
		total := parseSizeToBytes(fields[1])
		used := parseSizeToBytes(fields[2])
		if total > 0 {
			pct := int(used / total * 100)
			info.MemPct = fmt.Sprintf("%d%%", pct)
		}
	}
}

// FillMemoryFields populates MemTotal/MemUsed/MemPct from the Memory string
// when they're not already set (e.g., loaded from older cache entries).
func (info *HostInfo) FillMemoryFields() {
	if info.MemPct != "" || info.Memory == "" {
		return
	}
	parts := strings.SplitN(info.Memory, " / ", 2)
	if len(parts) != 2 {
		return
	}
	info.MemTotal = strings.TrimSpace(parts[0])
	info.MemUsed = strings.TrimSpace(parts[1])
	total := parseSizeToBytes(info.MemTotal)
	used := parseSizeToBytes(info.MemUsed)
	if total > 0 {
		pct := int(used / total * 100)
		info.MemPct = fmt.Sprintf("%d%%", pct)
	}
}

func parseSizeToBytes(s string) float64 {
	s = strings.TrimSpace(s)
	suffixes := []struct {
		suffix string
		mult   float64
	}{
		{"Ti", 1024 * 1024 * 1024 * 1024},
		{"Gi", 1024 * 1024 * 1024},
		{"Mi", 1024 * 1024},
		{"Ki", 1024},
		{"T", 1e12}, {"G", 1e9}, {"M", 1e6}, {"K", 1e3},
		{"B", 1},
	}
	for _, sf := range suffixes {
		if strings.HasSuffix(s, sf.suffix) {
			numStr := strings.TrimSuffix(s, sf.suffix)
			if v, err := strconv.ParseFloat(numStr, 64); err == nil {
				return v * sf.mult
			}
			return 0
		}
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	return 0
}
