package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/console"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/cli/opts"
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

	if options.imageIDFile != "" {
		// Avoid leaving a stale file if we eventually fail
		if err := os.Remove(options.imageIDFile); err != nil && !os.IsNotExist(err) {
			return errors.Wrap(err, "removing image ID file")
		}
	}

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

	s.Allow(authprovider.NewDockerAuthProvider())

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return s.Run(context.TODO(), dockerCli.Client().DialSession)
	})

	buildID := stringid.GenerateRandomID()
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

//nolint: gocyclo
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

	if err := opts.ValidateProgressOutput(options.progress); err != nil {
		return err
	}

	displayStatus := func(out *os.File, displayCh chan *client.SolveStatus) {
		var c console.Console
		// TODO: Handle tty output in non-tty environment.
		if cons, err := console.ConsoleFromFile(out); err == nil && (options.progress == "auto" || options.progress == "tty") {
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
				displayStatus(os.Stderr, displayCh)
			}
			return nil
		})
	} else {
		displayStatus(os.Stdout, t.displayCh)
	}
	defer close(t.displayCh)

	buf := bytes.NewBuffer(nil)

	imageID := ""
	writeAux := func(msg jsonmessage.JSONMessage) {
		if msg.ID == "moby.image.id" {
			var result types.BuildResult
			if err := json.Unmarshal(*msg.Aux, &result); err != nil {
				fmt.Fprintf(dockerCli.Err(), "failed to parse aux message: %v", err)
			}
			imageID = result.ID
			return
		}
		t.write(msg)
	}

	err = jsonmessage.DisplayJSONMessagesStream(response.Body, buf, dockerCli.Out().FD(), dockerCli.Out().IsTerminal(), writeAux)
	if err != nil {
		if jerr, ok := err.(*jsonmessage.JSONError); ok {
			// If no error code is set, default to 1
			if jerr.Code == 0 {
				jerr.Code = 1
			}
			return cli.StatusError{Status: jerr.Message, StatusCode: jerr.Code}
		}
	}

	// Everything worked so if -q was provided the output from the daemon
	// should be just the image ID and we'll print that to stdout.
	//
	// TODO: we may want to use Aux messages with ID "moby.image.id" regardless of options.quiet (i.e. don't send HTTP param q=1)
	// instead of assuming that output is image ID if options.quiet.
	if options.quiet {
		imageID = buf.String()
		fmt.Fprint(dockerCli.Out(), imageID)
	}

	if options.imageIDFile != "" {
		if imageID == "" {
			return errors.Errorf("cannot write %s because server did not provide an image ID", options.imageIDFile)
		}
		imageID = strings.TrimSpace(imageID)
		if err := ioutil.WriteFile(options.imageIDFile, []byte(imageID), 0666); err != nil {
			return errors.Wrap(err, "cannot write image ID file")
		}
	}
	return err
}

func resetUIDAndGID(s *fsutil.Stat) bool {
	s.Uid = 0
	s.Gid = 0
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
	// ignoring all messages that are not understood
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
