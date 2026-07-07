package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"time"
)

type Runner struct {
	Timeout time.Duration
	UseSudo bool
	Logger  *slog.Logger
}

type Result struct {
	Command  string
	Args     []string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

func (r Runner) Run(ctx context.Context, command string, args ...string) (Result, error) {
	if command == "" {
		return Result{}, errors.New("empty command")
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name := command
	argv := args
	if r.UseSudo {
		name = "sudo"
		argv = append([]string{"-n", command}, args...)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, name, argv...)
	stdout, stderr := &limitedBuffer{}, &limitedBuffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	duration := time.Since(start)

	result := Result{
		Command:  command,
		Args:     args,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
		Duration: duration,
	}
	if err != nil {
		result.ExitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		}
		if ctx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("command timeout: %s", command)
		}
	}

	if r.Logger != nil {
		r.Logger.Info("command executed", "command", result.Command, "args", result.Args, "exit_code", result.ExitCode, "duration_ms", duration.Milliseconds())
	}
	return result, err
}

type limitedBuffer struct {
	data []byte
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	const max = 64 * 1024
	remaining := max - len(b.data)
	if remaining > 0 {
		if len(p) > remaining {
			b.data = append(b.data, p[:remaining]...)
		} else {
			b.data = append(b.data, p...)
		}
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	return string(b.data)
}
