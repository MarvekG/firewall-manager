package command

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunnerCapturesStdout(t *testing.T) {
	runner := Runner{Timeout: time.Second}
	result, err := runner.Run(context.Background(), "sh", "-c", "printf hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.Stdout != "hello" || result.ExitCode != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestRunnerCapturesExitCodeAndStderr(t *testing.T) {
	runner := Runner{Timeout: time.Second}
	result, err := runner.Run(context.Background(), "sh", "-c", "printf problem >&2; exit 7")
	if err == nil {
		t.Fatalf("expected command error")
	}
	if result.ExitCode != 7 || result.Stderr != "problem" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestRunnerTimeout(t *testing.T) {
	runner := Runner{Timeout: 10 * time.Millisecond}
	_, err := runner.Run(context.Background(), "sh", "-c", "sleep 1")
	if err == nil || !strings.Contains(err.Error(), "command timeout") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestLimitedBufferCapsOutput(t *testing.T) {
	buf := &limitedBuffer{}
	chunk := strings.Repeat("a", 70*1024)
	n, err := buf.Write([]byte(chunk))
	if err != nil || n != len(chunk) {
		t.Fatalf("unexpected write result n=%d err=%v", n, err)
	}
	if len(buf.String()) != 64*1024 {
		t.Fatalf("expected capped buffer, got %d", len(buf.String()))
	}
}
