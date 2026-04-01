package container

import (
	"io"
	"os"
)

// verboseWriter is the destination for verbose/debug output.
// Defaults to os.Stderr.
var verboseWriter io.Writer = os.Stderr
