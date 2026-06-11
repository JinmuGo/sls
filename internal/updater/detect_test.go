package updater

import "testing"

func envFunc(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDetectMethod(t *testing.T) {
	tests := []struct {
		name    string
		exePath string
		goos    string
		env     map[string]string
		want    Method
	}{
		{
			name:    "homebrew apple silicon cellar",
			exePath: "/opt/homebrew/Cellar/sls/1.1.1/bin/sls",
			goos:    "darwin",
			want:    MethodBrew,
		},
		{
			name:    "homebrew bin symlink prefix unresolved",
			exePath: "/opt/homebrew/bin/sls",
			goos:    "darwin",
			want:    MethodBrew,
		},
		{
			name:    "homebrew intel mac cellar",
			exePath: "/usr/local/Cellar/sls/1.1.1/bin/sls",
			goos:    "darwin",
			want:    MethodBrew,
		},
		{
			name:    "linuxbrew cellar",
			exePath: "/home/linuxbrew/.linuxbrew/Cellar/sls/1.1.1/bin/sls",
			goos:    "linux",
			want:    MethodBrew,
		},
		{
			name:    "go install via GOBIN",
			exePath: "/custom/gobin/sls",
			goos:    "darwin",
			env:     map[string]string{"GOBIN": "/custom/gobin"},
			want:    MethodGoInstall,
		},
		{
			name:    "go install via GOPATH",
			exePath: "/home/u/go/bin/sls",
			goos:    "linux",
			env:     map[string]string{"GOPATH": "/home/u/go"},
			want:    MethodGoInstall,
		},
		{
			name:    "go install via HOME default",
			exePath: "/home/u/go/bin/sls",
			goos:    "linux",
			env:     map[string]string{"HOME": "/home/u"},
			want:    MethodGoInstall,
		},
		{
			name:    "linux package manager in usr bin",
			exePath: "/usr/bin/sls",
			goos:    "linux",
			want:    MethodPackage,
		},
		{
			name:    "linux manual binary in usr local bin",
			exePath: "/usr/local/bin/sls",
			goos:    "linux",
			want:    MethodBinary,
		},
		{
			name:    "darwin usr bin is not package",
			exePath: "/usr/bin/sls",
			goos:    "darwin",
			want:    MethodBinary,
		},
		{
			name:    "arbitrary path is binary",
			exePath: "/home/u/bin/sls",
			goos:    "linux",
			want:    MethodBinary,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectMethod(tt.exePath, tt.goos, envFunc(tt.env))
			if got != tt.want {
				t.Errorf("detectMethod(%q, %q) = %v, want %v", tt.exePath, tt.goos, got, tt.want)
			}
		})
	}
}

func TestMethodString(t *testing.T) {
	tests := []struct {
		m    Method
		want string
	}{
		{MethodBrew, "Homebrew"},
		{MethodPackage, "package manager"},
		{MethodGoInstall, "go install"},
		{MethodBinary, "binary download"},
		{MethodUnknown, "unknown"},
	}
	for _, tt := range tests {
		if got := tt.m.String(); got != tt.want {
			t.Errorf("Method(%d).String() = %q, want %q", tt.m, got, tt.want)
		}
	}
}
