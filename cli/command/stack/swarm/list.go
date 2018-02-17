package swarm

import (
	"sort"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"vbom.ml/util/sortorder"
)

// RunList is the swarm implementation of docker stack ls
func RunList(dockerCli command.Cli, opts options.List) error {
	client := dockerCli.Client()
	ctx := context.Background()

	stacks, err := getStacks(ctx, client)
	if err != nil {
		return err
	}
	format := opts.Format
	if len(format) == 0 {
		format = formatter.TableFormatKey
	}
	stackCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewStackFormat(format),
	}
	sort.Sort(byName(stacks))
	return formatter.StackWrite(stackCtx, stacks)
}

type byName []*formatter.Stack

func (n byName) Len() int           { return len(n) }
func (n byName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byName) Less(i, j int) bool { return sortorder.NaturalLess(n[i].Name, n[j].Name) }

func getStacks(ctx context.Context, apiclient client.APIClient) ([]*formatter.Stack, error) {
	services, err := apiclient.ServiceList(
		ctx,
		types.ServiceListOptions{Filters: getAllStacksFilter()})
	if err != nil {
		return nil, err
	}
	m := make(map[string]*formatter.Stack)
	for _, service := range services {
		labels := service.Spec.Labels
		name, ok := labels[convert.LabelNamespace]
		if !ok {
			return nil, errors.Errorf("cannot get label %s for service %s",
				convert.LabelNamespace, service.ID)
		}
		ztack, ok := m[name]
		if !ok {
			m[name] = &formatter.Stack{
				Name:     name,
				Services: 1,
			}
		} else {
			ztack.Services++
		}
	}
	var stacks []*formatter.Stack
	for _, stack := range m {
		stacks = append(stacks, stack)
	}
	return stacks, nil
}
