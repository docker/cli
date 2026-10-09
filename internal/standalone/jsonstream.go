//go:build linux

package standalone

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"iter"
	"sync"

	"github.com/moby/moby/api/types/jsonstream"
)

// jsonStream is an in-memory stream of [jsonstream.Message] values that
// satisfies client.ImagePullResponse and client.ImagePushResponse. The
// producer calls send/fail/finish; the CLI consumes it as a newline-delimited
// JSON reader.
type jsonStream struct {
	pr   *io.PipeReader
	pw   *io.PipeWriter
	enc  *json.Encoder
	mu   sync.Mutex
	done chan struct{}
	err  error
}

func newJSONStream() *jsonStream {
	pr, pw := io.Pipe()
	return &jsonStream{pr: pr, pw: pw, enc: json.NewEncoder(pw), done: make(chan struct{})}
}

func (j *jsonStream) send(msg jsonstream.Message) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.encode(msg)
}

// encode writes a message to the stream. Write errors are ignored: they only
// mean that the consumer has gone away, which the producer cannot act on.
func (j *jsonStream) encode(msg jsonstream.Message) {
	if err := j.enc.Encode(msg); err != nil {
		return
	}
}

func (j *jsonStream) status(id, status string) {
	j.send(jsonstream.Message{ID: id, Status: status})
}

func (j *jsonStream) progress(id, status string, current, total int64) {
	j.send(jsonstream.Message{ID: id, Status: status, Progress: &jsonstream.Progress{Current: current, Total: total}})
}

// finish ends the stream. If err is non-nil an error message is emitted so
// that the CLI reports it.
func (j *jsonStream) finish(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	select {
	case <-j.done:
		return
	default:
	}
	j.err = err
	if err != nil {
		j.encode(jsonstream.Message{Error: &jsonstream.Error{Message: err.Error()}})
	}
	close(j.done)
	_ = j.pw.Close()
}

func (j *jsonStream) Read(p []byte) (int, error) { return j.pr.Read(p) }

func (j *jsonStream) Close() error {
	_ = j.pr.Close()
	return nil
}

func (j *jsonStream) Wait(ctx context.Context) error {
	// Drain so the producer is not blocked.
	go func() { _, _ = io.Copy(io.Discard, j.pr) }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-j.done:
		return j.err
	}
}

func (j *jsonStream) JSONMessages(ctx context.Context) iter.Seq2[jsonstream.Message, error] {
	return func(yield func(jsonstream.Message, error) bool) {
		dec := json.NewDecoder(j.pr)
		for {
			var msg jsonstream.Message
			if err := dec.Decode(&msg); err != nil {
				if !errors.Is(err, io.EOF) {
					yield(jsonstream.Message{}, err)
				}
				return
			}
			if ctx.Err() != nil {
				yield(jsonstream.Message{}, ctx.Err())
				return
			}
			if !yield(msg, nil) {
				return
			}
		}
	}
}
