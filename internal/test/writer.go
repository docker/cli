package test

import (
	"io"
)

type writerWithHook struct {
	actualWriter io.Writer
	hook         func([]byte)
}

func (w *writerWithHook) Write(p []byte) (n int, err error) {
	defer w.hook(p)
	return w.actualWriter.Write(p)
}

var _ io.Writer = (*writerWithHook)(nil)

// NewWriterWithHook returns a io.Writer that still
// writes to the actualWriter but also calls the hook function
// after every write. It is useful to use this function when
// you need to wait for a writer to complete writing inside a test.
func NewWriterWithHook(actualWriter io.Writer, hook func([]byte)) *writerWithHook {
	return &writerWithHook{actualWriter: actualWriter, hook: hook}
}
