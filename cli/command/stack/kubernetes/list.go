package kubernetes

import (
	"sort"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"vbom.ml/util/sortorder"
)

// RunList is the kubernetes implementation of docker stack ls
func RunList(dockerCli *KubeCli, opts options.List) error {
	stacks, err := getStacks(dockerCli)
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

func getStacks(kubeCli *KubeCli) ([]*formatter.Stack, error) {
	stackSvc, err := kubeCli.stacks()
	if err != nil {
		return nil, err
	}

	stacks, err := stackSvc.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var formattedStacks []*formatter.Stack
	for _, stack := range stacks.Items {
		cfg, err := loadStack(stack.Spec.ComposeFile)
		if err != nil {
			return nil, err
		}
		formattedStacks = append(formattedStacks, &formatter.Stack{
			Name:     stack.Name,
			Services: len(getServices(cfg)),
		})
	}
	return formattedStacks, nil
}

func loadStack(composefile string) (*composetypes.Config, error) {
	parsed, err := loader.ParseYAML([]byte(composefile))
	if err != nil {
		return nil, err
	}
	return loader.Load(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{
				Config: parsed,
			},
		},
	})
}
