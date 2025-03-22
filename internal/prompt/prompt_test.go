package prompt_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/prompt"
	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
)

func TestReadInput(t *testing.T) {
	t.Run("cancelling the context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		reader, _ := io.Pipe()

		buf := new(bytes.Buffer)
		bufioWriter := bufio.NewWriter(buf)

		wroteHook := make(chan struct{}, 1)
		promptOut := test.NewWriterWithHook(bufioWriter, func(p []byte) {
			wroteHook <- struct{}{}
		})

		promptErr := make(chan error, 1)
		go func() {
			_, err := prompt.ReadInput(ctx, streams.NewIn(reader), streams.NewOut(promptOut), "Enter something")
			promptErr <- err
		}()

		select {
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for prompt to write to buffer")
		case <-wroteHook:
			cancel()
		}

		select {
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for prompt to be canceled")
		case err := <-promptErr:
			assert.ErrorIs(t, err, prompt.ErrTerminated)
		}
	})

	t.Run("user input should be properly trimmed", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)

		reader, writer := io.Pipe()

		buf := new(bytes.Buffer)
		bufioWriter := bufio.NewWriter(buf)

		wroteHook := make(chan struct{}, 1)
		promptOut := test.NewWriterWithHook(bufioWriter, func(p []byte) {
			wroteHook <- struct{}{}
		})

		go func() {
			<-wroteHook
			_, _ = writer.Write([]byte("  foo  \n"))
		}()

		answer, err := prompt.ReadInput(ctx, streams.NewIn(reader), streams.NewOut(promptOut), "Enter something")
		assert.NilError(t, err)
		assert.Equal(t, answer, "foo")
	})
}

func TestConfirm(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	type promptResult struct {
		result bool
		err    error
	}

	buf := new(bytes.Buffer)
	bufioWriter := bufio.NewWriter(buf)

	var (
		promptWriter *io.PipeWriter
		promptReader *io.PipeReader
	)

	defer func() {
		if promptWriter != nil {
			_ = promptWriter.Close()
		}
		if promptReader != nil {
			_ = promptReader.Close()
		}
	}()

	for _, tc := range []struct {
		desc     string
		f        func() error
		expected promptResult
	}{
		{
			desc: "SIGINT",
			f: func() error {
				_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
				return nil
			},
			expected: promptResult{false, prompt.ErrTerminated},
		},
		{
			desc: "no",
			f: func() error {
				_, err := fmt.Fprintln(promptWriter, "n")
				return err
			},
			expected: promptResult{false, nil},
		},
		{
			desc: "yes",
			f: func() error {
				_, err := fmt.Fprintln(promptWriter, "y")
				return err
			},
			expected: promptResult{true, nil},
		},
		{
			desc: "any",
			f: func() error {
				_, err := fmt.Fprintln(promptWriter, "a")
				return err
			},
			expected: promptResult{false, nil},
		},
		{
			desc: "with space",
			f: func() error {
				_, err := fmt.Fprintln(promptWriter, " y")
				return err
			},
			expected: promptResult{true, nil},
		},
		{
			desc: "reader closed",
			f: func() error {
				return promptReader.Close()
			},
			expected: promptResult{false, nil},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			notifyCtx, notifyCancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
			t.Cleanup(notifyCancel)

			buf.Reset()
			promptReader, promptWriter = io.Pipe()

			wroteHook := make(chan struct{}, 1)
			promptOut := test.NewWriterWithHook(bufioWriter, func(p []byte) {
				wroteHook <- struct{}{}
			})

			result := make(chan promptResult, 1)
			go func() {
				r, err := prompt.Confirm(notifyCtx, promptReader, promptOut, "")
				result <- promptResult{r, err}
			}()

			select {
			case <-time.After(100 * time.Millisecond):
			case <-wroteHook:
			}

			assert.NilError(t, bufioWriter.Flush())
			assert.Equal(t, strings.TrimSpace(buf.String()), "Are you sure you want to proceed? [y/N]")

			// wait for the Prompt to write to the buffer
			drainChannel(ctx, wroteHook)

			assert.NilError(t, tc.f())

			select {
			case <-time.After(500 * time.Millisecond):
				t.Fatal("timeout waiting for prompt result")
			case r := <-result:
				assert.Equal(t, r, tc.expected)
			}
		})
	}
}

func drainChannel(ctx context.Context, ch <-chan struct{}) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
			}
		}
	}()
}
