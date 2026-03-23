package claude

import "io"

func newTestPipe() (*io.PipeReader, *io.PipeWriter) {
	return io.Pipe()
}
