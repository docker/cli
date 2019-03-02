package stack

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/stacks/pkg/types"
)

// LoadComposefile parse the composefile specified in the cli and returns its StackCreate
func LoadComposefile(ctx context.Context, dockerCli command.Cli, opts options.Deploy) (*types.StackCreate, error) {
	files, err := getComposeFiles(opts.Composefiles, dockerCli.In())
	if err != nil {
		return nil, err
	}
	input := types.ComposeInput{
		ComposeFiles: files,
	}

	dclient := dockerCli.Client()

	// Get the server to parse them into a StackCreate request
	stackCreate, err := dclient.ParseComposeInput(ctx, input)
	if err != nil {
		return nil, err
	}

	err = buildEnvironment(stackCreate)
	if err != nil {
		return nil, err
	}

	return stackCreate, nil
}

// getComposeFiles takes filenames (or "-") as input, and returns the yaml payloads
func getComposeFiles(composefiles []string, stdin io.Reader) ([]string, error) {
	payloads := []string{}
	for _, filename := range composefiles {
		var bytes []byte
		var err error
		if filename == "-" {
			bytes, err = ioutil.ReadAll(stdin)
		} else {
			bytes, err = ioutil.ReadFile(filename)
		}
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, string(bytes))
	}
	return payloads, nil
}

func buildEnvironment(stackCreate *types.StackCreate) error {
	finalPropertyValues := make([]string, len(stackCreate.Spec.PropertyValues))
	for i, prop := range stackCreate.Spec.PropertyValues {
		s := strings.SplitN(prop, "=", 2)
		val, present := os.LookupEnv(s[0])
		if !present && len(s) == 1 {
			// Not set in the environment, and no default value
			return fmt.Errorf("you must specify a value for variable %s", s[0])
		} else if present {
			// Environment overrides default
			finalPropertyValues[i] = s[0] + "=" + val
		} else {
			// no environment, so use the default
			finalPropertyValues[i] = prop
		}
	}
	stackCreate.Spec.PropertyValues = finalPropertyValues
	return nil
}
