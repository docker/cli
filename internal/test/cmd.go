package test

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/streams"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func TerminatePrompt(ctx context.Context, t *testing.T, cmd *cobra.Command, cli *FakeCli) {
	t.Helper()

	errChan := make(chan error)
	defer close(errChan)

	// wrap the out stream to detect when the prompt is ready
	writerHookChan := make(chan struct{})
	defer close(writerHookChan)

	outStream := streams.NewOut(NewWriterWithHook(cli.OutBuffer(), func(p []byte) {
		writerHookChan <- struct{}{}
	}))
	cli.SetOut(outStream)

	r, _, err := os.Pipe()
	assert.NilError(t, err)
	cli.SetIn(streams.NewIn(r))

	go func() {
		errChan <- cmd.ExecuteContext(ctx)
	}()

	writeCtx, writeCancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer writeCancel()

	// wait for the prompt to be ready
	select {
	case <-writeCtx.Done():
		t.Fatalf("command %s did not write prompt to stdout", cmd.Name())
	case <-writerHookChan:
		// drain the channel for future buffer writes
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-writerHookChan:
				}
			}
		}()
	}

	assert.Check(t, cli.OutBuffer().Len() > 0)

	// a small delay to ensure the plugin is prompting
	time.Sleep(100 * time.Microsecond)

	errCtx, errCancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer errCancel()

	// sigint and sigterm are caught by the prompt
	// this allows us to gracefully exit the prompt with a 0 exit code
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	select {
	case <-errCtx.Done():
		t.Logf("command stdout:\n%s\n", cli.OutBuffer().String())
		t.Logf("command stderr:\n%s\n", cli.ErrBuffer().String())
		t.Fatalf("command %s did not return after SIGINT", cmd.Name())
	case err := <-errChan:
		assert.ErrorIs(t, err, command.ErrPromptTerminated)
	}
}
