package service

import (
	"context"
	"io"

	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/command/internal/service/progress"
	"github.com/docker/cli/internal/jsonstream"
)

// WaitOnService waits for the service to converge. It outputs a progress bar,
// if appropriate based on the CLI flags.
//
// Deprecated: This function will be removed from the Docker CLI's
// public facing API. External code should avoid relying on it.
func WaitOnService(ctx context.Context, dockerCLI cli.Cli, serviceID string, quiet bool) error {
	errChan := make(chan error, 1)
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		errChan <- progress.ServiceProgress(ctx, dockerCLI.Client(), serviceID, pipeWriter)
	}()

	if quiet {
		go io.Copy(io.Discard, pipeReader)
		return <-errChan
	}

	err := jsonstream.Display(ctx, pipeReader, dockerCLI.Out())
	if err == nil {
		err = <-errChan
	}
	return err
}
