package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/elvisgastelum/t3-devices/internal/version"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestVersionOutput(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"version"}} {
		out, err := run(t, args...)
		if err != nil {
			t.Fatalf("%v: unexpected error: %v", args, err)
		}
		if !strings.Contains(out, version.Version) {
			t.Errorf("%v: output %q does not contain version %q", args, out, version.Version)
		}
		if !strings.Contains(out, version.UpdateHint) {
			t.Errorf("%v: output %q does not contain update hint", args, out)
		}
	}
}

func TestDebugHelp(t *testing.T) {
	out, err := run(t, "--debug", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "--debug") {
		t.Errorf("help output %q does not mention --debug", out)
	}
}

// fakeRunTUI swaps runTUI for a fake that records whether it was called.
func fakeRunTUI(t *testing.T) *bool {
	t.Helper()
	called := false
	orig := runTUI
	runTUI = func() error {
		called = true
		return nil
	}
	t.Cleanup(func() { runTUI = orig })
	return &called
}

// fakeInitLogging swaps initLogging for a fake that records each debug flag it is called with.
func fakeInitLogging(t *testing.T) *[]bool {
	t.Helper()
	var calls []bool
	orig := initLogging
	initLogging = func(debug bool) {
		calls = append(calls, debug)
	}
	t.Cleanup(func() { initLogging = orig })
	return &calls
}

func TestNoArgsRunsTUI(t *testing.T) {
	called := fakeRunTUI(t)
	if _, err := run(t); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !*called {
		t.Error("expected runTUI to be called")
	}
}

func TestDebugFlagInitsLogging(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		debug bool
	}{
		{[]string{"--debug"}, true},
		{nil, false},
	} {
		fakeRunTUI(t)
		calls := fakeInitLogging(t)
		if _, err := run(t, tc.args...); err != nil {
			t.Fatalf("%v: unexpected error: %v", tc.args, err)
		}
		if len(*calls) != 1 || (*calls)[0] != tc.debug {
			t.Errorf("%v: initLogging calls = %v, want one call with debug=%v", tc.args, *calls, tc.debug)
		}
	}
}

func TestRejectsPositionalArgs(t *testing.T) {
	if _, err := run(t, "bogus"); err == nil {
		t.Fatal("expected error for unknown argument")
	}
}
