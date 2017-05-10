package main

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api"
	"github.com/docker/docker/client"
	"github.com/docker/engine-api-proxy/pipeline"
	"github.com/docker/engine-api-proxy/proxy"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type options struct {
	backend  string
	listen   string
	logLevel string
	scope    string
	group    string
}

func newMainCommand() *cobra.Command {
	opts := options{}
	cmd := &cobra.Command{
		Use:           "proxy",
		Short:         "Proxy the engine API",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errors.New("no args allowed")
			}
			return runMain(opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.backend, "backend", "b", "unix:///var/run/docker.sock",
		"Socket or host:port of the engine API to proxy")
	flags.StringVarP(&opts.listen, "listen", "l", "unix:///var/run/docker-proxy.sock",
		"Socket or port to listen on")
	flags.StringVar(&opts.logLevel, "log-level", "INFO", "Log level")
	flags.StringVar(&opts.scope, "scope", "", "Name of project or pipeline scope")
	flags.StringVar(&opts.group, "group", "docker", "Group of the listen socket")
	return cmd
}

func runMain(opts options) error {
	if err := setupLogging(opts.logLevel); err != nil {
		return errors.Wrap(err, "failed to configure logging")
	}

	client, err := NewClient(opts.backend)
	if err != nil {
		return errors.Wrap(err, "failed to create backend client")
	}

	routes, err := setupDemoMiddleware(opts, client)
	if err != nil {
		return err
	}

	apiProxy, err := proxy.NewProxy(proxy.Options{
		Listen:      opts.listen,
		Backend:     opts.backend,
		SocketGroup: opts.group,
		Routes:      routes,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create proxy")
	}
	return apiProxy.Start()
}

// NewClient creates a new Docker API client
// TODO: move this
func NewClient(addr string) (client.APIClient, error) {
	return client.NewClient(addr, api.DefaultVersion, nil, nil)
}

func setupDemoMiddleware(opts options, client client.APIClient) ([]proxy.MiddlewareRoute, error) {
	switch {
	case opts.scope != "":
		lookup := func() pipeline.Scoper {
			return pipeline.NewPipelineScoper(opts.scope)
		}
		return pipeline.MiddlewareRoutes(client, lookup), nil
	default:
		return nil, errors.New("this demo requires --scope")
	}
}

func setupLogging(level string) error {
	logLevel, err := log.ParseLevel(level)
	log.SetLevel(logLevel)
	return err
}

func main() {
	if err := newMainCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(-1)
	}
}
