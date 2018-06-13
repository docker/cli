package image

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containerd/console"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/pkg/urlutil"
	controlapi "github.com/moby/buildkit/api/services/control"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/filesync"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/pkg/errors"
	"github.com/tonistiigi/fsutil"
	"golang.org/x/sync/errgroup"
)

const uploadRequestRemote = "upload-request"

var errDockerfileConflict = errors.New("ambiguous Dockerfile source: both stdin and flag correspond to Dockerfiles")

//nolint: gocyclo
func runBuildBuildKit(dockerCli command.Cli, options buildOptions) error {
	ctx := appcontext.Context()

	s, err := trySession(dockerCli, options.context)
	if err != nil {
		return err
	}
	if s == nil {
		return errors.Errorf("buildkit not supported by daemon")
	}

	buildID := stringid.GenerateRandomID()

	var (
		remote           string
		body             io.Reader
		dockerfileName   = options.dockerfileName
		dockerfileReader io.ReadCloser
		dockerfileDir    string
		contextDir       string
	)

	switch {
	case options.contextFromStdin():
		if options.dockerfileFromStdin() {
			return errStdinConflict
		}
		rc, isArchive, err := build.DetectArchiveReader(os.Stdin)
		if err != nil {
			return err
		}
		if isArchive {
			body = rc
			remote = uploadRequestRemote
		} else {
			if options.dockerfileName != "" {
				return errDockerfileConflict
			}
			dockerfileReader = rc
			remote = clientSessionRemote
			// TODO: make fssync handle empty contextdir
			contextDir, _ = ioutil.TempDir("", "empty-dir")
			defer os.RemoveAll(contextDir)
		}
	case isLocalDir(options.context):
		contextDir = options.context
		if options.dockerfileFromStdin() {
			dockerfileReader = os.Stdin
		} else if options.dockerfileName != "" {
			dockerfileName = filepath.Base(options.dockerfileName)
			dockerfileDir = filepath.Dir(options.dockerfileName)
		} else {
			dockerfileDir = options.context
		}
		remote = clientSessionRemote
	case urlutil.IsGitURL(options.context):
		remote = options.context
	case urlutil.IsURL(options.context):
		remote = options.context
	default:
		return errors.Errorf("unable to prepare context: path %q not found", options.context)
	}

	if dockerfileReader != nil {
		dockerfileName = build.DefaultDockerfileName
		dockerfileDir, err = build.WriteTempDockerfile(dockerfileReader)
		if err != nil {
			return err
		}
		defer os.RemoveAll(dockerfileDir)
	}

	if dockerfileDir != "" {
		s.Allow(filesync.NewFSSyncProvider([]filesync.SyncedDir{
			{
				Name: "context",
				Dir:  contextDir,
				Map:  resetUIDAndGID,
			},
			{
				Name: "dockerfile",
				Dir:  dockerfileDir,
			},
		}))
	}

	// statusContext, cancelStatus := context.WithCancel(ctx)
	// defer cancelStatus()

	// if span := opentracing.SpanFromContext(ctx); span != nil {
	// 	statusContext = opentracing.ContextWithSpan(statusContext, span)
	// }

	s.Allow(authprovider.NewDockerAuthProvider())

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return s.Run(context.TODO(), dockerCli.Client().DialSession)
	})

	if body != nil {
		eg.Go(func() error {
			buildOptions := types.ImageBuildOptions{
				Version: types.BuilderBuildKit,
				BuildID: uploadRequestRemote + ":" + buildID,
			}

			response, err := dockerCli.Client().ImageBuild(context.Background(), body, buildOptions)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			return nil
		})
	}

	eg.Go(func() error {
		defer func() { // make sure the Status ends cleanly on build errors
			s.Close()
		}()

		buildOptions := imageBuildOptions(dockerCli, options)
		buildOptions.Version = types.BuilderBuildKit
		buildOptions.Dockerfile = dockerfileName
		//buildOptions.AuthConfigs = authConfigs   // handled by session
		buildOptions.RemoteContext = remote
		buildOptions.SessionID = s.ID()
		buildOptions.BuildID = buildID
		return doBuild(ctx, eg, dockerCli, options, buildOptions)
	})

	return eg.Wait()
}

