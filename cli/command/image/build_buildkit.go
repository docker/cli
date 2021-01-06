package image

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/console"
	"github.com/containerd/containerd/platforms"
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
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/filesync"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	"github.com/moby/buildkit/session/sshforward/sshprovider"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/moby/buildkit/util/progress/progresswriter"
	"github.com/pkg/errors"
	fsutiltypes "github.com/tonistiigi/fsutil/types"
	"github.com/tonistiigi/go-rosetta"
	"golang.org/x/sync/errgroup"
)

const uploadRequestRemote = "upload-request"

var errDockerfileConflict = errors.New("ambiguous Dockerfile source: both stdin and flag correspond to Dockerfiles")

//nolint: gocyclo
func runBuildBuildKit(dockerCli command.Cli, options buildOptions) error {
	ctx := appcontext.Context()

	s, err := trySession(dockerCli, options.context, false)
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

	stdoutUsed := false

	switch {
	case options.contextFromStdin():
		if options.dockerfileFromStdin() {
			return errStdinConflict
		}
		rc, isArchive, err := build.DetectArchiveReader(dockerCli.In())
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
			dockerfileReader = dockerCli.In()
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

	outputs, err := parseOutputs(options.outputs)
	if err != nil {
		return errors.Wrapf(err, "failed to parse outputs")
	}

	for _, out := range outputs {
		switch out.Type {
		case "local":
			// dest is handled on client side for local exporter
			outDir, ok := out.Attrs["dest"]
			if !ok {
				return errors.Errorf("dest is required for local output")
			}
			delete(out.Attrs, "dest")
			s.Allow(filesync.NewFSSyncTargetDir(outDir))
		case "tar":
			// dest is handled on client side for tar exporter
			outFile, ok := out.Attrs["dest"]
			if !ok {
				return errors.Errorf("dest is required for tar output")
			}
			var w io.WriteCloser
			if outFile == "-" {
				if _, err := console.ConsoleFromFile(os.Stdout); err == nil {
					return errors.Errorf("refusing to write output to console")
				}
				w = os.Stdout
				stdoutUsed = true
			} else {
				f, err := os.Create(outFile)
				if err != nil {
					return errors.Wrapf(err, "failed to open %s", outFile)
				}
				w = f
			}
			output := func(map[string]string) (io.WriteCloser, error) { return w, nil }
			s.Allow(filesync.NewFSSyncTarget(output))
		}
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

	dockerAuthProvider := authprovider.NewDockerAuthProvider(os.Stderr)
	s.Allow(dockerAuthProvider)
	if len(options.secrets) > 0 {
		sp, err := parseSecretSpecs(options.secrets)
		if err != nil {
			return errors.Wrapf(err, "could not parse secrets: %v", options.secrets)
		}
		s.Allow(sp)
	}
	if len(options.ssh) > 0 {
		sshp, err := parseSSHSpecs(options.ssh)
		if err != nil {
			return errors.Wrapf(err, "could not parse ssh: %v", options.ssh)
		}
		s.Allow(sshp)
	}

	eg, ctx := errgroup.WithContext(ctx)

	dialSession := func(ctx context.Context, proto string, meta map[string][]string) (net.Conn, error) {
		return dockerCli.Client().DialHijack(ctx, "/session", proto, meta)
	}
	eg.Go(func() error {
		return s.Run(context.TODO(), dialSession)
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

	if v := os.Getenv("BUILDKIT_PROGRESS"); v != "" && options.progress == "auto" {
		options.progress = v
	}

	if strings.EqualFold(options.platform, "local") {
		p := platforms.DefaultSpec()
		p.Architecture = rosetta.NativeArch() // current binary architecture might be emulated
		options.platform = platforms.Format(p)
	}

	eg.Go(func() error {
		defer func() { // make sure the Status ends cleanly on build errors
			s.Close()
		}()

		buildOptions := imageBuildOptions(dockerCli, options)
		buildOptions.Version = types.BuilderBuildKit
		buildOptions.Dockerfile = dockerfileName
		// buildOptions.AuthConfigs = authConfigs   // handled by session
		buildOptions.RemoteContext = remote
		buildOptions.SessionID = s.ID()
		buildOptions.BuildID = buildID
		buildOptions.Outputs = outputs
		return doBuild(ctx, eg, dockerCli, stdoutUsed, options, buildOptions, dockerAuthProvider)
	})

	return eg.Wait()
}

//nolint: gocyclo
func doBuild(ctx context.Context, eg *errgroup.Group, dockerCli command.Cli, stdoutUsed bool, options buildOptions, buildOptions types.ImageBuildOptions, at session.Attachable) (finalErr error) {
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
		// not using shared context to not disrupt display but let it finish reporting errors
		eg.Go(func() error {
			return progressui.DisplaySolveStatus(context.TODO(), "", c, out, displayCh)
		})
		if s, ok := at.(interface {
			SetLogger(progresswriter.Logger)
		}); ok {
			s.SetLogger(func(s *client.SolveStatus) {
				displayCh <- s
			})
		}
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
		displayStatus(os.Stderr, t.displayCh)
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
	if options.quiet && !stdoutUsed {
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

func resetUIDAndGID(_ string, s *fsutiltypes.Stat) bool {
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

func parseSecretSpecs(sl []string) (session.Attachable, error) {
	fs := make([]secretsprovider.Source, 0, len(sl))
	for _, v := range sl {
		s, err := parseSecret(v)
		if err != nil {
			return nil, err
		}
		fs = append(fs, *s)
	}
	store, err := secretsprovider.NewStore(fs)
	if err != nil {
		return nil, err
	}
	return secretsprovider.NewSecretProvider(store), nil
}

func parseSecret(value string) (*secretsprovider.Source, error) {
	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse csv secret")
	}

	fs := secretsprovider.Source{}

	var typ string
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		key := strings.ToLower(parts[0])

		if len(parts) != 2 {
			return nil, errors.Errorf("invalid field '%s' must be a key=value pair", field)
		}

		value := parts[1]
		switch key {
		case "type":
			if value != "file" && value != "env" {
				return nil, errors.Errorf("unsupported secret type %q", value)
			}
			typ = value
		case "id":
			fs.ID = value
		case "source", "src":
			fs.FilePath = value
		case "env":
			fs.Env = value
		default:
			return nil, errors.Errorf("unexpected key '%s' in '%s'", key, field)
		}
	}
	if typ == "env" && fs.Env == "" {
		fs.Env = fs.FilePath
		fs.FilePath = ""
	}
	return &fs, nil
}

func parseSSHSpecs(sl []string) (session.Attachable, error) {
	configs := make([]sshprovider.AgentConfig, 0, len(sl))
	for _, v := range sl {
		c := parseSSH(v)
		configs = append(configs, *c)
	}
	return sshprovider.NewSSHAgentProvider(configs)
}

func parseSSH(value string) *sshprovider.AgentConfig {
	parts := strings.SplitN(value, "=", 2)
	cfg := sshprovider.AgentConfig{
		ID: parts[0],
	}
	if len(parts) > 1 {
		cfg.Paths = strings.Split(parts[1], ",")
	}
	return &cfg
}
