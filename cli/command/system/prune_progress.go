package system

import (
	"fmt"
	"io"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"golang.org/x/net/context"
)

type pruneProgressChan struct {
	errorChan chan error
}

func startPruneProgress(dockerCli command.Cli, progressChans *pruneProgressChan, options pruneOptions, ctx context.Context) error {
	eventChan := make(chan string, 1)
	monitorRemovingEvents(dockerCli, eventChan, progressChans, ctx)
	objCount, err := countObjectsToRemove(dockerCli, options, ctx)
	if err != nil {
		return err
	}
	outputPruningProgress(dockerCli, eventChan, objCount, progressChans)

	return nil
}

func outputPruningProgress(dockerCli command.Cli, events chan string, numEvents int, progressStruct *pruneProgressChan) {
	pipeReader, pipeWriter := io.Pipe()
	progressOut := streamformatter.NewJSONProgressOutput(pipeWriter, false)

	go func() {
		var i int64 = 1
		for s := range events {
			prog := progress.Progress{
				ID:         "Progress",
				Action:     s,
				Current:    i,
				Total:      int64(numEvents),
				HideCounts: true,
			}
			if err := progressOut.WriteProgress(prog); err != nil {
				progressStruct.errorChan <- err
				return
			}
			i++
		}
		pipeWriter.Close()
		close(events)
	}()

	go func() {
		if err := jsonmessage.DisplayJSONMessagesToStream(pipeReader, dockerCli.Out(), nil); err != nil {
			progressStruct.errorChan <- err
			return
		}
	}()
}

func countObjectsToRemove(dockerCli command.Cli, options pruneOptions, ctx context.Context) (int, error) {
	volCount := 0
	if options.pruneVolumes {
		//Volumes listing call
		volFilters := filters.NewArgs()
		vols, err := dockerCli.Client().VolumeList(ctx, volFilters)
		if err != nil {
			return 0, err
		}
		volCount = len(vols.Volumes)
	}

	//Containers listing call.
	//We get all the containers because we need to check which images are attached to them.
	cntFilters := filters.NewArgs()
	cntOpts := types.ContainerListOptions{
		Quiet:   true,
		Size:    false,
		All:     true,
		Latest:  false,
		Since:   "",
		Before:  "",
		Limit:   -1,
		Filters: cntFilters,
	}
	cnts, err := dockerCli.Client().ContainerList(ctx, cntOpts)
	if err != nil {
		return 0, err
	}

	cntCount := 0
	//We check which containers are unused to know which images are unused, also.
	imgUsage := make(map[string]int)
	for _, cnt := range cnts {
		if cnt.State == "running" {
			imgUsage[cnt.ImageID]++
		} else {
			imgUsage[cnt.ImageID] += 0
			cntCount++
		}
	}

	//Images listing call.
	imgFilters := filters.NewArgs()
	if !options.all {
		imgFilters.Add("dangling", "true")
	}
	imgOpts := types.ImageListOptions{
		All:     options.all,
		Filters: imgFilters,
	}
	imgs, err := dockerCli.Client().ImageList(ctx, imgOpts)
	if err != nil {
		return 0, err
	}

	imgCount := 0
	//Here we merge the images from the containers call with the images listing call.
	for _, img := range imgs {
		v, ok := imgUsage[img.ID]
		if !ok || v == 0 {
			imgCount++
		}
	}

	//Network listing call.
	ntwFilters := filters.NewArgs()
	ntwFilters.Add("type", "custom")
	ntwOpts := types.NetworkListOptions{
		Filters: ntwFilters,
	}
	ntws, err := dockerCli.Client().NetworkList(ctx, ntwOpts)
	if err != nil {
		return 0, err
	}

	return cntCount + volCount + imgCount + len(ntws), nil
}

func monitorRemovingEvents(dockerCli command.Cli, out chan<- string, progressChans *pruneProgressChan, ctx context.Context) {
	fArgs := filters.NewArgs(
		filters.Arg("type", "container"),
		filters.Arg("type", "volume"),
		filters.Arg("type", "network"),
		filters.Arg("type", "image"),
		//Destroy for container, network and volume.
		filters.Arg("event", "destroy"),
		//Delete only for image.
		filters.Arg("event", "delete"),
	)
	eOpts := types.EventsOptions{
		Since:   "",
		Until:   "",
		Filters: fArgs,
	}

	go func() {
		events, errors := dockerCli.Client().Events(ctx, eOpts)
		for {
			select {
			case event := <-events:
				out <- fmt.Sprintf("%s %s %s", event.Type, event.Action, event.Actor.ID)
			case err := <-errors:
				progressChans.errorChan <- err
				return
			}
		}
	}()
}
