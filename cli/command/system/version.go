package system

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/kubernetes"
	"github.com/docker/cli/templates"
	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	kubernetesClient "k8s.io/client-go/kubernetes"
)

var versionTemplate = `{{with .Client -}}
Client:{{if ne .Platform.Name ""}} {{.Platform.Name}}{{end}}
 Version:	{{.Version}}
 API version:	{{.APIVersion}}{{if ne .APIVersion .DefaultAPIVersion}} (downgraded from {{.DefaultAPIVersion}}){{end}}
 Go version:	{{.GoVersion}}
 Git commit:	{{.GitCommit}}
 Built:	{{.BuildTime}}
 OS/Arch:	{{.Os}}/{{.Arch}}
 Experimental:	{{.Experimental}}
 Orchestrator:	{{.Orchestrator}}
{{- end}}

{{- if .ServerOK}}{{with .Server}}

Server:{{if ne .Platform.Name ""}} {{.Platform.Name}}{{end}}
 {{- range $component := .Components}}
 {{$component.Name}}:
  {{- if eq $component.Name "Engine" }}
  Version:	{{.Version}}
  API version:	{{index .Details "ApiVersion"}} (minimum version {{index .Details "MinAPIVersion"}})
  Go version:	{{index .Details "GoVersion"}}
  Git commit:	{{index .Details "GitCommit"}}
  Built:	{{index .Details "BuildTime"}}
  OS/Arch:	{{index .Details "Os"}}/{{index .Details "Arch"}}
  Experimental:	{{index .Details "Experimental"}}
  {{- else }}
  Version:	{{$component.Version}}
  {{- $detailsOrder := getDetailsOrder $component}}
  {{- range $key := $detailsOrder}}
  {{$key}}:	{{index $component.Details $key}}
   {{- end}}
  {{- end}}
 {{- end}}
 {{- end}}{{- end}}`

type versionOptions struct {
	format     string
	kubeConfig string
}

// versionInfo contains version information of both the Client, and Server
type versionInfo struct {
	Client clientVersion
	Server *types.Version
}

type clientVersion struct {
	Platform struct{ Name string } `json:",omitempty"`

	Version           string
	APIVersion        string `json:"ApiVersion"`
	DefaultAPIVersion string `json:"DefaultAPIVersion,omitempty"`
	GitCommit         string
	GoVersion         string
	Os                string
	Arch              string
	BuildTime         string `json:",omitempty"`
	Experimental      bool
	Orchestrator      string `json:",omitempty"`
}

type kubernetesVersion struct {
	Kubernetes string
	StackAPI   string
}

// ServerOK returns true when the client could connect to the docker server
// and parse the information received. It returns false otherwise.
func (v versionInfo) ServerOK() bool {
	return v.Server != nil
}

