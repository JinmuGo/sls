package updater

const (
	// packageUpgradeOneLiner re-runs the documented install script, which installs
	// the latest version through the system package manager (apt/yum). The inner
	// "sudo" handles privilege elevation.
	packageUpgradeOneLiner = "curl -fsSL https://package.jinmu.me/install.sh | sudo sh -s sls"

	// goModulePath is the module path used by `go install`.
	goModulePath = "github.com/jinmugo/sls"

	// ReleasesURL is the GitHub releases page, shown for manual install methods.
	ReleasesURL = "https://github.com/jinmugo/sls/releases"
)

// UpgradeCommand maps an install method to the command that upgrades sls.
//
// The returned argv is executed verbatim by the caller; needsSudo is purely
// informational (it drives the confirmation message) and the caller never
// prepends "sudo" — the package one-liner already contains its own. ok is false
// when there is no safe automatic command (manual binary / unknown installs), in
// which case the caller should print manual instructions instead.
func UpgradeCommand(m Method) (argv []string, needsSudo bool, ok bool) {
	switch m {
	case MethodBrew:
		return []string{"brew", "upgrade", "sls"}, false, true
	case MethodPackage:
		return []string{"sh", "-c", packageUpgradeOneLiner}, true, true
	case MethodGoInstall:
		return []string{"go", "install", goModulePath + "@latest"}, false, true
	default:
		return nil, false, false
	}
}
