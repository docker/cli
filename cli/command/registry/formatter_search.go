package registry

import (
	"strconv"
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	registrytypes "github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
)

const (
	defaultSearchTableFormat = "table {{.Name}}\t{{.Description}}\t{{.StarCount}}\t{{.IsOfficial}}"

	starsHeader     = "STARS"
	officialHeader  = "OFFICIAL"
	automatedHeader = "AUTOMATED"
)

// newFormat returns a Format for rendering using a searchContext.
func newFormat(source string) formatter.Format {
	switch source {
	case "", formatter.TableFormatKey:
		return defaultSearchTableFormat
	}
	return formatter.Format(source)
}

// formatWrite writes the context.
func formatWrite(fmtCtx formatter.Context, results client.ImageSearchResult) error {
	searchCtx := &searchContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"Name":        formatter.NameHeader,
				"Description": formatter.DescriptionHeader,
				"StarCount":   starsHeader,
				"IsOfficial":  officialHeader,
			},
		},
	}
	return fmtCtx.Write(searchCtx, func(format func(subContext formatter.SubContext) error) error {
		for _, result := range results.Items {
			if err := format(&searchContext{
				trunc: fmtCtx.Trunc,
				s:     result,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

type searchContext struct {
	formatter.HeaderContext
	trunc bool
	json  bool
	s     registrytypes.SearchResult
}

func (c *searchContext) MarshalJSON() ([]byte, error) {
	c.json = true
	return formatter.MarshalJSON(c)
}

func (c *searchContext) Name() string {
	return c.s.Name
}

func (c *searchContext) Description() string {
	desc := strings.ReplaceAll(c.s.Description, "\n", " ")
	desc = strings.ReplaceAll(desc, "\r", " ")
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
