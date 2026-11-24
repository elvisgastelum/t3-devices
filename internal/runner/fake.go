package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ErrNoFixture is returned (wrapped) by Fake.Run when no fixture matches.
var ErrNoFixture = errors.New("no fixture for command")

// Fake is a CommandRunner that replays canned results.
type Fake struct {
	mu       sync.Mutex
	fixtures map[string]Result
	calls    []Command
}

var _ CommandRunner = (*Fake)(nil)

// NewFake returns an empty Fake.
func NewFake() *Fake {
	return &Fake{fixtures: map[string]Result{}}
}

// Add registers res for cmd, keyed by cmd.Key() (stdin is ignored).
func (f *Fake) Add(cmd Command, res Result) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fixtures[cmd.Key()] = cloneResult(res)
}

// Run records the call and returns a copy of the matching Result.
func (f *Fake) Run(ctx context.Context, cmd Command) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, fmt.Errorf("run %s: %w", cmd, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, Command{
		Name:  cmd.Name,
		Args:  append([]string(nil), cmd.Args...),
		Stdin: bytes.Clone(cmd.Stdin),
	})
	res, ok := f.fixtures[cmd.Key()]
	if !ok {
		return Result{}, fmt.Errorf("run %s: %w", cmd, ErrNoFixture)
	}
	return cloneResult(res), nil
}

// Calls returns a copy of every command passed to Run, in order.
func (f *Fake) Calls() []Command {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Command, len(f.calls))
	for i, c := range f.calls {
		out[i] = Command{Name: c.Name, Args: append([]string(nil), c.Args...), Stdin: bytes.Clone(c.Stdin)}
	}
	return out
}

func cloneResult(r Result) Result {
	return Result{
		Stdout:   bytes.Clone(r.Stdout),
		Stderr:   bytes.Clone(r.Stderr),
		ExitCode: r.ExitCode,
	}
}

// FixtureDir returns the fixture directory for a tool version under root.
func FixtureDir(root, tool, version string) string {
	return filepath.Join(root, tool, version)
}

type fixtureEntry struct {
	Name       string   `json:"name"`
	Args       []string `json:"args"`
	StdoutFile string   `json:"stdout_file"`
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`
	ExitCode   int      `json:"exit_code"`
}

// LoadFake builds a Fake from dir/fixtures.json, a JSON array of entries with
// name, args, stdout_file (relative to dir) or inline stdout, stderr and
// exit_code.
func LoadFake(dir string) (*Fake, error) {
	data, err := os.ReadFile(filepath.Join(dir, "fixtures.json"))
	if err != nil {
		return nil, fmt.Errorf("load fixtures: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var entries []fixtureEntry
	if err := dec.Decode(&entries); err != nil {
		return nil, fmt.Errorf("parse fixtures.json: %w", err)
	}

	f := NewFake()
	for i, e := range entries {
		cmd := Command{Name: e.Name, Args: e.Args}
		if _, dup := f.fixtures[cmd.Key()]; dup {
			return nil, fmt.Errorf("fixtures.json entry %d: duplicate command %q", i, cmd)
		}
		if e.StdoutFile != "" && e.Stdout != "" {
			return nil, fmt.Errorf("fixtures.json entry %d (%s): stdout and stdout_file are mutually exclusive", i, cmd)
		}
		stdout := []byte(e.Stdout)
		if e.StdoutFile != "" {
			if !filepath.IsLocal(e.StdoutFile) {
				return nil, fmt.Errorf("fixtures.json entry %d (%s): stdout_file %q escapes fixture dir", i, cmd, e.StdoutFile)
			}
			stdout, err = os.ReadFile(filepath.Join(dir, e.StdoutFile))
			if err != nil {
				return nil, fmt.Errorf("fixtures.json entry %d (%s): %w", i, cmd, err)
			}
		}
		f.fixtures[cmd.Key()] = Result{Stdout: stdout, Stderr: []byte(e.Stderr), ExitCode: e.ExitCode}
	}
	return f, nil
}
