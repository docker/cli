package service

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/service/progress"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/spf13/pflag"
	"golang.org/x/net/context"
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

	err := jsonmessage.DisplayJSONMessagesToStream(pipeReader, dockerCli.Out(), nil)
	if err == nil {
		err = <-errChan
	}
	return err
}

// warnDetachDefault warns about the --detach flag future change if it's supported.
func warnDetachDefault(err io.Writer, clientVersion string, flags *pflag.FlagSet, msg string) {
	if !flags.Changed("detach") && versions.GreaterThanOrEqualTo(clientVersion, "1.29") {
		fmt.Fprintf(err, "Since --detach=false was not specified, tasks will be %s in the background.\n"+
			"In a future release, --detach=false will become the default.\n", msg)
	}
}
