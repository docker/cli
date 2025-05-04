package manifest

import (
	"fmt"
	"strings"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/manifest/types"
)

const (
	defaultManifestListQuietFormat = "{{.Name}}"
	defaultManifestListTableFormat = "table {{.Repository}}\t{{.Tag}}\t{{.Platforms}}"

	repositoryHeader = "REPOSITORY"
	tagHeader        = "TAG"
	platformsHeader  = "PLATFORMS"
)

// NewFormat returns a Format for rendering using a manifest list Context
func NewFormat(source string, quiet bool) formatter.Format {
	switch source {
	case formatter.TableFormatKey:
		if quiet {
			return defaultManifestListQuietFormat
		}
		return defaultManifestListTableFormat
	case formatter.RawFormatKey:
		if quiet {
			return `name: {{.Name}}`
		}
		return `repo: {{.Repository}}\ntag: {{.Tag}}\n`
	}
	return formatter.Format(source)
}

// FormatWrite writes formatted manifestLists using the Context
func FormatWrite(ctx formatter.Context, manifestLists []reference.Reference, manifests map[string][]types.ImageManifest) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, manifestList := range manifestLists {
			if n, ok := manifestList.(reference.Named); ok {
				if nt, ok := n.(reference.NamedTagged); ok {
					if err := format(&manifestListContext{
						name:           reference.FamiliarString(manifestList),
						repo:           reference.FamiliarName(nt),
						tag:            nt.Tag(),
						imageManifests: manifests[manifestList.String()],
					}); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	return ctx.Write(newManifestListContext(), render)
}

type manifestListContext struct {
	formatter.HeaderContext
	name           string
	repo           string
	tag            string
	imageManifests []types.ImageManifest
}

func newManifestListContext() *manifestListContext {
	manifestListCtx := manifestListContext{}
	manifestListCtx.Header = formatter.SubHeaderContext{
		"Name":       formatter.NameHeader,
		"Repository": repositoryHeader,
		"Tag":        tagHeader,
		"Platforms":  platformsHeader,
	}
	return &manifestListCtx
}

func (c *manifestListContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *manifestListContext) Name() string {
	return c.name
}

func (c *manifestListContext) Repository() string {
	return c.repo
}

func (c *manifestListContext) Tag() string {
	return c.tag
}

func (c *manifestListContext) Platforms() string {
	platforms := []string{}
	for _, manifest := range c.imageManifests {
		os := manifest.Descriptor.Platform.OS
		arch := manifest.Descriptor.Platform.Architecture
		platforms = append(platforms, fmt.Sprintf("%s/%s", os, arch))
	}
	return strings.Join(platforms, ", ")
}
