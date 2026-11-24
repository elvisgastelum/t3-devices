package runner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFakeAddRun(t *testing.T) {
	f := NewFake()
	f.Add(Command{Name: "tailscale", Args: []string{"ip"}}, Result{Stdout: []byte("x"), ExitCode: 2})

	res, err := f.Run(context.Background(), Command{Name: "tailscale", Args: []string{"ip"}, Stdin: []byte("ignored")})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if string(res.Stdout) != "x" || res.ExitCode != 2 {
		t.Errorf("Run() = %+v, want stdout x exit 2", res)
	}
}

func TestFakeMiss(t *testing.T) {
	f := NewFake()
	f.Add(Command{Name: "tailscale", Args: []string{"ip", "-4"}}, Result{})

	tests := []Command{
		{Name: "tailscale", Args: []string{"ip", "-6"}},
		{Name: "tailscale", Args: []string{"ip"}},
		{Name: "other"},
	}
	for _, cmd := range tests {
		_, err := f.Run(context.Background(), cmd)
		if !errors.Is(err, ErrNoFixture) {
			t.Errorf("Run(%v) error = %v, want ErrNoFixture", cmd, err)
			continue
		}
		if !strings.Contains(err.Error(), cmd.String()) {
			t.Errorf("error %q does not name command %q", err, cmd)
		}
	}
}

func TestFakeCalls(t *testing.T) {
	f := NewFake()
	f.Add(Command{Name: "t3", Args: []string{"a"}}, Result{})
	cmd := Command{Name: "t3", Args: []string{"a"}, Stdin: []byte("in")}
	_, _ = f.Run(context.Background(), cmd)
	_, _ = f.Run(context.Background(), Command{Name: "t3", Args: []string{"b"}})

	got := f.Calls()
	if len(got) != 2 || !reflect.DeepEqual(got[0], cmd) || got[1].Args[0] != "b" {
		t.Errorf("Calls() = %+v", got)
	}
	got[0].Name = "mutated"
	if f.Calls()[0].Name != "t3" {
		t.Error("Calls() did not return a copy")
	}
}

func TestFakeCanceledContext(t *testing.T) {
	f := NewFake()
	f.Add(Command{Name: "t3"}, Result{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.Run(ctx, Command{Name: "t3"}); !errors.Is(err, context.Canceled) {
		t.Errorf("Run() error = %v, want Canceled", err)
	}
}

func TestLoadFakeTailscale(t *testing.T) {
	f, err := LoadFake(FixtureDir("../../testdata", "tailscale", "1.90.6"))
	if err != nil {
		t.Fatalf("LoadFake() error = %v", err)
	}
	res, err := f.Run(context.Background(), Command{Name: "tailscale", Args: []string{"status", "--json"}})
	if err != nil {
		t.Fatalf("status Run() error = %v", err)
	}
	if !strings.Contains(string(res.Stdout), "example-tailnet.ts.net") {
		t.Error("status stdout missing example-tailnet.ts.net")
	}
	var v map[string]any
	if err := json.Unmarshal(res.Stdout, &v); err != nil {
		t.Errorf("status stdout is not JSON: %v", err)
	}
	res, err = f.Run(context.Background(), Command{Name: "tailscale", Args: []string{"ip", "-4"}})
	if err != nil {
		t.Fatalf("ip Run() error = %v", err)
	}
	if got := string(res.Stdout); got != "100.64.0.1\n" {
		t.Errorf("ip stdout = %q", got)
	}
}

func TestLoadFakeErrors(t *testing.T) {
	tests := []struct {
		name  string
		json  string // empty means no fixtures.json
		files map[string]string
	}{
		{"both stdout and stdout_file", `[{"name":"a","stdout":"x","stdout_file":"o.txt"}]`, map[string]string{"o.txt": "y"}},
		{"stdout_file with ..", `[{"name":"a","stdout_file":"../o.txt"}]`, nil},
		{"stdout_file absolute", `[{"name":"a","stdout_file":"/etc/passwd"}]`, nil},
		{"unknown field", `[{"name":"a","bogus":1}]`, nil},
		{"duplicate entry", `[{"name":"a","args":["x"]},{"name":"a","args":["x"]}]`, nil},
		{"missing fixtures.json", ``, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.json != "" {
				if err := os.WriteFile(filepath.Join(dir, "fixtures.json"), []byte(tt.json), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for name, body := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := LoadFake(dir); err == nil {
				t.Error("LoadFake() error = nil, want error")
			}
		})
	}
}
