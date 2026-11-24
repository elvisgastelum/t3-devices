// Package version holds build metadata for t3-devices.
package version

import "fmt"

// Set at build time via -ldflags "-X github.com/elvisgastelum/t3-devices/internal/version.Version=...".
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// UpdateHint tells the user how to update t3-devices.
const UpdateHint = "Update via the install script, " +
	"`go install github.com/elvisgastelum/t3-devices/cmd/t3-devices@latest`, " +
	"or `mise run install`."

// String returns the one-line version summary.
func String() string {
	return fmt.Sprintf("t3-devices %s (commit %s, built %s)", Version, Commit, Date)
}
