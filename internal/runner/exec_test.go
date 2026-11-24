package runner

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecRunnerCapturesOutputAndExitCode(t *testing.T) {
	res, err := ExecRunner{}.Run(context.Background(), Command{
		Name: "sh", Args: []string{"-c", "echo out; echo err >&2; exit 3"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got := string(res.Stdout); got != "out\n" {
		t.Errorf("Stdout = %q, want %q", got, "out\n")
	}
	if got := string(res.Stderr); got != "err\n" {
		t.Errorf("Stderr = %q, want %q", got, "err\n")
	}
	if res.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", res.ExitCode)
	}
}

func TestExecRunnerPipesStdin(t *testing.T) {
	res, err := ExecRunner{}.Run(context.Background(), Command{Name: "cat", Stdin: []byte("hello")})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := string(res.Stdout); got != "hello" {
		t.Errorf("Stdout = %q, want %q", got, "hello")
	}
}

func TestExecRunnerMissingBinary(t *testing.T) {
	res, err := ExecRunner{}.Run(context.Background(), Command{Name: "t3-devices-no-such-binary"})
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if res.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", res.ExitCode)
	}
}

func TestExecRunnerContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := ExecRunner{}.Run(ctx, Command{Name: "sleep", Args: []string{"5"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want DeadlineExceeded", err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("Run() took %v, want quick return", d)
	}
}

func TestExecRunnerGrandchildHoldsStdout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := ExecRunner{}.Run(ctx, Command{Name: "sh", Args: []string{"-c", "sleep 8; :"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want DeadlineExceeded", err)
	}
	if d := time.Since(start); d > 4*time.Second {
		t.Errorf("Run() took %v, want return within 4s", d)
	}
}

func TestExecRunnerBackgroundChildKeepsExitCode(t *testing.T) {
	tests := []struct {
		name   string
		script string
		code   int
	}{
		{"success", "(sleep 3 &); echo done; exit 0", 0},
		{"failure", "(sleep 3 &); echo done; exit 4", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			res, err := ExecRunner{}.Run(context.Background(), Command{Name: "sh", Args: []string{"-c", tt.script}})
			if err != nil {
				t.Fatalf("Run() error = %v, want nil", err)
			}
			if res.ExitCode != tt.code {
				t.Errorf("ExitCode = %d, want %d", res.ExitCode, tt.code)
			}
			if got := string(res.Stdout); got != "done\n" {
				t.Errorf("Stdout = %q, want %q", got, "done\n")
			}
			if d := time.Since(start); d > 2500*time.Millisecond {
				t.Errorf("Run() took %v, want return near WaitDelay", d)
			}
		})
	}
}
