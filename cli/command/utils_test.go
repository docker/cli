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
		desc     string
		f        func() error
		expected promptResult
	}{
		{"SIGINT", func() error {
			syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			return nil
		}, promptResult{false, command.ErrPromptTerminated}},
		{"no", func() error {
			_, err := fmt.Fprint(promptWriter, "n\n")
			return err
		}, promptResult{false, nil}},
		{"yes", func() error {
			_, err := fmt.Fprint(promptWriter, "y\n")
			return err
		}, promptResult{true, nil}},
		{"any", func() error {
			_, err := fmt.Fprint(promptWriter, "a\n")
			return err
		}, promptResult{false, nil}},
		{"with space", func() error {
			_, err := fmt.Fprint(promptWriter, " y\n")
			return err
		}, promptResult{true, nil}},
		{"reader closed", func() error {
			return promptReader.Close()
		}, promptResult{false, nil}},
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
