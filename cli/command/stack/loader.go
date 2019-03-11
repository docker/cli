package stack

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/stacks/pkg/types"
)

// LoadComposefile parse the composefile specified in the cli and returns its StackCreate
func LoadComposefile(ctx context.Context, dockerCli command.Cli, opts options.Deploy) (*types.StackCreate, error) {
	files, workingDir, err := getComposeFiles(opts.Composefiles, dockerCli.In())
	if err != nil {
		return nil, err
	}
	input := types.ComposeInput{
		ComposeFiles: files,
	}

	dclient := dockerCli.Client()

	// TODO server side handling of variables not yet suppored
	// so we substitute client-side still
	err = substituteProperties(&input, workingDir)
	if err != nil {
		return nil, err
	}

	// Get the server to parse them into a StackCreate request
	stackCreate, err := dclient.ParseComposeInput(ctx, input)
	if err != nil {
		return nil, err
	}

	// Replace any env_file references with their content before
	// performing the actual create/update
	err = loadEnvFiles(stackCreate, workingDir)
	if err != nil {
		return nil, err
	}

	return stackCreate, nil
}

// getComposeFiles takes filenames (or "-") as input, and returns the yaml payloads
func getComposeFiles(composefiles []string, stdin io.Reader) ([]string, string, error) {
	payloads := []string{}
	var workingDir string
	var err error
	if composefiles[0] == "-" && len(composefiles) == 1 {
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, workingDir, err
		}
	} else {
		absPath, err := filepath.Abs(composefiles[0])
		if err != nil {
			return nil, workingDir, err
		}
		workingDir = filepath.Dir(absPath)
	}
	for _, filename := range composefiles {
		var bytes []byte
		var err error
		if filename == "-" {
			bytes, err = ioutil.ReadAll(stdin)
		} else {
			bytes, err = ioutil.ReadFile(filename)
		}
		if err != nil {
			return nil, workingDir, err
		}
		payloads = append(payloads, string(bytes))
	}
	return payloads, workingDir, nil
}
