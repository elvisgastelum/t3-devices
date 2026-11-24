package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// ExecRunner runs commands on the host with os/exec. The zero value is usable.
type ExecRunner struct{}

var _ CommandRunner = ExecRunner{}

// Run executes cmd and captures its output.
func (ExecRunner) Run(ctx context.Context, cmd Command) (Result, error) {
	c := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	// Bound the wait for output pipes after the process exits or ctx is
	// canceled, so a grandchild that keeps stdout/stderr open cannot hang Run.
	c.WaitDelay = time.Second
	if cmd.Stdin != nil {
		c.Stdin = bytes.NewReader(cmd.Stdin)
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	res := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if ctxErr := ctx.Err(); ctxErr != nil {
		res.ExitCode = -1
		if c.ProcessState != nil {
			res.ExitCode = c.ProcessState.ExitCode()
		}
		return res, fmt.Errorf("run %s: %w", cmd, ctxErr)
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			return res, nil
		}
		// The command exited but a background process kept its output
		// pipes open past WaitDelay: the exit status is still real.
		if errors.Is(err, exec.ErrWaitDelay) && c.ProcessState != nil {
			res.ExitCode = c.ProcessState.ExitCode()
			return res, nil
		}
		res.ExitCode = -1
		return res, fmt.Errorf("run %s: %w", cmd, err)
	}
	return res, nil
}
