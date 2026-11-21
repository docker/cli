package stack

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

// logColors is the rotating palette used to distinguish services, in the
// same spirit as "docker compose logs".
var logColors = []string{"36", "33", "32", "35", "34", "96", "93", "92", "95", "94"}

// logPrefix returns the (optionally colored) per-line prefix for a service,
// padded so that log lines of all services align.
func logPrefix(name string, idx, maxLen int, noColor bool) []byte {
	padded := fmt.Sprintf("%-*s | ", maxLen, name)
	if noColor {
		return []byte(padded)
	}
	c := logColors[idx%len(logColors)]
	return []byte(fmt.Sprintf("\x1b[%sm%s\x1b[0m", c, padded))
}

// prefixWriter prefixes every complete line written to it. Partial lines are
// buffered until a newline arrives, so a line is never split between two
// writes, and the shared mutex keeps lines from concurrent services intact.
type prefixWriter struct {
	out    io.Writer
	prefix []byte
	mu     *sync.Mutex
	buf    bytes.Buffer
}

func newPrefixWriter(out io.Writer, prefix []byte, mu *sync.Mutex) *prefixWriter {
	return &prefixWriter{out: out, prefix: prefix, mu: mu}
}

func (w *prefixWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadBytes('\n')
		if err != nil {
			// no complete line yet: keep the remainder buffered
			w.buf.Write(line)
			return len(p), nil
		}
		w.mu.Lock()
		_, werr := w.out.Write(append(w.prefix, line...))
		w.mu.Unlock()
		if werr != nil {
			return len(p), werr
		}
	}
}
