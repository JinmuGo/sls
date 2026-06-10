package consts

const (
	SSHConfigHost     = "Host"
	SSHConfigHostName = "HostName"
	SSHConfigUser     = "User"
	SSHConfigPort     = "Port"
)

var RequiredSSHConfigOptions = map[string]struct{}{
	SSHConfigHost:     {},
	SSHConfigHostName: {},
	SSHConfigUser:     {},
}

// AllSSHConfigOptions lists per-host keywords offered when adding an option to a
// Host block. "Host" and "Match" are intentionally excluded: they are structural
// block headers, not per-host directives, and writing them as a host's key/value
// would corrupt ~/.ssh/config.
var AllSSHConfigOptions = map[string]struct{}{
	"HostName":                         {},
	"User":                             {},
	"Port":                             {},
	"IdentityFile":                     {},
	"ProxyCommand":                     {},
	"ProxyJump":                        {},
	"ForwardAgent":                     {},
	"ForwardX11":                       {},
	"ForwardX11Trusted":                {},
	"LocalForward":                     {},
	"RemoteForward":                    {},
	"ServerAliveInterval":              {},
	"ServerAliveCountMax":              {},
	"StrictHostKeyChecking":            {},
	"UserKnownHostsFile":               {},
	"GlobalKnownHostsFile":             {},
	"ControlMaster":                    {},
	"ControlPath":                      {},
	"ControlPersist":                   {},
	"Compression":                      {},
	"Ciphers":                          {},
	"MACs":                             {},
	"KexAlgorithms":                    {},
	"LogLevel":                         {},
	"AddKeysToAgent":                   {},
	"AddressFamily":                    {},
	"BatchMode":                        {},
	"BindAddress":                      {},
	"CheckHostIP":                      {},
	"DynamicForward":                   {},
	"Include":                          {},
	"SendEnv":                          {},
	"SetEnv":                           {},
	"TCPKeepAlive":                     {},
	"PermitLocalCommand":               {},
	"VisualHostKey":                    {},
	"CertificateFile":                  {},
	"IdentitiesOnly":                   {},
	"PreferredAuthentications":         {},
	"GSSAPIAuthentication":             {},
	"GSSAPIDelegateCredentials":        {},
	"HashKnownHosts":                   {},
	"CanonicalizeHostname":             {},
	"CanonicalDomains":                 {},
	"CanonicalizeFallbackLocal":        {},
	"CanonicalizeMaxDots":              {},
	"CanonicalizePermittedCNAMEs":      {},
	"RekeyLimit":                       {},
	"PermitRemoteOpen":                 {},
	"NoHostAuthenticationForLocalhost": {},
	"ObscureKeystrokeTiming":           {},
	"UpdateHostKeys":                   {},
	"Tag":                              {},
	"Tunnel":                           {},
	"TunnelDevice":                     {},
	"XAuthLocation":                    {},
}
