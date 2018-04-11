package help

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/client"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/docker/docker/api/types"
	"golang.org/x/net/context"
)

func TestHideUnsupportedFeaturesWithAnnotation(t *testing.T) {
	tests := []struct {
		isHidden    bool
		annotations []string
		details     fakeDetails
		name        string
	}{
		{name: "default is visible"},
		{name: "swarm annotation is visible if swarm is activated", isHidden: false, annotations: []string{"swarm"}, details: fakeDetails{orchestrator: "swarm"}},
		{name: "swarm annotation is hidden if kubernetes is activated", isHidden: true, annotations: []string{"swarm"}, details: fakeDetails{orchestrator: "kubernetes", clientExperimental: true}},
		{name: "kubernetes annotation is visible if kubernetes is activated", isHidden: false, annotations: []string{"kubernetes"}, details: fakeDetails{
			orchestrator:       "kubernetes",
			clientExperimental: true},
		},
		{name: "kubernetes annotation is hidden if not experimental", isHidden: true, annotations: []string{"kubernetes"}, details: fakeDetails{orchestrator: "kubernetes"}},
		{name: "kubernetes annotation is hidden if swarm is activated", isHidden: true, annotations: []string{"kubernetes"}, details: fakeDetails{orchestrator: "swarm"}},
		{name: "experimental annotation is hidden if server is not experimental", isHidden: true, annotations: []string{"experimental"}, details: fakeDetails{serverExperimental: false}},
		{name: "experimental annotation is visible if server is experimental", isHidden: false, annotations: []string{"experimental"}, details: fakeDetails{serverExperimental: true}},
		{name: "experimentalCLI annotation is hidden if client is not experimental", isHidden: true, annotations: []string{"experimentalCLI"}, details: fakeDetails{clientExperimental: false}},
		{name: "experimentalCLI annotation is visible if client is experimental", isHidden: false, annotations: []string{"experimentalCLI"}, details: fakeDetails{clientExperimental: true}},
		{name: "multiple annotations not matching all requirements is hidden", isHidden: true, annotations: []string{"experimental", "kubernetes"}, details: fakeDetails{serverExperimental: true}},
		{name: "multiple annotations matching all requirements is visible", isHidden: false, annotations: []string{"experimental", "kubernetes"}, details: fakeDetails{
			serverExperimental: true,
			clientExperimental: true,
			orchestrator:       "kubernetes",
		}},
	}
	for _, test := range tests {
		// Test with flag
		{
			cmd := makeFlaggedCommand(test.annotations...)
			hideUnsupportedFeatures(cmd, &test.details)
			checkFlagVisibility(t, cmd, test.isHidden, test.name)
		}
		// Test with subcommand
		{
			cmd := makeSubCommand(test.annotations...)
			hideUnsupportedFeatures(cmd, &test.details)
			checkSubCommandVisibility(t, cmd, test.isHidden, test.name)
		}
	}
}

func TestHideUnsupportedVersionAnnotation(t *testing.T) {
	tests := []struct {
		isHidden       bool
		minimumVersion string
		clientVersion  string
		name           string
	}{
		{name: "Annotation with greater version than client is hidden", isHidden: true, minimumVersion: "1.25", clientVersion: "1.24"}, // client version can be downgraded depending negotiation results
		{name: "Annotation with lower or equal version than client is visible", isHidden: false, minimumVersion: "1.25", clientVersion: "1.25"},
	}
	for _, test := range tests {
		// Test with flag
		{
			cmd := makeFlaggedCommand()
			cmd.Flags().SetAnnotation("myflag", "version", []string{test.minimumVersion})
			hideUnsupportedFeatures(cmd, &fakeDetails{clientVersion: test.clientVersion})
			checkFlagVisibility(t, cmd, test.isHidden, test.name)
		}
		// Test with subcommand
		{
			cmd := makeSubCommand()
			cmd.Commands()[0].Annotations["version"] = test.minimumVersion
			hideUnsupportedFeatures(cmd, &fakeDetails{clientVersion: test.clientVersion})
			checkSubCommandVisibility(t, cmd, test.isHidden, test.name)
		}
	}
}

func TestHideUnsupportedOsTypeFlag(t *testing.T) {
	cmd := makeFlaggedCommand()
	cmd.Flags().SetAnnotation("myflag", "ostype", []string{"myos"})
	hideUnsupportedFeatures(cmd, &fakeDetails{serverOsType: "otheros"})
	checkFlagVisibility(t, cmd, true, "flag with wrong os type is hidden")
}

