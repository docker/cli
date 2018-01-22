package progress

import (
	"io"
	"io/ioutil"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// WaitOnServiceOptions to configure the behaviour of WaitOnService
type WaitOnServiceOptions struct {
	Quiet   bool
	Timeout *time.Duration
}

// WaitOnService waits for the service to converge. By default it prints status
// bars to stdout, which can be silenced using options.Quiet.
func WaitOnService(ctx context.Context, cli command.Cli, serviceID string, options WaitOnServiceOptions) error {
	errChan := make(chan error, 2)
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		errChan <- ServiceProgress(ctx, cli.Client(), serviceID, pipeWriter)
	}()

	go func() {
		if options.Quiet {
			_, err := io.Copy(ioutil.Discard, pipeReader)
			errChan <- err
			return
		}
		errChan <- jsonmessage.DisplayJSONMessagesToStream(pipeReader, cli.Out(), nil)
	}()

	if options.Timeout == nil {
		err := <-errChan
		if err != nil {
			return err
		}
		return <-errChan
	}

	select {
	case err := <-errChan:
		return err
	case <-time.After(*options.Timeout):

		return errors.Errorf("timeout (%s) waiting on %s to converge. %s",
			options.Timeout, serviceID, msgOperationContinuingInBackground)
	}
}
