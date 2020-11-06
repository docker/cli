package registry

import (
	"strconv"
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	registry "github.com/docker/docker/api/types/registry"
)

const (
	defaultSearchTableFormat = "table {{.Name}}\t{{.Description}}\t{{.StarCount}}\t{{.IsOfficial}}\t{{.IsAutomated}}"

	starsHeader     = "STARS"
	officialHeader  = "OFFICIAL"
	automatedHeader = "AUTOMATED"
)

// NewSearchFormat returns a Format for rendering using a network Context
func NewSearchFormat(source string) formatter.Format {
	switch source {
	case "":
		return defaultSearchTableFormat
	case formatter.TableFormatKey:
		return defaultSearchTableFormat
	}
	return formatter.Format(source)
}

// SearchWrite writes the context
func SearchWrite(ctx formatter.Context, results []registry.SearchResult) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, result := range results {
			searchCtx := &searchContext{trunc: ctx.Trunc, s: result}
			if err := format(searchCtx); err != nil {
				return err
			}
		}
		return nil
	}
	searchCtx := searchContext{}
	searchCtx.Header = formatter.SubHeaderContext{
		"Name":        formatter.NameHeader,
		"Description": formatter.DescriptionHeader,
		"StarCount":   starsHeader,
		"IsOfficial":  officialHeader,
		"IsAutomated": automatedHeader,
	}
	return ctx.Write(&searchCtx, render)
}

type searchContext struct {
	formatter.HeaderContext
	trunc bool
	json  bool
	s     registry.SearchResult
}

func (c *searchContext) MarshalJSON() ([]byte, error) {
	c.json = true
	return formatter.MarshalJSON(c)
}

func (c *searchContext) Name() string {
	return c.s.Name
}

func (c *searchContext) Description() string {
	desc := strings.Replace(c.s.Description, "\n", " ", -1)
	desc = strings.Replace(desc, "\r", " ", -1)
	if c.trunc {
		desc = formatter.Ellipsis(desc, 45)
	}
	return desc
}

func (c *searchContext) StarCount() string {
	return strconv.Itoa(c.s.StarCount)
}

func (c *searchContext) formatBool(value bool) string {
	switch {
	case value && c.json:
		return "true"
	case value:
		return "[OK]"
	case c.json:
		return "false"
	default:
		return ""
	}
}

func (c *searchContext) IsOfficial() string {
	return c.formatBool(c.s.IsOfficial)
}

func (c *searchContext) IsAutomated() string {
	return c.formatBool(c.s.IsAutomated)
}