// NewVersionCommand creates a new cobra.Command for `docker version`
func NewVersionCommand(dockerCli command.Cli) *cobra.Command {
	var opts versionOptions

	cmd := &cobra.Command{
		Use:   "version [OPTIONS]",
		Short: "Show the Docker version information",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", "Format the output using the given Go template")
	flags.StringVarP(&opts.kubeConfig, "kubeconfig", "k", "", "Kubernetes config file")
	flags.SetAnnotation("kubeconfig", "kubernetes", nil)

	return cmd
}

func reformatDate(buildTime string) string {
	t, errTime := time.Parse(time.RFC3339Nano, buildTime)
	if errTime == nil {
		return t.Format(time.ANSIC)
	}
	return buildTime
}

func runVersion(dockerCli command.Cli, opts *versionOptions) error {
	templateFormat := versionTemplate
	tmpl := templates.New("version")
	if opts.format != "" {
		templateFormat = opts.format
	} else {
		tmpl = tmpl.Funcs(template.FuncMap{"getDetailsOrder": getDetailsOrder})
	}

	var err error
	tmpl, err = tmpl.Parse(templateFormat)
	if err != nil {
		return cli.StatusError{StatusCode: 64,
			Status: "Template parsing error: " + err.Error()}
	}

	vd := versionInfo{
		Client: clientVersion{
			Platform:          struct{ Name string }{cli.PlatformName},
			Version:           cli.Version,
			APIVersion:        dockerCli.Client().ClientVersion(),
			DefaultAPIVersion: dockerCli.DefaultVersion(),
			GoVersion:         runtime.Version(),
			GitCommit:         cli.GitCommit,
			BuildTime:         reformatDate(cli.BuildTime),
			Os:                runtime.GOOS,
			Arch:              runtime.GOARCH,
			Experimental:      dockerCli.ClientInfo().HasExperimental,
			Orchestrator:      string(dockerCli.ClientInfo().Orchestrator),
		},
	}

	sv, err := dockerCli.Client().ServerVersion(context.Background())
	if err == nil {
		vd.Server = &sv
		kubeVersion := getKubernetesVersion(dockerCli, opts.kubeConfig)
		foundEngine := false
		foundKubernetes := false
		for _, component := range sv.Components {
			switch component.Name {
			case "Engine":
				foundEngine = true
				buildTime, ok := component.Details["BuildTime"]
				if ok {
					component.Details["BuildTime"] = reformatDate(buildTime)
				}
			case "Kubernetes":
				foundKubernetes = true
				if _, ok := component.Details["StackAPI"]; !ok && kubeVersion != nil {
					component.Details["StackAPI"] = kubeVersion.StackAPI
				}
			}
		}

		if !foundEngine {
			vd.Server.Components = append(vd.Server.Components, types.ComponentVersion{
				Name:    "Engine",
				Version: sv.Version,
				Details: map[string]string{
					"ApiVersion":    sv.APIVersion,
					"MinAPIVersion": sv.MinAPIVersion,
					"GitCommit":     sv.GitCommit,
					"GoVersion":     sv.GoVersion,
					"Os":            sv.Os,
					"Arch":          sv.Arch,
					"BuildTime":     reformatDate(vd.Server.BuildTime),
					"Experimental":  fmt.Sprintf("%t", sv.Experimental),
				},
			})
		}
		if !foundKubernetes && kubeVersion != nil {
			vd.Server.Components = append(vd.Server.Components, types.ComponentVersion{
				Name:    "Kubernetes",
				Version: kubeVersion.Kubernetes,
				Details: map[string]string{
					"StackAPI": kubeVersion.StackAPI,
				},
			})
		}
	}
	t := tabwriter.NewWriter(dockerCli.Out(), 15, 1, 1, ' ', 0)
	if err2 := tmpl.Execute(t, vd); err2 != nil && err == nil {
		err = err2
	}
	t.Write([]byte("\n"))
	t.Flush()
	return err
}

func getDetailsOrder(v types.ComponentVersion) []string {
	out := make([]string, 0, len(v.Details))
	for k := range v.Details {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func getKubernetesVersion(dockerCli command.Cli, kubeConfig string) *kubernetesVersion {
	if !dockerCli.ClientInfo().HasKubernetes() {
		return nil
	}

	version := kubernetesVersion{
		Kubernetes: "Unknown",
		StackAPI:   "Unknown",
	}
	clientConfig := kubernetes.NewKubernetesConfig(kubeConfig)
	config, err := clientConfig.ClientConfig()
	if err != nil {
		logrus.Debugf("failed to get Kubernetes configuration: %s", err)
		return &version
	}
	kubeClient, err := kubernetesClient.NewForConfig(config)
	if err != nil {
		logrus.Debugf("failed to get Kubernetes client: %s", err)
		return &version
	}
	version.StackAPI = getStackVersion(kubeClient)
	version.Kubernetes = getKubernetesServerVersion(kubeClient)
	return &version
}

func getStackVersion(client *kubernetesClient.Clientset) string {
	apiVersion, err := kubernetes.GetStackAPIVersion(client)
	if err != nil {
		logrus.Debugf("failed to get Stack API version: %s", err)
		return "Unknown"
	}
	return string(apiVersion)
}

func getKubernetesServerVersion(client *kubernetesClient.Clientset) string {
	kubeVersion, err := client.DiscoveryClient.ServerVersion()
	if err != nil {
		logrus.Debugf("failed to get Kubernetes server version: %s", err)
		return "Unknown"
	}
	return kubeVersion.String()
}
