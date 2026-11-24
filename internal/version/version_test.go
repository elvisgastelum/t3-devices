package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	oldV, oldC, oldD := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = oldV, oldC, oldD })

	Version, Commit, Date = "v1.2.3", "abc1234", "2026-01-02T03:04:05Z"
	want := "t3-devices v1.2.3 (commit abc1234, built 2026-01-02T03:04:05Z)"
	if got := String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestStringDefaults(t *testing.T) {
	oldV, oldC, oldD := Version, Commit, Date
	t.Cleanup(func() { Version, Commit, Date = oldV, oldC, oldD })

	Version, Commit, Date = "dev", "none", "unknown"
	want := "t3-devices dev (commit none, built unknown)"
	if got := String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestUpdateHintNamesAllMethods(t *testing.T) {
	for _, want := range []string{
		"install script",
		"go install github.com/elvisgastelum/t3-devices/cmd/t3-devices@latest",
		"mise run install",
	} {
		if !strings.Contains(UpdateHint, want) {
			t.Errorf("UpdateHint %q does not mention %q", UpdateHint, want)
		}
	}
}
