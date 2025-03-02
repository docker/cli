package task

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/idresolver"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/api/types/swarm"
	"github.com/fvbommel/sortorder"
)

type tasksSortable []swarm.Task

func (t tasksSortable) Len() int {
	return len(t)
}

func (t tasksSortable) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t tasksSortable) Less(i, j int) bool {
	if t[i].Name != t[j].Name {
		return sortorder.NaturalLess(t[i].Name, t[j].Name)
	}
	// Sort tasks for the same service and slot by most recent.
	return t[j].Meta.CreatedAt.Before(t[i].CreatedAt)
}

// Print task information in a format.
// Besides this, command `docker node ps <node>`
// and `docker stack ps` will call this, too.
func Print(ctx context.Context, dockerCli command.Cli, tasks []swarm.Task, resolver *idresolver.IDResolver, trunc, quiet bool, format string) error {
	tasks, err := generateTaskNames(ctx, tasks, resolver)
	if err != nil {
		return err
	}

	// First sort tasks, so that all tasks (including previous ones) of the same
	// service and slot are together. This must be done first, to print "previous"
	// tasks indented
	sort.Stable(tasksSortable(tasks))

	names := map[string]string{}
	nodes := map[string]string{}

	tasksCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewTaskFormat(format, quiet),
		Trunc:  trunc,
	}

	var indent string
	if tasksCtx.Format.IsTable() {
		indent = ` \_ `
	}
	prevName := ""
	for _, task := range tasks {
		if task.Name == prevName {
			// Indent previous tasks of the same slot
			names[task.ID] = indent + task.Name
		} else {
			names[task.ID] = task.Name
		}
		prevName = task.Name

		nodeValue, err := resolver.Resolve(ctx, swarm.Node{}, task.NodeID)
		if err != nil {
			return err
		}
		nodes[task.ID] = nodeValue
	}

	return FormatWrite(tasksCtx, tasks, names, nodes)
}

// generateTaskNames generates names for the given tasks, and returns a copy of
// the slice with the 'Name' field set.
//
// Depending if the "--no-resolve" option is set, names have the following pattern:
//
// - ServiceName.Slot or ServiceID.Slot for tasks that are part of a replicated service
// - ServiceName.NodeName or ServiceID.NodeID for tasks that are part of a global service
//
// Task-names are not unique in cases where "tasks" contains previous/rotated tasks.
func generateTaskNames(ctx context.Context, tasks []swarm.Task, resolver *idresolver.IDResolver) ([]swarm.Task, error) {
	// Use a copy of the tasks list, to not modify the original slice
	// see https://github.com/go101/go101/wiki/How-to-efficiently-clone-a-slice%3F
	t := append(tasks[:0:0], tasks...) //nolint:gocritic // ignore appendAssign: append result not assigned to the same slice

	for i, task := range t {
		serviceName, err := resolver.Resolve(ctx, swarm.Service{}, task.ServiceID)
		if err != nil {
			return nil, err
		}
		if task.Slot != 0 {
			t[i].Name = fmt.Sprintf("%v.%v", serviceName, task.Slot)
		} else {
			t[i].Name = fmt.Sprintf("%v.%v", serviceName, task.NodeID)
		}
	}
	return t, nil
}

// DefaultFormat returns the default format from the config file, or table
// format if nothing is set in the config.
func DefaultFormat(configFile *configfile.ConfigFile, quiet bool) string {
	if len(configFile.TasksFormat) > 0 && !quiet {
		return configFile.TasksFormat
	}
	return formatter.TableFormatKey
}
