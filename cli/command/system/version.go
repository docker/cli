package system

import (
	"fmt"
	"runtime"
	"sort"
	"text/template"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/templates"
	"github.com/docker/docker/api/types"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
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
  {{$key}}:		{{index $component.Details $key}}
   {{- end}}
  {{- end}}
 {{- end}}
{{- end}}{{end}}`

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
	Experimental      bool
}

// ServerOK returns true when the client could connect to the docker server
// and parse the information received. It returns false otherwise.
func (v versionInfo) ServerOK() bool {
	return v.Server != nil
}

// NewVersionCommand creates a new cobra.Command for `docker version`
func NewVersionCommand(dockerCli *command.DockerCli) *cobra.Command {
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

	return cmd
}

func reformatDate(buildTime string) string {
	t, errTime := time.Parse(time.RFC3339Nano, buildTime)
	if errTime == nil {
		return t.Format(time.ANSIC)
	}
	return buildTime
}

func runVersion(dockerCli *command.DockerCli, opts *versionOptions) error {
	ctx := context.Background()

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
			Version:           cli.Version,
			APIVersion:        dockerCli.Client().ClientVersion(),
			DefaultAPIVersion: dockerCli.DefaultVersion(),
			GoVersion:         runtime.Version(),
			GitCommit:         cli.GitCommit,
			BuildTime:         cli.BuildTime,
			Os:                runtime.GOOS,
			Arch:              runtime.GOARCH,
			Experimental:      dockerCli.ClientInfo().HasExperimental,
		},
	}
	vd.Client.Platform.Name = cli.PlatformName

	// first we need to make BuildTime more human friendly
	vd.Client.BuildTime = reformatDate(vd.Client.BuildTime)

	sv, err := dockerCli.Client().ServerVersion(ctx)
	if err == nil {
		vd.Server = &sv
		foundEngine := false
		for _, component := range sv.Components {
			if component.Name == "Engine" {
				foundEngine = true
				buildTime, ok := component.Details["BuildTime"]
				if ok {
					component.Details["BuildTime"] = reformatDate(buildTime)
				}
				break
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
	}

	if err2 := tmpl.Execute(dockerCli.Out(), vd); err2 != nil && err == nil {
		err = err2
	}
	dockerCli.Out().Write([]byte{'\n'})
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
