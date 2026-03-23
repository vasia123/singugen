package claude

import (
	"context"
	"io"
	"sync"
)

// FakeConn represents one launch of the fake process.
type FakeConn struct {
	// StdinReader lets tests read what was written to Claude's stdin.
	StdinReader *io.PipeReader
	// StdoutWriter lets tests write events to Claude's stdout.
	StdoutWriter *io.PipeWriter

	stdinWriter  *io.PipeWriter
	stdoutReader *io.PipeReader
	waitCh       chan struct{}
}

// Close closes all pipes in this connection.
func (c *FakeConn) Close() {
	c.stdinWriter.Close()
	c.stdoutReader.Close()
	c.StdinReader.Close()
	c.StdoutWriter.Close()
}

// FakeLauncher is a test double for ProcessLauncher that uses io.Pipe.
// Each Launch creates a fresh set of pipes. Tests receive new connections
// via the Conns channel.
type FakeLauncher struct {
	mu      sync.Mutex
	argsLog [][]string

	// Conns receives a FakeConn each time Launch is called.
	Conns chan *FakeConn

	// WaitErr is the error returned by the wait function.
	WaitErr error
}

func NewFakeLauncher() *FakeLauncher {
	return &FakeLauncher{
		Conns: make(chan *FakeConn, 10),
	}
}

func (f *FakeLauncher) Launch(_ context.Context, args []string) (io.WriteCloser, io.ReadCloser, func() error, error) {
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	conn := &FakeConn{
		StdinReader:  stdinReader,
		StdoutWriter: stdoutWriter,
		stdinWriter:  stdinWriter,
		stdoutReader: stdoutReader,
		waitCh:       make(chan struct{}),
	}

	f.mu.Lock()
	f.argsLog = append(f.argsLog, args)
	f.mu.Unlock()

	f.Conns <- conn

	wait := func() error {
		<-conn.waitCh
		return f.WaitErr
	}

	return stdinWriter, stdoutReader, wait, nil
}

// ArgsLog returns a copy of all args passed to Launch calls.
func (f *FakeLauncher) ArgsLog() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	log := make([][]string, len(f.argsLog))
	copy(log, f.argsLog)
	return log
}

// Launches returns how many times Launch was called.
func (f *FakeLauncher) Launches() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.argsLog)
}