func TestUnsupportedFeatures(t *testing.T) {
	tests := []struct {
		isUnsupported bool
		annotations   []string
		details       fakeDetails
		name          string
	}{
		{name: "default is supported"},
		{name: "swarm annotation is supported if swarm is activated", isUnsupported: false, annotations: []string{"swarm"}, details: fakeDetails{orchestrator: "swarm"}},
		{name: "swarm annotation is unsupported if kubernetes is activated", isUnsupported: true, annotations: []string{"swarm"}, details: fakeDetails{
			orchestrator:       "kubernetes",
			clientExperimental: true,
		}},
		{name: "kubernetes annotation is supported if kubernetes is activated", isUnsupported: false, annotations: []string{"kubernetes"}, details: fakeDetails{
			orchestrator:       "kubernetes",
			clientExperimental: true,
		}},
		{name: "kubernetes annotation is unsupported if not experimental", isUnsupported: true, annotations: []string{"kubernetes"}, details: fakeDetails{orchestrator: "kubernetes"}},
		{name: "kubernetes annotation is unsupported if swarm is activated", isUnsupported: true, annotations: []string{"kubernetes"}, details: fakeDetails{orchestrator: "swarm"}},
		{name: "experimental annotation is unsupported if server is not experimental", isUnsupported: true, annotations: []string{"experimental"}, details: fakeDetails{serverExperimental: false}},
		{name: "experimental annotation is supported if server is experimental", isUnsupported: false, annotations: []string{"experimental"}, details: fakeDetails{serverExperimental: true}},
		{name: "experimentalCLI annotation is unsupported if client is not experimental", isUnsupported: true, annotations: []string{"experimentalCLI"}, details: fakeDetails{clientExperimental: false}},
		{name: "experimentalCLI annotation is supported if client is experimental", isUnsupported: false, annotations: []string{"experimentalCLI"}, details: fakeDetails{clientExperimental: true}},
		{name: "multiple annotations not matching all requirements is unsupported", isUnsupported: true, annotations: []string{"experimental", "kubernetes"}, details: fakeDetails{serverExperimental: true}},
		{name: "multiple annotations matching all requirements is supported", isUnsupported: false, annotations: []string{"experimental", "kubernetes"}, details: fakeDetails{
			serverExperimental: true,
			clientExperimental: true,
			orchestrator:       "kubernetes",
		}},
	}
	for _, test := range tests {
		// Test with flag
		{
			cmd := makeFlaggedCommand(test.annotations...)
			if test.isUnsupported {
				assert.Check(t, IsSupported(cmd, &test.details) != nil, test.name)
			} else {
				assert.Check(t, is.Nil(IsSupported(cmd, &test.details)), test.name)
			}
		}
		// Test with subcommand
		{
			cmd := makeFlaggedCommand(test.annotations...)
			if test.isUnsupported {
				assert.Check(t, IsSupported(cmd, &test.details) != nil, test.name)
			} else {
				assert.Check(t, is.Nil(IsSupported(cmd, &test.details)), test.name)
			}
		}
	}
}

type fakeClient struct {
	client.Client
	version string
}

func (cli *fakeClient) ServerVersion(ctx context.Context) (types.Version, error) {
	return types.Version{}, nil
}
func (cli *fakeClient) ClientVersion() string {
	return cli.version
}

type fakeDetails struct {
	clientVersion      string
	clientExperimental bool
	orchestrator       command.Orchestrator

	serverOsType       string
	serverExperimental bool
}

func (f *fakeDetails) Client() client.APIClient {
	return &fakeClient{version: f.clientVersion}
}
func (f *fakeDetails) ClientInfo() command.ClientInfo {
	return command.ClientInfo{
		HasExperimental: f.clientExperimental,
		Orchestrator:    f.orchestrator,
	}
}
func (f *fakeDetails) ServerInfo() command.ServerInfo {
	return command.ServerInfo{
		OSType:          f.serverOsType,
		HasExperimental: f.serverExperimental,
	}
}

func makeFlaggedCommand(annotations ...string) *cobra.Command {
	cmd := &cobra.Command{}
	flags := cmd.Flags()
	flags.String("myflag", "myvalue", "myusage")
	flags.Set("myflag", "myvalue2")
	for _, annotation := range annotations {
		flags.SetAnnotation("myflag", annotation, nil)
	}
	return cmd
}

func checkFlagVisibility(t *testing.T, cmd *cobra.Command, isHidden bool, reason string) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		assert.Check(t, is.Equal(f.Hidden, isHidden), reason)
	})
}

func makeSubCommand(annotations ...string) *cobra.Command {
	cmd := &cobra.Command{}
	subCommand := &cobra.Command{Annotations: map[string]string{}}
	for _, annotation := range annotations {
		subCommand.Annotations[annotation] = ""
	}
	cmd.AddCommand(subCommand)
	return cmd
}

func checkSubCommandVisibility(t *testing.T, cmd *cobra.Command, isHidden bool, reason string) {
	assert.Check(t, is.Equal(cmd.Commands()[0].Hidden, isHidden), reason)
}
