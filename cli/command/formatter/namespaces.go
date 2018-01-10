package formatter

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stringid"
)

const (
	defaultNamespacesTableFormat = "table {{.ID}}\t{{.NAME}}\t{{.PID}}\t{{.PATH}}\t{{.CGROUP}}\t{{.IPC}}\t{{.MNT}}\t{{.NET}}\t{{.USER}}\t{{.UTS}}"
	defaultNamespacesRawFormat   = "id: {{.ID}}\nname: {{.NAME}}\nPID: {{.PID}}\nPATH: {{.PATH}}\nCGROUP: {{.CGROUP}}\nIPC: {{.IPC}}\nMNT: {{.MNT}}\nNET: {{.NET}}\nUser: {{.USER}}\nUTS: {{.UTS}}"

	pidHeader    = "PID"
	nsPathHeader = "PATH"
	cgroupHeader = "CGROUP"
	ipcHeader    = "IPC"
	mntHeader    = "MNT"
	netHeader    = "NET"
	userHeader   = "USER"
	utsHeader    = "UTS"
)

var reNsValue = regexp.MustCompile(`\[([0-9]+)\]`)

// NewNamespacesFormat creates a format based on source.
// source can be TableFormatKey or RawFormatKey.
func NewNamespacesFormat(source string) Format {
	switch source {
	case TableFormatKey:
		return defaultNamespacesTableFormat
	case RawFormatKey:
		return defaultNamespacesRawFormat
	}

	return Format(source)
}

// NamespacesWrite gathers namespaces info of containers and writes with info.
// Namespace info is gathered from /proc/<pid>/ns/{pid, path, cgroup, ipc, mnt, net, user, uts} files.
func NamespacesWrite(ctx Context, containers []types.Container, dockerCli command.Cli) error {
	header := namespacesHeaderContext{
		"ID":     containerIDHeader,
		"NAME":   nameHeader,
		"PID":    pidHeader,
		"PATH":   nsPathHeader,
		"CGROUP": cgroupHeader,
		"IPC":    ipcHeader,
		"MNT":    mntHeader,
		"NET":    netHeader,
		"USER":   userHeader,
		"UTS":    utsHeader,
	}

	client := dockerCli.Client()

	render := func(format func(subContext subContext) error) error {
		for _, container := range containers {
			err := format(newNamespacesContext(client, ctx.Trunc, container))
			if err != nil {
				return err
			}
		}

		return nil
	}

	nc := namespacesContext{}
	nc.header = header

	return ctx.Write(&nc, render)
}

type namespacesHeaderContext map[string]string

func (n namespacesHeaderContext) Label(name string) string {
	names := strings.Split(name, ".")
	r := strings.NewReplacer("-", " ", "_", " ")
	h := r.Replace(names[len(names)-1])

	return h
}

type namespacesContext struct {
	HeaderContext
	trunc     bool
	c         types.Container
	apiClient client.APIClient

	procPath string

	cJSON *types.ContainerJSON
}

func newNamespacesContext(client client.APIClient, t bool, c types.Container) *namespacesContext {
	return &namespacesContext{
		apiClient: client,
		trunc:     t,
		procPath:  "/proc", /* default proc path */
		c:         c,
	}
}

func (n *namespacesContext) FullHeader() interface{} {
	return n.header
}

func (n *namespacesContext) MarshalJSON() ([]byte, error) {
	return marshalJSON(n)
}

func (n *namespacesContext) ID() string {
	if n.trunc {
		return stringid.TruncateID(n.c.ID)
	}

	return n.c.ID
}

func (n *namespacesContext) containerJSON() (types.ContainerJSON, error) {
	if n.cJSON != nil {
		return *n.cJSON, nil
	}

	c, err := n.apiClient.ContainerInspect(context.Background(), n.c.ID)
	if err != nil {
		return c, err
	}

	n.cJSON = &c

	return c, nil
}

func (n *namespacesContext) PID() string {
	c, err := n.containerJSON()
	if err != nil {
		return err.Error()
	}

	pid, err := n.stat(c.State.Pid, "pid")
	if err != nil {
		return err.Error()
	}

	return pid
}

func (n *namespacesContext) PATH() string {
	c, err := n.containerJSON()
	if err != nil {
		return err.Error()
	}

	return c.Path
}

func (n *namespacesContext) CGROUP() string {
	c, err := n.containerJSON()
	if err != nil {
		return err.Error()
	}

	cgroup, err := n.stat(c.State.Pid, "cgroup")
	if err != nil {
		return err.Error()
	}

	return cgroup
}

func (n *namespacesContext) IPC() string {
	c, err := n.containerJSON()
	if err != nil {
		return err.Error()
	}

	ipc, err := n.stat(c.State.Pid, "ipc")
	if err != nil {
		return err.Error()
	}

	return ipc
}

func (n *namespacesContext) NAME() string {
	return n.c.Names[0]
}

func (n *namespacesContext) MNT() string {
	c, err := n.containerJSON()
	if err != nil {
		return err.Error()
	}

	mnt, err := n.stat(c.State.Pid, "mnt")
	if err != nil {
		return err.Error()
	}

	return mnt
}

func (n *namespacesContext) NET() string {
	c, err := n.containerJSON()
	if err != nil {
		return err.Error()
	}

	mnt, err := n.stat(c.State.Pid, "net")
	if err != nil {
		return err.Error()
	}

	return mnt
}

func (n *namespacesContext) USER() string {
	c, err := n.containerJSON()
	if err != nil {
		return err.Error()
	}

	mnt, err := n.stat(c.State.Pid, "user")
	if err != nil {
		return err.Error()
	}

	return mnt
}

func (n *namespacesContext) UTS() string {
	c, err := n.containerJSON()
	if err != nil {
		return err.Error()
	}

	mnt, err := n.stat(c.State.Pid, "uts")
	if err != nil {
		return err.Error()
	}

	return mnt
}

func (n *namespacesContext) stat(pid int, ns string) (string, error) {
	o, err := os.Readlink(n.procPath + "/" + strconv.Itoa(pid) + "/ns/" + ns)
	if err != nil {
		return o, err
	}

	var nsValue string

	matches := reNsValue.FindStringSubmatch(o)
	if len(matches) > 1 {
		nsValue = matches[1]
	}

	return nsValue, nil
}
