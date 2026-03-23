package webapp

import (
	"context"
	"io"
	"log/slog"
)

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type fakeLauncher struct{}

func (fakeLauncher) Launch(_ context.Context, _ []string) (io.WriteCloser, io.ReadCloser, func() error, error) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	_ = stdinR
	_ = stdoutW
	wait := func() error { select {} }
	return stdinW, stdoutR, wait, nil
}
