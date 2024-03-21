package command_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/test"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
)

func TestStringSliceReplaceAt(t *testing.T) {
	out, ok := command.StringSliceReplaceAt([]string{"abc", "foo", "bar", "bax"}, []string{"foo", "bar"}, []string{"baz"}, -1)
	assert.Assert(t, ok)
	assert.DeepEqual(t, []string{"abc", "baz", "bax"}, out)

	out, ok = command.StringSliceReplaceAt([]string{"foo"}, []string{"foo", "bar"}, []string{"baz"}, -1)
	assert.Assert(t, !ok)
	assert.DeepEqual(t, []string{"foo"}, out)

	out, ok = command.StringSliceReplaceAt([]string{"abc", "foo", "bar", "bax"}, []string{"foo", "bar"}, []string{"baz"}, 0)
	assert.Assert(t, !ok)
	assert.DeepEqual(t, []string{"abc", "foo", "bar", "bax"}, out)

	out, ok = command.StringSliceReplaceAt([]string{"foo", "bar", "bax"}, []string{"foo", "bar"}, []string{"baz"}, 0)
	assert.Assert(t, ok)
	assert.DeepEqual(t, []string{"baz", "bax"}, out)

	out, ok = command.StringSliceReplaceAt([]string{"abc", "foo", "bar", "baz"}, []string{"foo", "bar"}, nil, -1)
	assert.Assert(t, ok)
	assert.DeepEqual(t, []string{"abc", "baz"}, out)

	out, ok = command.StringSliceReplaceAt([]string{"foo"}, nil, []string{"baz"}, -1)
	assert.Assert(t, !ok)
	assert.DeepEqual(t, []string{"foo"}, out)
}

func TestValidateOutputPath(t *testing.T) {
	basedir := t.TempDir()
	dir := filepath.Join(basedir, "dir")
	notexist := filepath.Join(basedir, "notexist")
	err := os.MkdirAll(dir, 0o755)
	assert.NilError(t, err)
	file := filepath.Join(dir, "file")
	err = os.WriteFile(file, []byte("hi"), 0o644)
	assert.NilError(t, err)
	testcases := []struct {
		path string
		err  error
	}{
		{basedir, nil},
		{file, nil},
		{dir, nil},
		{dir + string(os.PathSeparator), nil},
		{notexist, nil},
		{notexist + string(os.PathSeparator), nil},
		{filepath.Join(notexist, "file"), errors.New("does not exist")},
	}

	for _, testcase := range testcases {
		t.Run(testcase.path, func(t *testing.T) {
			err := command.ValidateOutputPath(testcase.path)
			if testcase.err == nil {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, testcase.err.Error())
			}
		})
	}
}

func TestPromptForConfirmation(t *testing.T) {
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
			promptWriter.Close()
		}
		if promptReader != nil {
			promptReader.Close()
		}
	}()

	for _, tc := range []struct {
		desc string
		f    func(*testing.T, context.Context, chan promptResult)
	}{
		{"SIGINT", func(t *testing.T, ctx context.Context, c chan promptResult) {
			t.Helper()

			syscall.Kill(syscall.Getpid(), syscall.SIGINT)

			select {
			case <-ctx.Done():
				t.Fatal("PromptForConfirmation did not return after SIGINT")
			case r := <-c:
				assert.Check(t, !r.result)
				assert.ErrorContains(t, r.err, "prompt terminated")
			}
		}},
		{"no", func(t *testing.T, ctx context.Context, c chan promptResult) {
			t.Helper()

			_, err := fmt.Fprint(promptWriter, "n\n")
			assert.NilError(t, err)

			select {
			case <-ctx.Done():
				t.Fatal("PromptForConfirmation did not return after user input `n`")
			case r := <-c:
				assert.Check(t, !r.result)
				assert.NilError(t, r.err)
			}
		}},
		{"yes", func(t *testing.T, ctx context.Context, c chan promptResult) {
			t.Helper()

			_, err := fmt.Fprint(promptWriter, "y\n")
			assert.NilError(t, err)

			select {
			case <-ctx.Done():
				t.Fatal("PromptForConfirmation did not return after user input `y`")
			case r := <-c:
				assert.Check(t, r.result)
				assert.NilError(t, r.err)
			}
		}},
		{"any", func(t *testing.T, ctx context.Context, c chan promptResult) {
			t.Helper()

			_, err := fmt.Fprint(promptWriter, "a\n")
			assert.NilError(t, err)

			select {
			case <-ctx.Done():
				t.Fatal("PromptForConfirmation did not return after user input `a`")
			case r := <-c:
				assert.Check(t, !r.result)
				assert.NilError(t, r.err)
			}
		}},
		{"with space", func(t *testing.T, ctx context.Context, c chan promptResult) {
			t.Helper()

			_, err := fmt.Fprint(promptWriter, " y\n")
			assert.NilError(t, err)

			select {
			case <-ctx.Done():
				t.Fatal("PromptForConfirmation did not return after user input ` y`")
			case r := <-c:
				assert.Check(t, r.result)
				assert.NilError(t, r.err)
			}
		}},
		{"reader closed", func(t *testing.T, ctx context.Context, c chan promptResult) {
			t.Helper()

			assert.NilError(t, promptReader.Close())

			select {
			case <-ctx.Done():
				t.Fatal("PromptForConfirmation did not return after promptReader was closed")
			case r := <-c:
				assert.Check(t, !r.result)
				assert.NilError(t, r.err)
			}
		}},
	} {
		t.Run("case="+tc.desc, func(t *testing.T) {
			buf.Reset()
			promptReader, promptWriter = io.Pipe()

			wroteHook := make(chan struct{}, 1)
			promptOut := test.NewWriterWithHook(bufioWriter, func(p []byte) {
				wroteHook <- struct{}{}
			})

			result := make(chan promptResult, 1)
			go func() {
				r, err := command.PromptForConfirmation(ctx, promptReader, promptOut, "")
				result <- promptResult{r, err}
			}()

			// wait for the Prompt to write to the buffer
			pollForPromptOutput(ctx, t, wroteHook)
			drainChannel(ctx, wroteHook)

			assert.NilError(t, bufioWriter.Flush())
			assert.Equal(t, strings.TrimSpace(buf.String()), "Are you sure you want to proceed? [y/N]")

			resultCtx, resultCancel := context.WithTimeout(ctx, 500*time.Millisecond)
			defer resultCancel()

			tc.f(t, resultCtx, result)
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

func pollForPromptOutput(ctx context.Context, t *testing.T, wroteHook <-chan struct{}) {
	t.Helper()

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Prompt output was not written to before ctx was cancelled")
			return
		case <-wroteHook:
			return
		}
	}
}
