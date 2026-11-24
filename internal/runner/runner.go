// Package runner is the seam for running external commands. All external
// commands (tailscale, t3, etc.) go through a CommandRunner so they can be
// faked in tests.
package runner

import (
	"context"
	"strings"
)

// Command describes one external command invocation.
type Command struct {
	Name string
	Args []string
	// Stdin is piped to the command; nil means no stdin.
	Stdin []byte
}

// Result is the captured output of a command that ran.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// CommandRunner runs external commands.
//
// A command that runs and exits non-zero is NOT an error: err is nil and
// Result.ExitCode is set. err is non-nil only when the command cannot be
// started, the context is done, or (for the fake) no fixture matches.
type CommandRunner interface {
	Run(ctx context.Context, cmd Command) (Result, error)
}

// Key returns the fixture key for the command: Name and Args joined with
// "\x00". Stdin is not part of the key.
func (c Command) Key() string {
	parts := append([]string{c.Name}, c.Args...)
	return strings.Join(parts, "\x00")
}

// String returns the name and args space-joined, for error messages.
func (c Command) String() string {
	parts := append([]string{c.Name}, c.Args...)
	return strings.Join(parts, " ")
}
