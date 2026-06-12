package updater

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Method identifies how the running sls binary was installed, which determines
// the correct upgrade command.
type Method int

const (
	MethodUnknown Method = iota
	MethodBrew
	MethodPackage // deb/rpm via the package repo (curl install.sh)
	MethodGoInstall
	MethodBinary // manually downloaded binary
)

// String returns a human-readable label for the install method.
func (m Method) String() string {
	switch m {
	case MethodBrew:
		return "Homebrew"
	case MethodPackage:
		return "package manager"
	case MethodGoInstall:
		return "go install"
	case MethodBinary:
		return "binary download"
	default:
		return "unknown"
	}
}

// brewExclusivePrefixes are Homebrew roots used by no other installer, so any
// binary beneath them is a brew install even before symlink resolution. The
// Intel-mac root (/usr/local) is intentionally excluded: /usr/local/bin is also
// used by manual installs, so it is only treated as brew via its /Cellar/ path.
var brewExclusivePrefixes = []string{
	"/opt/homebrew",              // Apple Silicon
	"/home/linuxbrew/.linuxbrew", // Linuxbrew
}

// DetectMethod determines how the running sls binary was installed by inspecting
// the resolved executable path. Returns MethodUnknown only if the executable
// path cannot be determined.
func DetectMethod() Method {
	exe, err := os.Executable()
	if err != nil {
		return MethodUnknown
	}
	// Resolve symlinks so Homebrew bin shims (/opt/homebrew/bin/sls -> ../Cellar/...)
	// are detected by their real Cellar path.
	if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
		exe = resolved
	}
	return detectMethod(exe, runtime.GOOS, os.Getenv)
}

// detectMethod is the testable core of DetectMethod. exePath is the resolved
// path to the binary; goos is the target OS; env looks up environment variables.
func detectMethod(exePath, goos string, env func(string) string) Method {
	p := filepath.ToSlash(exePath)

	// Homebrew: a /Cellar/ component (covers /usr/local/Cellar on Intel macs and
	// resolved bin shims), or a brew-exclusive prefix (covers unresolved shims).
	if strings.Contains(p, "/Cellar/") {
		return MethodBrew
	}
	for _, prefix := range brewExclusivePrefixes {
		if strings.HasPrefix(p, prefix+"/") {
			return MethodBrew
		}
	}

	// go install: the binary lives in a Go bin directory.
	if slices.Contains(goBinDirs(env), path.Dir(p)) {
		return MethodGoInstall
	}

	// Package repo (deb/rpm) installs to /usr/bin on Linux (nfpm bindir).
	if goos == "linux" && p == "/usr/bin/sls" {
		return MethodPackage
	}

	return MethodBinary
}

// goBinDirs returns the candidate Go binary install directories, honoring GOBIN,
// GOPATH, and the default $HOME/go/bin.
func goBinDirs(env func(string) string) []string {
	var dirs []string
	if gobin := env("GOBIN"); gobin != "" {
		dirs = append(dirs, filepath.ToSlash(gobin))
	}
	if gopath := env("GOPATH"); gopath != "" {
		for _, gp := range filepath.SplitList(gopath) {
			dirs = append(dirs, filepath.ToSlash(filepath.Join(gp, "bin")))
		}
	}
	if home := env("HOME"); home != "" {
		dirs = append(dirs, filepath.ToSlash(filepath.Join(home, "go", "bin")))
	}
	return dirs
}
