package formatter

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/go-units"
)

const (
	defaultContainerTableFormat = "table {{.ID}}\t{{.Image}}\t{{.Command}}\t{{.RunningFor}}\t{{.Status}}\t{{.Ports}}\t{{.Names}}"

	containerIDHeader = "CONTAINER ID"
	namesHeader       = "NAMES"
	commandHeader     = "COMMAND"
	runningForHeader  = "CREATED"
	statusHeader      = "STATUS"
	portsHeader       = "PORTS"
	mountsHeader      = "MOUNTS"
	localVolumes      = "LOCAL VOLUMES"
	networksHeader    = "NETWORKS"
)

// NewContainerFormat returns a Format for rendering using a Context
func NewContainerFormat(source string, quiet bool, size bool) Format {
	switch source {
	case TableFormatKey:
		if quiet {
			return defaultQuietFormat
		}
		format := defaultContainerTableFormat
		if size {
			format += `\t{{.Size}}`
		}
		return Format(format)
	case RawFormatKey:
		if quiet {
			return `container_id: {{.ID}}`
		}
		format := `container_id: {{.ID}}
image: {{.Image}}
command: {{.Command}}
created_at: {{.CreatedAt}}
status: {{- pad .Status 1 0}}
names: {{.Names}}
labels: {{- pad .Labels 1 0}}
ports: {{- pad .Ports 1 0}}
`
		if size {
			format += `size: {{.Size}}\n`
		}
		return Format(format)
	}
	return Format(source)
}

