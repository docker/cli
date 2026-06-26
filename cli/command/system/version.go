package system

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strconv"
	"text/template"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/formatter/tabwriter"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/cli/version"
	"github.com/docker/cli/templates"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"github.com/tonistiigi/go-rosetta"
)

const defaultVersionTemplate = `{{with .Client -}}
Client:{{if ne .Platform nil}}{{if ne .Platform.Name ""}} {{.Platform.Name}}{{end}}{{end}}
 Version:	{{.Version}}
 API version:	{{.APIVersion}}{{if ne .APIVersion .DefaultAPIVersion}} (downgraded from {{.DefaultAPIVersion}}){{end}}
 Go version:	{{.GoVersion}}
 Git commit:	{{.GitCommit}}
 Built:	{{.BuildTime}}
 OS/Arch:	{{.Os}}/{{.Arch}}
 Context:	{{.Context}}
{{- end}}

{{- if ne .Server nil}}{{with .Server}}

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
	Server *serverVersion
}

type platformInfo struct {
	Name string `json:"Name,omitempty"`
}

type clientVersion struct {
	Platform          *platformInfo `json:"Platform,omitempty"`
	Version           string        `json:"Version,omitempty"`
	APIVersion        string        `json:"ApiVersion,omitempty"`
	DefaultAPIVersion string        `json:"DefaultAPIVersion,omitempty"`
	GitCommit         string        `json:"GitCommit,omitempty"`
	GoVersion         string        `json:"GoVersion,omitempty"`
	Os                string        `json:"Os,omitempty"`
	Arch              string        `json:"Arch,omitempty"`
	BuildTime         string        `json:"BuildTime,omitempty"`
	Context           string        `json:"Context"`
}

// serverVersion contains information about the Docker server host.
// it's the client-side presentation of [client.ServerVersionResult].
type serverVersion struct {
	Platform      client.PlatformInfo       `json:",omitempty"`              // Platform is the platform (product name) the server is running on.
	Version       string                    `json:"Version"`                 // Version is the version of the daemon.
	APIVersion    string                    `json:"ApiVersion"`              // APIVersion is the highest API version supported by the server.
	MinAPIVersion string                    `json:"MinAPIVersion,omitempty"` // MinAPIVersion is the minimum API version the server supports.
	Os            string                    `json:"Os"`                      // Os is the operating system the server runs on.
	Arch          string                    `json:"Arch"`                    // Arch is the hardware architecture the server runs on.
	Components    []system.ComponentVersion `json:"Components,omitempty"`    // Components contains version information for the components making up the server.

	// The following fields are deprecated, they relate to the Engine component and are kept for backwards compatibility

	GitCommit     string `json:"GitCommit,omitempty"`
	GoVersion     string `json:"GoVersion,omitempty"`
	KernelVersion string `json:"KernelVersion,omitempty"`
	Experimental  bool   `json:"Experimental,omitempty"`
	BuildTime     string `json:"BuildTime,omitempty"`
}

// newClientVersion constructs a new clientVersion. If a dockerCLI is
// passed as argument, additional information is included (API version),
// which may invoke an API connection. Pass nil to omit the additional
// information.
func newClientVersion(contextName string, dockerCli command.Cli) clientVersion {
	v := clientVersion{
		Version:           version.Version,
		DefaultAPIVersion: client.MaxAPIVersion,
		GoVersion:         runtime.Version(),
		GitCommit:         version.GitCommit,
		BuildTime:         reformatDate(version.BuildTime),
		Os:                runtime.GOOS,
		Arch:              arch(),
		Context:           contextName,
	}
	if version.PlatformName != "" {
		v.Platform = &platformInfo{Name: version.PlatformName}
	}
	if dockerCli != nil {
		v.APIVersion = dockerCli.CurrentVersion()
	}
	return v
}

func newServerVersion(sv client.ServerVersionResult) *serverVersion {
	out := &serverVersion{
		Platform:      sv.Platform,
		Version:       sv.Version,
		APIVersion:    sv.APIVersion,
		MinAPIVersion: sv.MinAPIVersion,
		Os:            sv.Os,
		Arch:          sv.Arch,
		Experimental:  sv.Experimental, //nolint:staticcheck // ignore deprecated field.
		Components:    make([]system.ComponentVersion, 0, len(sv.Components)),
	}
	foundEngine := false
	for _, component := range sv.Components {
		if component.Name == "Engine" {
			foundEngine = true
			buildTime, ok := component.Details["BuildTime"]
			if ok {
				component.Details["BuildTime"] = reformatDate(buildTime)
			}
			out.GitCommit = component.Details["GitCommit"]
			out.GoVersion = component.Details["GoVersion"]
			out.KernelVersion = component.Details["KernelVersion"]
			out.Experimental = func() bool { b, _ := strconv.ParseBool(component.Details["Experimental"]); return b }()
			out.BuildTime = buildTime
		}
		out.Components = append(out.Components, component)
	}

	if !foundEngine {
		out.Components = append(out.Components, system.ComponentVersion{
			Name:    "Engine",
			Version: sv.Version,
			Details: map[string]string{
				"ApiVersion":    sv.APIVersion,
				"MinAPIVersion": sv.MinAPIVersion,
				"Os":            sv.Os,
				"Arch":          sv.Arch,
			},
		})
	}
	return out
}

// newVersionCommand creates a new cobra.Command for `docker version`
func newVersionCommand(dockerCLI command.Cli) *cobra.Command {
	var opts versionOptions

	cmd := &cobra.Command{
		Use:   "version [OPTIONS]",
		Short: "Show the Docker version information",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(cmd.Context(), dockerCLI, &opts)
		},
		Annotations: map[string]string{
			"category-top": "10",
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
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
	out := runtime.GOARCH
	if rosetta.Enabled() {
		out += " (rosetta)"
	}
	return out
}

func runVersion(ctx context.Context, dockerCLI command.Cli, opts *versionOptions) error {
	var err error
	tmpl, err := newVersionTemplate(opts.format)
	if err != nil {
		return cli.StatusError{StatusCode: 64, Status: err.Error()}
	}

	vd := versionInfo{
		Client: newClientVersion(dockerCLI.CurrentContext(), dockerCLI),
	}
	sv, err := dockerCLI.Client().ServerVersion(ctx, client.ServerVersionOptions{})
	if err == nil {
		vd.Server = newServerVersion(sv)
	}
	if err2 := prettyPrintVersion(dockerCLI.Out(), vd, tmpl); err2 != nil && err == nil {
		err = err2
	}
	return err
}

func prettyPrintVersion(out io.Writer, vd versionInfo, tmpl *template.Template) error {
	t := tabwriter.NewWriter(out, 20, 1, 1, ' ', 0)
	err := tmpl.Execute(t, vd)
	_, _ = t.Write([]byte("\n"))
	_ = t.Flush()
	return err
}

func newVersionTemplate(templateFormat string) (*template.Template, error) {
	switch templateFormat {
	case "":
		templateFormat = defaultVersionTemplate
	case formatter.JSONFormatKey:
		templateFormat = formatter.JSONFormat
	}
	tmpl, err := templates.New("version").Funcs(template.FuncMap{"getDetailsOrder": getDetailsOrder}).Parse(templateFormat)
	if err != nil {
		return nil, fmt.Errorf("template parsing error: %w", err)
	}
	return tmpl, nil
}

func getDetailsOrder(v system.ComponentVersion) []string {
	out := make([]string, 0, len(v.Details))
	for k := range v.Details {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
