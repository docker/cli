package types

import "io"

// CloseWriter allows for a write to be closed
type CloseWriter interface {
	CloseWrite() error
}

// ReadWriteCloseCloseWriter extends io.ReadWriteCloser with CloseWriter
type ReadWriteCloseCloseWriter interface {
	io.ReadWriteCloser
	CloseWriter
}
