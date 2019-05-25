package store

import (
	"fmt"
	"io"
	"io/ioutil"
)

// limitedReader a wrapper on io.Reader for error handling of ReadAll
type limitedReader struct {
	r     io.Reader
	limit int64
	n     int64
	err   error
}

// newLimitedReader creates a limited reader with defined byte limit.
// This basically extends io.LimitReader with proper errors as io.LimitReader only errors with EOF.
func newLimitedReader(r io.Reader, l int64) io.Reader {
	return &limitedReader{r: r, limit: l, n: l}
}

// limitedReadAll reads all content of a limited reader.
func limitedReadAll(r io.Reader, limit int64) ([]byte, error) {
	r = newLimitedReader(r, limit)
	return ioutil.ReadAll(r)
}

func (lr *limitedReader) Read(p []byte) (int, error) {
	if lr.err != nil {
		return 0, lr.err
	}

	if len(p) == 0 {
		return 0, nil
	}

	// Capping the read to remaining(n) + 1 as it will be sufficient enough to indicate if we hit the limit
	if int64(len(p)) > lr.n+1 {
		p = p[:lr.n+1]
	}

	n, err := lr.r.Read(p)

	if int64(n) <= lr.n {
		lr.n -= int64(n)
		lr.err = err
		return n, err
	}

	n = int(lr.n)
	lr.n = 0

	lr.err = fmt.Errorf("read exceeds the defined limit of %d on the reader", lr.limit)
	return n, lr.err
}
