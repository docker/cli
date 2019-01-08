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
	kubecontext "github.com/docker/cli/cli/context/kubernetes"
	"github.com/docker/cli/cli/version"
	"github.com/docker/cli/kubernetes"
	"github.com/docker/cli/templates"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	kubernetesClient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
	flags.StringVar(&opts.kubeConfig, "kubeconfig", "", "Kubernetes config file")
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
	var err error
	tmpl, err := newVersionTemplate(opts.format)
	if err != nil {
		return cli.StatusError{StatusCode: 64, Status: err.Error()}
	}

	orchestrator, err := dockerCli.StackOrchestrator("")
	if err != nil {
		return cli.StatusError{StatusCode: 64, Status: err.Error()}
	}

	vd := versionInfo{
		Client: clientVersion{
			Platform:          struct{ Name string }{version.PlatformName},
			Version:           version.Version,
			APIVersion:        dockerCli.Client().ClientVersion(),
			DefaultAPIVersion: dockerCli.DefaultVersion(),
			GoVersion:         runtime.Version(),
			GitCommit:         version.GitCommit,
			BuildTime:         reformatDate(version.BuildTime),
			Os:                runtime.GOOS,
			Arch:              runtime.GOARCH,
			Experimental:      dockerCli.ClientInfo().HasExperimental,
		},
	}

	sv, err := dockerCli.Client().ServerVersion(context.Background())
	if err == nil {
		vd.Server = &sv
		var kubeVersion *kubernetesVersion
		if orchestrator.HasKubernetes() {
			kubeVersion = getKubernetesVersion(dockerCli, opts.kubeConfig)
		}
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
	if err2 := prettyPrintVersion(dockerCli, vd, tmpl); err2 != nil && err == nil {
		err = err2
	}
	return err
}

func prettyPrintVersion(dockerCli command.Cli, vd versionInfo, tmpl *template.Template) error {
	t := tabwriter.NewWriter(dockerCli.Out(), 20, 1, 1, ' ', 0)
	err := tmpl.Execute(t, vd)
	t.Write([]byte("\n"))
	t.Flush()
	return err
}

func newVersionTemplate(templateFormat string) (*template.Template, error) {
	if templateFormat == "" {
		templateFormat = versionTemplate
	}
	tmpl := templates.New("version").Funcs(template.FuncMap{"getDetailsOrder": getDetailsOrder})
	tmpl, err := tmpl.Parse(templateFormat)

	return tmpl, errors.Wrap(err, "Template parsing error")
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
	version := kubernetesVersion{
		Kubernetes: "Unknown",
		StackAPI:   "Unknown",
	}
	var (
		clientConfig clientcmd.ClientConfig
		err          error
	)
	if dockerCli.CurrentContext() == "" {
		clientConfig = kubernetes.NewKubernetesConfig(kubeConfig)
	} else {
		clientConfig, err = kubecontext.ConfigFromContext(dockerCli.CurrentContext(), dockerCli.ContextStore())
	}
	if err != nil {
		logrus.Debugf("failed to get Kubernetes configuration: %s", err)
		return &version
	}
	config, err := clientConfig.ClientConfig()
	if err != nil {
		logrus.Debugf("failed to get Kubernetes client config: %s", err)
		return &version
	}
	kubeClient, err := kubernetesClient.NewForConfig(config)
	if err != nil {
		logrus.Debugf("failed to get Kubernetes client: %s", err)
		return &version
	}
	version.StackAPI = getStackVersion(kubeClient, dockerCli.ClientInfo().HasExperimental)
	version.Kubernetes = getKubernetesServerVersion(kubeClient)
	return &version
}

func getStackVersion(client *kubernetesClient.Clientset, experimental bool) string {
	apiVersion, err := kubernetes.GetStackAPIVersion(client, experimental)
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
