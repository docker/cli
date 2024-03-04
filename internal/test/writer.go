package test

import (
	"io"
)

// WriterWithHook is an io.Writer that calls a hook function
// after every write.
// This is useful in testing to wait for a write to complete,
// or to check what was written.
// To create a WriterWithHook use the NewWriterWithHook function.
type WriterWithHook struct {
	actualWriter io.Writer
	hook         func([]byte)
}

// Write writes p to the actual writer and then calls the hook function.
func (w *WriterWithHook) Write(p []byte) (n int, err error) {
	defer w.hook(p)
	return w.actualWriter.Write(p)
}

var _ io.Writer = (*WriterWithHook)(nil)

// NewWriterWithHook returns a new WriterWithHook that still writes to the actualWriter
// but also calls the hook function after every write.
// The hook function is useful for testing, or waiting for a write to complete.
func NewWriterWithHook(actualWriter io.Writer, hook func([]byte)) *WriterWithHook {
	return &WriterWithHook{actualWriter: actualWriter, hook: hook}
}
