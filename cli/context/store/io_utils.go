package store

import (
	"errors"
	"io"
)

var errLimitExceeded = errors.New("read exceeds the defined limit")

// limitedReader is a fork of [io.LimitedReader] to override Read.
type limitedReader struct {
	R io.Reader
	N int64 // max bytes remaining
}

// Read is a fork of [io.LimitedReader.Read] that returns an error when limit exceeded.
func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.N < 0 {
		return 0, errLimitExceeded
	}
	// have to cap N + 1 otherwise we won't hit limit err
	if int64(len(p)) > l.N+1 {
		p = p[0 : l.N+1]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	if l.N < 0 {
		// The limit must take precedence over the underlying error: a reader
		// is allowed to return data together with [io.EOF], and returning that
		// EOF here would make [io.ReadAll] report success for content that was
		// silently truncated.
		return n, errLimitExceeded
	}
	return n, err
}
