package logx

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetOnCleanup closes any open log file when the test ends.
func resetOnCleanup(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { Init(false) })
}

func TestInitDebugWritesRedactedLog(t *testing.T) {
	resetOnCleanup(t)
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)

	const pairingJSON = `{"token":"FAKE-TOKEN-0000-placeholder","code":"FAKE-1234","pairing_code":"ABCD-1234","pair_url":"https://device.example.invalid/pair#token=faketoken0000placeholder","expires_in":300}`
	const link = "https://device.example.invalid/pair#token=faketoken0000placeholder"

	type pairing struct {
		Token   string `json:"Token"`
		PairURL string `json:"PairURL"`
	}

	Init(true)
	Debug("pairing created", "response", pairingJSON)
	Debug(pairingJSON)
	Info("open link", "url", link)
	Info("please open " + link + " on the device")
	Debug("raw bytes", "body", []byte(pairingJSON))
	Debug("struct value", slog.Any("pairing", pairing{
		Token:   "FAKE-TOKEN-0000-placeholder",
		PairURL: link,
	}))
	Debug("camel case key", "pairingCode", "ABCD-1234")
	Init(false)

	path := filepath.Join(state, "t3-devices", "debug.log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("log file is empty")
	}
	got := string(data)

	for _, want := range []string{Redacted, "pairing created", "expires_in"} {
		if !strings.Contains(got, want) {
			t.Errorf("log does not contain %q:\n%s", want, got)
		}
	}
	for _, secret := range []string{
		"FAKE-TOKEN-0000-placeholder",
		"faketoken0000placeholder",
		"/pair#",
		"FAKE-1234",
		"ABCD-1234",
		"token=fake",
	} {
		if strings.Contains(got, secret) {
			t.Errorf("log leaks %q:\n%s", secret, got)
		}
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}
	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir mode = %o, want 700", perm)
	}
}

func TestInitFalseCreatesNothing(t *testing.T) {
	resetOnCleanup(t)
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)

	Init(false)
	Debug("x", "token", "abc")

	entries, err := os.ReadDir(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("state dir has %d entries, want none", len(entries))
	}
}

func TestFallbackToHome(t *testing.T) {
	resetOnCleanup(t)

	cases := []struct {
		name string
		xdg  string
	}{
		{"empty XDG_STATE_HOME", ""},
		{"relative XDG_STATE_HOME", "relative/dir"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("XDG_STATE_HOME", tc.xdg)
			t.Setenv("HOME", home)

			want := filepath.Join(home, ".local", "state", "t3-devices", "debug.log")
			got, err := LogPath()
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("LogPath() = %q, want %q", got, want)
			}

			Init(true)
			Info("hello")
			Init(false)

			data, err := os.ReadFile(want)
			if err != nil {
				t.Fatalf("read log: %v", err)
			}
			if !strings.Contains(string(data), "hello") {
				t.Errorf("log %q does not contain hello", data)
			}
		})
	}
}

func TestInitUnwritableDegrades(t *testing.T) {
	resetOnCleanup(t)
	base := t.TempDir()
	blocker := filepath.Join(base, "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", filepath.Join(blocker, "sub"))

	Init(true)
	Debug("x")
	Info("y", "token", "FAKE-value")
}

func TestWithAttrsAndGroupRedacted(t *testing.T) {
	resetOnCleanup(t)
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)

	Init(true)
	Logger().With("token", "FAKE-secret-value").WithGroup("g").Info("m", "code", "FAKE-9876")
	Init(false)

	data, err := os.ReadFile(filepath.Join(state, "t3-devices", "debug.log"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, Redacted) {
		t.Errorf("log does not contain %q:\n%s", Redacted, got)
	}
	for _, secret := range []string{"FAKE-secret-value", "FAKE-9876"} {
		if strings.Contains(got, secret) {
			t.Errorf("log leaks %q:\n%s", secret, got)
		}
	}
}

type fakeErr struct{ msg string }

func (e *fakeErr) Error() string { return e.msg }

func TestLogEdgeValuesRedactedWithoutPanic(t *testing.T) {
	resetOnCleanup(t)
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	Init(true)

	var nilErr *fakeErr
	Debug("typed nil error", "err", nilErr)
	Debug("struct link", "p", struct{ Link string }{"https://device.example.invalid/pair#v=1&token=FAKESECRET"})
	Debug("map query", "m", map[string]string{"u": "https://device.example.invalid/x?a=<b>&token=FAKESECRET"})
	Debug("unmarshalable", "s", struct {
		Token string
		C     chan int
	}{Token: "FAKESECRET"})
	Debug("nested json", "b", struct{ Body string }{`{"data":"{\"token\":\"FAKESECRET\"}"}`})
	Init(false)

	data, err := os.ReadFile(filepath.Join(state, "t3-devices", "debug.log"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "typed nil error") || !strings.Contains(got, "unmarshalable") {
		t.Errorf("log missing expected records:\n%s", got)
	}
	if strings.Contains(got, "FAKESECRET") {
		t.Errorf("log leaked secret:\n%s", got)
	}
}
