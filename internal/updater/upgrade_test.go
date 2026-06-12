package updater

import (
	"reflect"
	"testing"
)

func TestUpgradeCommand(t *testing.T) {
	tests := []struct {
		name     string
		method   Method
		wantArgv []string
		wantSudo bool
		wantOK   bool
	}{
		{
			name:     "brew",
			method:   MethodBrew,
			wantArgv: []string{"brew", "upgrade", "sls"},
			wantSudo: false,
			wantOK:   true,
		},
		{
			name:     "package re-runs install one-liner with sudo",
			method:   MethodPackage,
			wantArgv: []string{"sh", "-c", "curl -fsSL https://package.jinmu.me/install.sh | sudo sh -s sls"},
			wantSudo: true,
			wantOK:   true,
		},
		{
			name:     "go install",
			method:   MethodGoInstall,
			wantArgv: []string{"go", "install", "github.com/jinmugo/sls@latest"},
			wantSudo: false,
			wantOK:   true,
		},
		{
			name:   "binary has no automatic command",
			method: MethodBinary,
			wantOK: false,
		},
		{
			name:   "unknown has no automatic command",
			method: MethodUnknown,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argv, sudo, ok := UpgradeCommand(tt.method)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if !reflect.DeepEqual(argv, tt.wantArgv) {
				t.Errorf("argv = %v, want %v", argv, tt.wantArgv)
			}
			if sudo != tt.wantSudo {
				t.Errorf("needsSudo = %v, want %v", sudo, tt.wantSudo)
			}
		})
	}
}
