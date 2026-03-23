package claude

import (
	"context"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// ProcessLauncher abstracts how the claude binary is started.
// Production uses ExecLauncher; tests use FakeLauncher.
type ProcessLauncher interface {
	Launch(ctx context.Context, args []string) (stdin io.WriteCloser, stdout io.ReadCloser, wait func() error, err error)
}

// ExecLauncher starts a real process via os/exec.
type ExecLauncher struct {
	Binary string
}

func NewExecLauncher(binary string) *ExecLauncher {
	return &ExecLauncher{Binary: binary}
}

func (l *ExecLauncher) Launch(ctx context.Context, args []string) (io.WriteCloser, io.ReadCloser, func() error, error) {
	cmd := exec.CommandContext(ctx, l.Binary, args...)
	cmd.Stderr = os.Stderr

	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 5 * time.Second

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, nil, nil, err
	}

	return stdin, stdout, cmd.Wait, nil
}
