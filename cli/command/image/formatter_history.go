package image

import (
	"strconv"
	"strings"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/stringid"
	units "github.com/docker/go-units"
)

const (
	defaultHistoryTableFormat  = "table {{.ID}}\t{{.CreatedSince}}\t{{.CreatedBy}}\t{{.Size}}\t{{.Comment}}"
	nonHumanHistoryTableFormat = "table {{.ID}}\t{{.CreatedAt}}\t{{.CreatedBy}}\t{{.Size}}\t{{.Comment}}"

	historyIDHeader = "IMAGE"
	createdByHeader = "CREATED BY"
	commentHeader   = "COMMENT"
)

// NewHistoryFormat returns a format for rendering an HistoryContext
func NewHistoryFormat(source string, quiet bool, human bool) formatter.Format {
	switch source {
	case formatter.TableFormatKey:
		switch {
		case quiet:
			return formatter.DefaultQuietFormat
		case !human:
			return nonHumanHistoryTableFormat
		default:
			return defaultHistoryTableFormat
		}
	}

	return formatter.Format(source)
}

// HistoryWrite writes the context
func HistoryWrite(ctx formatter.Context, human bool, histories []image.HistoryResponseItem) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, history := range histories {
			historyCtx := &historyContext{trunc: ctx.Trunc, h: history, human: human}
			if err := format(historyCtx); err != nil {
				return err
			}
		}
		return nil
	}
	historyCtx := &historyContext{}
	historyCtx.Header = formatter.SubHeaderContext{
		"ID":           historyIDHeader,
		"CreatedSince": formatter.CreatedSinceHeader,
		"CreatedAt":    formatter.CreatedAtHeader,
		"CreatedBy":    createdByHeader,
		"Size":         formatter.SizeHeader,
		"Comment":      commentHeader,
	}
	return ctx.Write(historyCtx, render)
}

type historyContext struct {
	formatter.HeaderContext
	trunc bool
	human bool
	h     image.HistoryResponseItem
}

func (c *historyContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *historyContext) ID() string {
	if c.trunc {
		return stringid.TruncateID(c.h.ID)
	}
	return c.h.ID
}

func (c *historyContext) CreatedAt() string {
	return time.Unix(c.h.Created, 0).Format(time.RFC3339)
}

func (c *historyContext) CreatedSince() string {
	if !c.human {
		return c.CreatedAt()
	}
	created := units.HumanDuration(time.Now().UTC().Sub(time.Unix(c.h.Created, 0)))
	return created + " ago"
}

func (c *historyContext) CreatedBy() string {
	createdBy := strings.Replace(c.h.CreatedBy, "\t", " ", -1)
	if c.trunc {
		return formatter.Ellipsis(createdBy, 45)
	}
	return createdBy
}

func (c *historyContext) Size() string {
	if c.human {
		return units.HumanSizeWithPrecision(float64(c.h.Size), 3)
	}
	return strconv.FormatInt(c.h.Size, 10)
}

func (c *historyContext) Comment() string {
	return c.h.Comment
}
