package image

import (
	"strconv"
	"strings"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

const (
	defaultHistoryTableFormat  = "table {{.ID}}\t{{.CreatedSince}}\t{{.CreatedBy}}\t{{.Size}}\t{{.Comment}}"
	nonHumanHistoryTableFormat = "table {{.ID}}\t{{.CreatedAt}}\t{{.CreatedBy}}\t{{.Size}}\t{{.Comment}}"

	historyIDHeader = "IMAGE"
	createdByHeader = "CREATED BY"
	commentHeader   = "COMMENT"
)

// newHistoryFormat returns a format for rendering a historyContext.
func newHistoryFormat(source string, quiet bool, human bool) formatter.Format {
	if source == formatter.TableFormatKey {
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

// historyWrite writes the context
func historyWrite(fmtCtx formatter.Context, human bool, history client.ImageHistoryResult) error {
	historyCtx := &historyContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"ID":           historyIDHeader,
				"CreatedSince": formatter.CreatedSinceHeader,
				"CreatedAt":    formatter.CreatedAtHeader,
				"CreatedBy":    createdByHeader,
				"Size":         formatter.SizeHeader,
				"Comment":      commentHeader,
			},
		},
	}
	return fmtCtx.Write(historyCtx, func(format func(subContext formatter.SubContext) error) error {
		for _, h := range history.Items {
			if err := format(&historyContext{
				trunc: fmtCtx.Trunc,
				h:     h,
				human: human,
			}); err != nil {
				return err
			}
		}
		return nil
	})
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
		return formatter.TruncateID(c.h.ID)
	}
	return c.h.ID
}

func (c *historyContext) CreatedAt() string {
	return time.Unix(c.h.Created, 0).Format(time.RFC3339)
}

// epoch is the time before which created-at dates are not displayed with human units.
var epoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

func (c *historyContext) CreatedSince() string {
	if !c.human {
		return c.CreatedAt()
	}
	if c.h.Created <= epoch {
		return "N/A"
	}
	created := units.HumanDuration(time.Now().UTC().Sub(time.Unix(c.h.Created, 0)))
	return created + " ago"
}

func (c *historyContext) CreatedBy() string {
	createdBy := strings.ReplaceAll(c.h.CreatedBy, "\t", " ")
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
