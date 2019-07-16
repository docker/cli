package service

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/service/progress"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
)

// waitOnService waits for the service to converge. It outputs a progress bar,
// if appropriate based on the CLI flags.
func waitOnService(ctx context.Context, dockerCli command.Cli, serviceID string, quiet bool) error {
	errChan := make(chan error, 1)
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		errChan <- progress.ServiceProgress(ctx, dockerCli.Client(), serviceID, pipeWriter)
	}()

	if quiet {
		go io.Copy(ioutil.Discard, pipeReader)
		return <-errChan
	}

	errFD, errIsTerminal := term.GetFdInfo(dockerCli.Err())
	err := jsonmessage.DisplayJSONMessagesStream(pipeReader, dockerCli.Err(), errFD, errIsTerminal, nil)
	if err == nil {
		err = <-errChan
	}
	return err
}
