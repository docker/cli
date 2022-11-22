package system

import (
	"context"
	"runtime"
	"sort"
	"strconv"
	"text/template"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter/tabwriter"
	"github.com/docker/cli/cli/version"
	"github.com/docker/cli/templates"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tonistiigi/go-rosetta"
)

var versionTemplate = `{{with .Client -}}
Client:{{if ne .Platform.Name ""}} {{.Platform.Name}}{{end}}
 Version:	{{.Version}}
 API version:	{{.APIVersion}}{{if ne .APIVersion .DefaultAPIVersion}} (downgraded from {{.DefaultAPIVersion}}){{end}}
 Go version:	{{.GoVersion}}
 Git commit:	{{.GitCommit}}
 Built:	{{.BuildTime}}
 OS/Arch:	{{.Os}}/{{.Arch}}
 Context:	{{.Context}}
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
	format string
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
	Context           string
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
		Annotations: map[string]string{
			"category-top": "10",
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", "Format the output using the given Go template")

	return cmd
}

func reformatDate(buildTime string) string {
	t, errTime := time.Parse(time.RFC3339Nano, buildTime)
	if errTime == nil {
		return t.Format(time.ANSIC)
	}
	return buildTime
}

func arch() string {
	arch := runtime.GOARCH
	if rosetta.Enabled() {
		arch += " (rosetta)"
	}
	return arch
}

func runVersion(dockerCli command.Cli, opts *versionOptions) error {
	var err error
	tmpl, err := newVersionTemplate(opts.format)
	if err != nil {
		return cli.StatusError{StatusCode: 64, Status: err.Error()}
	}

	// TODO print error if kubernetes is used?

	vd := versionInfo{
		Client: clientVersion{
			Platform:          struct{ Name string }{version.PlatformName},
			Version:           version.Version,
			APIVersion:        dockerCli.CurrentVersion(),
			DefaultAPIVersion: dockerCli.DefaultVersion(),
			GoVersion:         runtime.Version(),
			GitCommit:         version.GitCommit,
			BuildTime:         reformatDate(version.BuildTime),
			Os:                runtime.GOOS,
			Arch:              arch(),
			Context:           dockerCli.CurrentContext(),
		},
	}

	sv, err := dockerCli.Client().ServerVersion(context.Background())
	if err == nil {
		vd.Server = &sv
		foundEngine := false
		for _, component := range sv.Components {
			switch component.Name {
			case "Engine":
				foundEngine = true
				buildTime, ok := component.Details["BuildTime"]
				if ok {
					component.Details["BuildTime"] = reformatDate(buildTime)
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
					"Experimental":  strconv.FormatBool(sv.Experimental),
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

	return tmpl, errors.Wrap(err, "template parsing error")
}

func getDetailsOrder(v types.ComponentVersion) []string {
	out := make([]string, 0, len(v.Details))
	for k := range v.Details {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
