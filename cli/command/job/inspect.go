// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package job

import (
	"context"
	"encoding/json"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	jobsv0 "github.com/docker/cli/internal/jobsapi/v0"
	"github.com/spf13/cobra"
)

// inspectOptions holds the parsed flags/args of `docker job inspect`.
type inspectOptions struct {
	job string
	run string
}

// newInspectCommand wires the `docker job inspect [--run RUN] JOB` cobra
// command. client is injected explicitly, following the same pattern as
// newListCommand/newLogsCommand/newRunCommand (see list.go/logs.go/run.go).
func newInspectCommand(dockerCli command.Cli, client jobsClient) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] JOB",
		Short: "Display detailed information on a job",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.job = args[0]
			return runInspect(cmd.Context(), dockerCli, client, &opts)
		},
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.run, "run", "", "Inspect a specific run instead of the job")
	return cmd
}

// jobView is a jobsv0.Job as printed by `docker job inspect`: the container
// spec is a JSON document carried as bytes, so it is shown as that document
// rather than as the base64 string a []byte marshals to. The embedded
// structs' ContainerSpec and Spec fields are shadowed by the ones declared
// here.
type jobView struct {
	jobsv0.Job
	Spec *specView
}

type specView struct {
	jobsv0.JobSpec
	ContainerSpec json.RawMessage
}

func newJobView(j *jobsv0.Job) *jobView {
	if j == nil {
		return nil
	}
	v := &jobView{Job: *j}
	if j.Spec != nil {
		v.Spec = &specView{JobSpec: *j.Spec, ContainerSpec: j.Spec.ContainerSpec}
	}
	return v
}

// runInspect prints the jobsv0.Job (or jobsv0.Run, with opts.run set) as
// indented JSON on dockerCli.Out() -- a single object, not wrapped in an
// array, unlike the generic cli/command/inspect package.
func runInspect(ctx context.Context, dockerCli command.Cli, client jobsClient, opts *inspectOptions) error {
	var v any
	if opts.run != "" {
		reply, err := client.InspectRun(ctx, &jobsv0.InspectRunRequest{JobRef: opts.job, RunRef: opts.run})
		if err != nil {
			return wrapJobsError(err)
		}
		v = reply.Run
	} else {
		reply, err := client.Inspect(ctx, &jobsv0.InspectRequest{JobRef: opts.job})
		if err != nil {
			return wrapJobsError(err)
		}
		v = newJobView(reply.Job)
	}

	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = dockerCli.Out().Write(append(out, '\n'))
	return err
}
