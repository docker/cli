package system

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

type versionOptions struct {
	format string
}

// NewVersionCommand crea il comando Cobra per 'docker version'
func NewVersionCommand(dockerCLI command.Cli) *cobra.Command {
	var opts versionOptions

	cmd := &cobra.Command{
		Use:   "version [OPTIONS]",
		Short: "Show the Docker version information",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(cmd.Context(), dockerCLI, &opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", "Format the output using the given Go template")

	return cmd
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

	// Chiamata API al Daemon Docker di background
	sv, err := dockerCLI.Client().ServerVersion(ctx, client.ServerVersionOptions{})
	if err == nil {
		vd.Server = newServerVersion(sv)
		
		// -------------------------------------------------------------------------
		// NOSTRA MODIFICA: Log esplicito in caso di Downgrade dell'API Session
		// -------------------------------------------------------------------------
		if vd.Client.APIVersion != sv.APIVersion {
			_, _ = fmt.Fprintf(dockerCLI.Err(), "Note: API version session negotiated to %s (Client supports %s)\n", sv.APIVersion, vd.Client.APIVersion)
		}
		// -------------------------------------------------------------------------
	}

	if err2 := prettyPrintVersion(dockerCLI.Out(), vd, tmpl); err2 != nil && err == nil {
		err = err2
	}
	return err
}

type platformInfo struct {
	Name string `json:",omitempty"`
}

type clientVersion struct {
	Platform          *platformInfo `json:",omitempty"`
	Version           string
	APIVersion        string
	DefaultAPIVersion string
	GitCommit         string
	GoVersion         string
	Os                string
	Arch              string
	BuildTime         string
	Context           string
}

type serverVersion struct {
	types.Version
}

type versionInfo struct {
	Client clientVersion
	Server *serverVersion `json:",omitempty"`
}

func newClientVersion(curContext string, dockerCLI command.Cli) clientVersion {
	return clientVersion{
		Version:           dockerCLI.VersionInfo().Version,
		APIVersion:        dockerCLI.Client().ClientVersion(),
		DefaultAPIVersion: dockerCLI.DefaultVersion(),
		GitCommit:         dockerCLI.VersionInfo().GitCommit,
		GoVersion:         runtime.Version(),
		Os:                runtime.GOOS,
		Arch:              runtime.GOARCH,
		BuildTime:         dockerCLI.VersionInfo().BuildTime,
		Context:           curContext,
	}
}

func newServerVersion(sv types.Version) *serverVersion {
	return &serverVersion{Version: sv}
}

func prettyPrintVersion(w io.Writer, vd versionInfo, tmpl *template.Template) error {
	return tmpl.Execute(w, vd)
}

func newVersionTemplate(tmplStr string) (*template.Template, error) {
	if tmplStr != "" {
		return template.New("version").Parse(tmplStr + "\n")
	}
	return template.New("version").Parse(defaultVersionTemplate)
}

func formatTime(t string) string {
	if t == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return t
	}
	return parsed.Format(time.ANSIC)
}

const defaultVersionTemplate = `Client:
 Version:           {{.Client.Version}}
 API version:       {{.Client.APIVersion}}{{if ne .Client.APIVersion .Client.DefaultAPIVersion}} (downgraded from {{.Client.DefaultAPIVersion}}){{end}}
 Go version:        {{.Client.GoVersion}}
 Git commit:        {{.Client.GitCommit}}
 Built:             {{formatTime .Client.BuildTime}}
 OS/Arch:           {{.Client.Os}}/{{.Client.Arch}}
 Context:           {{.Client.Context}}
{{if .Server}}
Server:
 Engine:
  Version:          {{.Server.Version}}
  API version:      {{.Server.APIVersion}} (minimum version {{.Server.MinAPIVersion}})
  Go version:       {{.Server.GoVersion}}
  Git commit:       {{.Server.GitCommit}}
  Built:            {{formatTime .Server.BuildTime}}
  OS/Arch:          {{.Server.Os}}/{{.Server.Arch}}
  Experimental:     {{.Server.Experimental}}
{{if .Server.Components}}{{range .Server.Components}} {{.Name}}:
  Version:          {{.Version}}
  GitCommit:        {{index .Details "GitCommit"}}
{{end}}{{end}}{{end}}`

func init() {
	var _ func(string) string = formatTime
}
