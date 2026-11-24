package runner

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runCheck runs the fixture checker on dir and returns its exit code and
// combined output.
func runCheck(t *testing.T, dir string) (int, string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not on PATH")
	}
	out, err := exec.Command("bash", "../../scripts/check-fixtures.sh", dir).CombinedOutput()
	if err == nil {
		return 0, string(out)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("running check-fixtures.sh: %v", err)
	}
	return exitErr.ExitCode(), string(out)
}

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "tool", "1.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tool", "1.0", "out.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCheckFixturesClean(t *testing.T) {
	content := "host example-tailnet.ts.net\n" +
		"node box.example-tailnet.ts.net\n" +
		"https://x/pair#token=REDACTED\n" +
		"https://x/pair#token=EXAMPLE-TOKEN\n" +
		"https://x/?u=pair%23token%3DREDACTED\n" +
		"key tskey-REDACTED and tskey-auth-EXAMPLE-TOKEN\n" +
		"id 00000000-0000-0000-0000-000000000000\n"
	code, out := runCheck(t, writeFixture(t, content))
	if code != 0 {
		t.Errorf("exit code = %d, want 0; output:\n%s", code, out)
	}
}

func TestCheckFixturesRejects(t *testing.T) {
	tests := []struct {
		name    string
		content string
		rule    string
	}{
		{"tailnet host", "host " + "mybox.tail" + "a1b2c3.ts.net\n", "tailnet name"},
		{"pairing token", "https://x/pair#" + "token=" + "abcdef\n", "pairing token"},
		{"tailscale key", "key " + "tskey-" + "auth-kABC123CNTRL-xyz\n", "tailscale key"},
		{"long hex", "key " + strings.Repeat("a1b2", 10) + "\n", "token-like string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out := runCheck(t, writeFixture(t, tt.content))
			if code != 1 {
				t.Errorf("exit code = %d, want 1; output:\n%s", code, out)
			}
			if !strings.Contains(out, tt.rule) {
				t.Errorf("output does not name rule %q:\n%s", tt.rule, out)
			}
		})
	}
}

func TestCheckFixturesRejectsCaseAndBinary(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"nul byte", "\x00 host " + "mybox.tail" + "abc1.ts.net\n"},
		{"uppercase pairing", "https://x/PAIR#" + "TOKEN=" + "abc\n"},
		{"url-encoded pairing", "https://x/?u=pair%23" + "token=" + "abc\n"},
		{"fully url-encoded pairing", "https://x/?u=pair%23" + "token%3D" + "abc\n"},
		{"uppercase tskey", "key " + "TSKEY-" + "AUTH-" + "kabc\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out := runCheck(t, writeFixture(t, tt.content))
			if code != 1 {
				t.Errorf("exit code = %d, want 1; output:\n%s", code, out)
			}
		})
	}
}

func TestCheckFixturesUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read any file")
	}
	dir := writeFixture(t, "clean\n")
	f := filepath.Join(dir, "tool", "1.0", "out.txt")
	if err := os.Chmod(f, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(f, 0o644) })
	code, out := runCheck(t, dir)
	if code != 2 {
		t.Errorf("exit code = %d, want 2; output:\n%s", code, out)
	}
}

func TestCheckFixturesFollowsSymlinks(t *testing.T) {
	outside := t.TempDir()
	target := filepath.Join(outside, "bad.txt")
	if err := os.WriteFile(target, []byte("host "+"mybox.tail"+"abc1.ts.net\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(target, filepath.Join(dir, "link.txt")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	code, out := runCheck(t, dir)
	if code != 1 {
		t.Errorf("exit code = %d, want 1; output:\n%s", code, out)
	}
}

func TestCheckFixturesMissingDir(t *testing.T) {
	code, out := runCheck(t, filepath.Join(t.TempDir(), "missing"))
	if code != 2 {
		t.Errorf("exit code = %d, want 2; output:\n%s", code, out)
	}
}

func TestCheckFixturesCommittedTestdata(t *testing.T) {
	code, out := runCheck(t, "../../testdata")
	if code != 0 {
		t.Errorf("committed testdata failed; exit code = %d; output:\n%s", code, out)
	}
}
