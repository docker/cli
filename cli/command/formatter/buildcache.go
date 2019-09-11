package formatter

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/go-units"
)

const (
	defaultBuildCacheTableFormat = "table {{.ID}}\t{{.Type}}\t{{.Size}}\t{{.CreatedSince}}\t{{.LastUsedSince}}\t{{.UsageCount}}\t{{.Shared}}\t{{.Description}}"

	cacheIDHeader       = "CACHE ID"
	cacheTypeHeader     = "CACHE TYPE"
	parentHeader        = "PARENT"
	lastUsedSinceHeader = "LAST USED"
	usageCountHeader    = "USAGE"
	inUseHeader         = "IN USE"
	sharedHeader        = "SHARED"
)

// NewBuildCacheFormat returns a Format for rendering using a Context
func NewBuildCacheFormat(source string, quiet bool) Format {
	switch source {
	case TableFormatKey:
		if quiet {
			return DefaultQuietFormat
		}
		return Format(defaultBuildCacheTableFormat)
	case RawFormatKey:
		if quiet {
			return `build_cache_id: {{.ID}}`
		}
		format := `build_cache_id: {{.ID}}
parent_id: {{.Parent}}
build_cache_type: {{.CacheType}}
description: {{.Description}}
created_at: {{.CreatedAt}}
created_since: {{.CreatedSince}}
last_used_at: {{.LastUsedAt}}
last_used_since: {{.LastUsedSince}}
usage_count: {{.UsageCount}}
in_use: {{.InUse}}
shared: {{.Shared}}
`
		return Format(format)
	}
	return Format(source)
}

func buildCacheSort(buildCache []*types.BuildCache) {
	sort.Slice(buildCache, func(i, j int) bool {
		lui, luj := buildCache[i].LastUsedAt, buildCache[j].LastUsedAt
		switch {
		case lui == nil && luj == nil:
			return strings.Compare(buildCache[i].ID, buildCache[j].ID) < 0
		case lui == nil:
			return true
		case luj == nil:
			return false
		case lui.Equal(*luj):
			return strings.Compare(buildCache[i].ID, buildCache[j].ID) < 0
		default:
			return lui.Before(*luj)
		}
	})
}

// BuildCacheWrite renders the context for a list of containers
func BuildCacheWrite(ctx Context, buildCaches []*types.BuildCache) error {
	render := func(format func(subContext SubContext) error) error {
		buildCacheSort(buildCaches)
		for _, bc := range buildCaches {
			err := format(&buildCacheContext{trunc: ctx.Trunc, v: bc})
			if err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newBuildCacheContext(), render)
}

type buildCacheContext struct {
	HeaderContext
	trunc bool
	v     *types.BuildCache
}

func newBuildCacheContext() *buildCacheContext {
	buildCacheCtx := buildCacheContext{}
	buildCacheCtx.Header = SubHeaderContext{
		"ID":            cacheIDHeader,
		"Parent":        parentHeader,
		"CacheType":     cacheTypeHeader,
		"Size":          SizeHeader,
		"CreatedSince":  CreatedSinceHeader,
		"LastUsedSince": lastUsedSinceHeader,
		"UsageCount":    usageCountHeader,
		"InUse":         inUseHeader,
		"Shared":        sharedHeader,
		"Description":   DescriptionHeader,
	}
	return &buildCacheCtx
}

func (c *buildCacheContext) MarshalJSON() ([]byte, error) {
	return MarshalJSON(c)
}

func (c *buildCacheContext) ID() string {
	id := c.v.ID
	if c.trunc {
		id = stringid.TruncateID(c.v.ID)
	}
	if c.v.InUse {
		return id + "*"
	}
	return id
}

func (c *buildCacheContext) Parent() string {
	if c.trunc {
		return stringid.TruncateID(c.v.Parent)
	}
	return c.v.Parent
}

func (c *buildCacheContext) CacheType() string {
	return c.v.Type
}

func (c *buildCacheContext) Description() string {
	return c.v.Description
}

func (c *buildCacheContext) Size() string {
	return units.HumanSizeWithPrecision(float64(c.v.Size), 3)
}

func (c *buildCacheContext) CreatedAt() string {
	return c.v.CreatedAt.String()
}

func (c *buildCacheContext) CreatedSince() string {
	return units.HumanDuration(time.Now().UTC().Sub(c.v.CreatedAt)) + " ago"
}

func (c *buildCacheContext) LastUsedAt() string {
	if c.v.LastUsedAt == nil {
		return ""
	}
	return c.v.LastUsedAt.String()
}

func (c *buildCacheContext) LastUsedSince() string {
	if c.v.LastUsedAt == nil {
		return ""
	}
	return units.HumanDuration(time.Now().UTC().Sub(*c.v.LastUsedAt)) + " ago"
}

func (c *buildCacheContext) UsageCount() string {
	return fmt.Sprintf("%d", c.v.UsageCount)
}

func (c *buildCacheContext) InUse() string {
	return fmt.Sprintf("%t", c.v.InUse)
}

func (c *buildCacheContext) Shared() string {
	return fmt.Sprintf("%t", c.v.Shared)
}