// ContainerWrite renders the context for a list of containers
func ContainerWrite(ctx Context, containers []types.Container) error {
	render := func(format func(subContext subContext) error) error {
		for _, container := range containers {
			err := format(&containerContext{trunc: ctx.Trunc, c: container})
			if err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(newContainerContext(), render)
}

type containerHeaderContext map[string]string

func (c containerHeaderContext) Label(name string) string {
	n := strings.Split(name, ".")
	r := strings.NewReplacer("-", " ", "_", " ")
	h := r.Replace(n[len(n)-1])

	return h
}

type containerContext struct {
	HeaderContext
	trunc bool
	c     types.Container
}

func newContainerContext() *containerContext {
	containerCtx := containerContext{}
	containerCtx.header = containerHeaderContext{
		"ID":           containerIDHeader,
		"Names":        namesHeader,
		"Image":        imageHeader,
		"Command":      commandHeader,
		"CreatedAt":    createdAtHeader,
		"RunningFor":   runningForHeader,
		"Ports":        portsHeader,
		"Status":       statusHeader,
		"Size":         sizeHeader,
		"Labels":       labelsHeader,
		"Mounts":       mountsHeader,
		"LocalVolumes": localVolumes,
		"Networks":     networksHeader,
	}
	return &containerCtx
}

func (c *containerContext) MarshalJSON() ([]byte, error) {
	return marshalJSON(c)
}

func (c *containerContext) ID() string {
	if c.trunc {
		return stringid.TruncateID(c.c.ID)
	}
	return c.c.ID
}

func (c *containerContext) Names() string {
	names := stripNamePrefix(c.c.Names)
	if c.trunc {
		for _, name := range names {
			if len(strings.Split(name, "/")) == 1 {
				names = []string{name}
				break
			}
		}
	}
	return strings.Join(names, ",")
}

func (c *containerContext) Image() string {
	if c.c.Image == "" {
		return "<no image>"
	}
	if c.trunc {
		if trunc := stringid.TruncateID(c.c.ImageID); trunc == stringid.TruncateID(c.c.Image) {
			return trunc
		}
		// truncate digest if no-trunc option was not selected
		ref, err := reference.ParseNormalizedNamed(c.c.Image)
		if err == nil {
			if nt, ok := ref.(reference.NamedTagged); ok {
				// case for when a tag is provided
				if namedTagged, err := reference.WithTag(reference.TrimNamed(nt), nt.Tag()); err == nil {
					return reference.FamiliarString(namedTagged)
				}
			} else {
				// case for when a tag is not provided
				named := reference.TrimNamed(ref)
				return reference.FamiliarString(named)
			}
		}
	}

	return c.c.Image
}

func (c *containerContext) Command() string {
	command := c.c.Command
	if c.trunc {
		command = Ellipsis(command, 20)
	}
	return strconv.Quote(command)
}

func (c *containerContext) CreatedAt() string {
	return time.Unix(c.c.Created, 0).String()
}

func (c *containerContext) RunningFor() string {
	createdAt := time.Unix(c.c.Created, 0)
	return units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"
}

func (c *containerContext) Ports() string {
	return DisplayablePorts(c.c.Ports)
}

func (c *containerContext) Status() string {
	return c.c.Status
}

func (c *containerContext) Size() string {
	srw := units.HumanSizeWithPrecision(float64(c.c.SizeRw), 3)
	sv := units.HumanSizeWithPrecision(float64(c.c.SizeRootFs), 3)

	sf := srw
	if c.c.SizeRootFs > 0 {
		sf = fmt.Sprintf("%s (virtual %s)", srw, sv)
	}
	return sf
}

func (c *containerContext) Labels() string {
	if c.c.Labels == nil {
		return ""
	}

	var joinLabels []string
	for k, v := range c.c.Labels {
		joinLabels = append(joinLabels, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(joinLabels, ",")
}

func (c *containerContext) Label(name string) string {
	if c.c.Labels == nil {
		return ""
	}
	return c.c.Labels[name]
}

func (c *containerContext) Mounts() string {
	var name string
	var mounts []string
	for _, m := range c.c.Mounts {
		if m.Name == "" {
			name = m.Source
		} else {
			name = m.Name
		}
		if c.trunc {
			name = Ellipsis(name, 15)
		}
		mounts = append(mounts, name)
	}
	return strings.Join(mounts, ",")
}

func (c *containerContext) LocalVolumes() string {
	count := 0
	for _, m := range c.c.Mounts {
		if m.Driver == "local" {
			count++
		}
	}

	return fmt.Sprintf("%d", count)
}

func (c *containerContext) Networks() string {
	if c.c.NetworkSettings == nil {
		return ""
	}

	networks := []string{}
	for k := range c.c.NetworkSettings.Networks {
		networks = append(networks, k)
	}

	return strings.Join(networks, ",")
}

// DisplayablePorts returns formatted string representing open ports of container
// e.g. "0.0.0.0:80->9090/tcp, 9988/tcp"
// it's used by command 'docker ps'
func DisplayablePorts(ports []types.Port) string {
	type portGroup struct {
		first uint16
		last  uint16
	}
	groupMap := make(map[string]*portGroup)
	var result []string
	var hostMappings []string
	var groupMapKeys []string
	sort.Sort(byPortInfo(ports))
	for _, port := range ports {
		current := port.PrivatePort
		portKey := port.Type
		if port.IP != "" {
			if port.PublicPort != current {
				hostMappings = append(hostMappings, fmt.Sprintf("%s:%d->%d/%s", port.IP, port.PublicPort, port.PrivatePort, port.Type))
				continue
			}
			portKey = fmt.Sprintf("%s/%s", port.IP, port.Type)
		}
		group := groupMap[portKey]

		if group == nil {
			groupMap[portKey] = &portGroup{first: current, last: current}
			// record order that groupMap keys are created
			groupMapKeys = append(groupMapKeys, portKey)
			continue
		}
		if current == (group.last + 1) {
			group.last = current
			continue
		}

		result = append(result, formGroup(portKey, group.first, group.last))
		groupMap[portKey] = &portGroup{first: current, last: current}
	}
	for _, portKey := range groupMapKeys {
		g := groupMap[portKey]
		result = append(result, formGroup(portKey, g.first, g.last))
	}
	result = append(result, hostMappings...)
	return strings.Join(result, ", ")
}

func formGroup(key string, start, last uint16) string {
	parts := strings.Split(key, "/")
	groupType := parts[0]
	var ip string
	if len(parts) > 1 {
		ip = parts[0]
		groupType = parts[1]
	}
	group := strconv.Itoa(int(start))
	if start != last {
		group = fmt.Sprintf("%s-%d", group, last)
	}
	if ip != "" {
		group = fmt.Sprintf("%s:%s->%s", ip, group, group)
	}
	return fmt.Sprintf("%s/%s", group, groupType)
}

// byPortInfo is a temporary type used to sort types.Port by its fields
type byPortInfo []types.Port

func (r byPortInfo) Len() int      { return len(r) }
func (r byPortInfo) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r byPortInfo) Less(i, j int) bool {
	if r[i].PrivatePort != r[j].PrivatePort {
		return r[i].PrivatePort < r[j].PrivatePort
	}

	if r[i].IP != r[j].IP {
		return r[i].IP < r[j].IP
	}

	if r[i].PublicPort != r[j].PublicPort {
		return r[i].PublicPort < r[j].PublicPort
	}

	return r[i].Type < r[j].Type
}