func doBuild(ctx context.Context, eg *errgroup.Group, dockerCli command.Cli, options buildOptions, buildOptions types.ImageBuildOptions) (finalErr error) {
	response, err := dockerCli.Client().ImageBuild(context.Background(), nil, buildOptions)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	done := make(chan struct{})
	defer close(done)
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return dockerCli.Client().BuildCancel(context.TODO(), buildOptions.BuildID)
		case <-done:
		}
		return nil
	})

	t := newTracer()
	ssArr := []*client.SolveStatus{}

	displayStatus := func(displayCh chan *client.SolveStatus) {
		var c console.Console
		out := os.Stderr
		// TODO: Handle interactive output in non-interactive environment.
		consoleOpt := options.console.Value()
		if cons, err := console.ConsoleFromFile(out); err == nil && (consoleOpt == nil || *consoleOpt) {
			c = cons
		}
		// not using shared context to not disrupt display but let is finish reporting errors
		eg.Go(func() error {
			return progressui.DisplaySolveStatus(context.TODO(), c, out, displayCh)
		})
	}

	if options.quiet {
		eg.Go(func() error {
			// TODO: make sure t.displayCh closes
			for ss := range t.displayCh {
				ssArr = append(ssArr, ss)
			}
			<-done
			// TODO: verify that finalErr is indeed set when error occurs
			if finalErr != nil {
				displayCh := make(chan *client.SolveStatus)
				go func() {
					for _, ss := range ssArr {
						displayCh <- ss
					}
					close(displayCh)
				}()
				displayStatus(displayCh)
			}
			return nil
		})
	} else {
		displayStatus(t.displayCh)
	}
	defer close(t.displayCh)
	err = jsonmessage.DisplayJSONMessagesStream(response.Body, os.Stdout, dockerCli.Out().FD(), dockerCli.Out().IsTerminal(), t.write)
	if err != nil {
		if jerr, ok := err.(*jsonmessage.JSONError); ok {
			// If no error code is set, default to 1
			if jerr.Code == 0 {
				jerr.Code = 1
			}
			// if options.quiet {
			// 	fmt.Fprintf(dockerCli.Err(), "%s%s", progBuff, buildBuff)
			// }
			return cli.StatusError{Status: jerr.Message, StatusCode: jerr.Code}
		}
		return err
	}

	return nil
}

func resetUIDAndGID(s *fsutil.Stat) bool {
	s.Uid = uint32(0)
	s.Gid = uint32(0)
	return true
}

type tracer struct {
	displayCh chan *client.SolveStatus
}

func newTracer() *tracer {
	return &tracer{
		displayCh: make(chan *client.SolveStatus),
	}
}

func (t *tracer) write(msg jsonmessage.JSONMessage) {
	var resp controlapi.StatusResponse

	if msg.ID != "moby.buildkit.trace" {
		return
	}

	var dt []byte
	if err := json.Unmarshal(*msg.Aux, &dt); err != nil {
		return
	}
	if err := (&resp).Unmarshal(dt); err != nil {
		return
	}

	s := client.SolveStatus{}
	for _, v := range resp.Vertexes {
		s.Vertexes = append(s.Vertexes, &client.Vertex{
			Digest:    v.Digest,
			Inputs:    v.Inputs,
			Name:      v.Name,
			Started:   v.Started,
			Completed: v.Completed,
			Error:     v.Error,
			Cached:    v.Cached,
		})
	}
	for _, v := range resp.Statuses {
		s.Statuses = append(s.Statuses, &client.VertexStatus{
			ID:        v.ID,
			Vertex:    v.Vertex,
			Name:      v.Name,
			Total:     v.Total,
			Current:   v.Current,
			Timestamp: v.Timestamp,
			Started:   v.Started,
			Completed: v.Completed,
		})
	}
	for _, v := range resp.Logs {
		s.Logs = append(s.Logs, &client.VertexLog{
			Vertex:    v.Vertex,
			Stream:    int(v.Stream),
			Data:      v.Msg,
			Timestamp: v.Timestamp,
		})
	}

	t.displayCh <- &s
}
