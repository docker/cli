package node

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	units "github.com/docker/go-units"
)

const (
	defaultNodeTableFormat                     = "table {{.ID}} {{if .Self}}*{{else}} {{ end }}\t{{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}\t{{.EngineVersion}}"
	nodeInspectPrettyTemplate formatter.Format = `ID:			{{.ID}}
{{- if .Name }}
Name:			{{.Name}}
{{- end }}
{{- if .Labels }}
Labels:
{{- range $k, $v := .Labels }}
 - {{ $k }}{{if $v }}={{ $v }}{{ end }}
{{- end }}{{ end }}
Hostname:              	{{.Hostname}}
Joined at:             	{{.CreatedAt}}
Status:
 State:			{{.StatusState}}
 {{- if .HasStatusMessage}}
 Message:              	{{.StatusMessage}}
 {{- end}}
 Availability:         	{{.SpecAvailability}}
 {{- if .Status.Addr}}
 Address:		{{.StatusAddr}}
 {{- end}}
{{- if .HasManagerStatus}}
Manager Status:
 Address:		{{.ManagerStatusAddr}}
 Raft Status:		{{.ManagerStatusReachability}}
 {{- if .IsManagerStatusLeader}}
 Leader:		Yes
 {{- else}}
 Leader:		No
 {{- end}}
{{- end}}
Platform:
 Operating System:	{{.PlatformOS}}
 Architecture:		{{.PlatformArchitecture}}
Resources:
 CPUs:			{{.ResourceNanoCPUs}}
 Memory:		{{.ResourceMemory}}
{{- if .HasEnginePlugins}}
Plugins:
{{- range $k, $v := .EnginePlugins }}
 {{ $k }}:{{if $v }}		{{ $v }}{{ end }}
{{- end }}
{{- end }}
Engine Version:		{{.EngineVersion}}
{{- if .EngineLabels}}
Engine Labels:
{{- range $k, $v := .EngineLabels }}
 - {{ $k }}{{if $v }}={{ $v }}{{ end }}
{{- end }}{{- end }}
{{- if .HasTLSInfo}}
TLS Info:
 TrustRoot:
{{.TLSInfoTrustRoot}}
 Issuer Subject:	{{.TLSInfoCertIssuerSubject}}
 Issuer Public Key:	{{.TLSInfoCertIssuerPublicKey}}
{{- end}}`
	nodeIDHeader        = "ID"
	selfHeader          = ""
	hostnameHeader      = "HOSTNAME"
	availabilityHeader  = "AVAILABILITY"
	managerStatusHeader = "MANAGER STATUS"
	engineVersionHeader = "ENGINE VERSION"
	tlsStatusHeader     = "TLS STATUS"
)

// NewFormat returns a Format for rendering using a node Context
func NewFormat(source string, quiet bool) formatter.Format {
	switch source {
	case formatter.PrettyFormatKey:
		return nodeInspectPrettyTemplate
	case formatter.TableFormatKey:
		if quiet {
			return formatter.DefaultQuietFormat
		}
		return defaultNodeTableFormat
	case formatter.RawFormatKey:
		if quiet {
			return `node_id: {{.ID}}`
		}
		return `node_id: {{.ID}}\nhostname: {{.Hostname}}\nstatus: {{.Status}}\navailability: {{.Availability}}\nmanager_status: {{.ManagerStatus}}\n`
	}
	return formatter.Format(source)
}

// FormatWrite writes the context
func FormatWrite(ctx formatter.Context, nodes []swarm.Node, info types.Info) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, node := range nodes {
			nodeCtx := &nodeContext{n: node, info: info}
			if err := format(nodeCtx); err != nil {
				return err
			}
		}
		return nil
	}
	nodeCtx := nodeContext{}
	nodeCtx.Header = formatter.SubHeaderContext{
		"ID":            nodeIDHeader,
		"Self":          selfHeader,
		"Hostname":      hostnameHeader,
		"Status":        formatter.StatusHeader,
		"Availability":  availabilityHeader,
		"ManagerStatus": managerStatusHeader,
		"EngineVersion": engineVersionHeader,
		"TLSStatus":     tlsStatusHeader,
	}
	return ctx.Write(&nodeCtx, render)
}

type nodeContext struct {
	formatter.HeaderContext
	n    swarm.Node
	info types.Info
}

func (c *nodeContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *nodeContext) ID() string {
	return c.n.ID
}

func (c *nodeContext) Self() bool {
	return c.n.ID == c.info.Swarm.NodeID
}

func (c *nodeContext) Hostname() string {
	return c.n.Description.Hostname
}

func (c *nodeContext) Status() string {
	return command.PrettyPrint(string(c.n.Status.State))
}

func (c *nodeContext) Availability() string {
	return command.PrettyPrint(string(c.n.Spec.Availability))
}

func (c *nodeContext) ManagerStatus() string {
	reachability := ""
	if c.n.ManagerStatus != nil {
		if c.n.ManagerStatus.Leader {
			reachability = "Leader"
		} else {
			reachability = string(c.n.ManagerStatus.Reachability)
		}
	}
	return command.PrettyPrint(reachability)
}

func (c *nodeContext) TLSStatus() string {
	if c.info.Swarm.Cluster == nil || reflect.DeepEqual(c.info.Swarm.Cluster.TLSInfo, swarm.TLSInfo{}) || reflect.DeepEqual(c.n.Description.TLSInfo, swarm.TLSInfo{}) {
		return "Unknown"
	}
	if reflect.DeepEqual(c.n.Description.TLSInfo, c.info.Swarm.Cluster.TLSInfo) {
		return "Ready"
	}
	return "Needs Rotation"
}

func (c *nodeContext) EngineVersion() string {
	return c.n.Description.Engine.EngineVersion
}

// InspectFormatWrite renders the context for a list of nodes
func InspectFormatWrite(ctx formatter.Context, refs []string, getRef inspect.GetRefFunc) error {
	if ctx.Format != nodeInspectPrettyTemplate {
		return inspect.Inspect(ctx.Output, refs, string(ctx.Format), getRef)
	}
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, ref := range refs {
			nodeI, _, err := getRef(ref)
			if err != nil {
				return err
			}
			node, ok := nodeI.(swarm.Node)
			if !ok {
				return fmt.Errorf("got wrong object to inspect :%v", ok)
			}
			if err := format(&nodeInspectContext{Node: node}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(&nodeInspectContext{}, render)
}

type nodeInspectContext struct {
	swarm.Node
	formatter.SubContext
}

func (ctx *nodeInspectContext) ID() string {
	return ctx.Node.ID
}

func (ctx *nodeInspectContext) Name() string {
	return ctx.Node.Spec.Name
}

func (ctx *nodeInspectContext) Labels() map[string]string {
	return ctx.Node.Spec.Labels
}

func (ctx *nodeInspectContext) Hostname() string {
	return ctx.Node.Description.Hostname
}

func (ctx *nodeInspectContext) CreatedAt() string {
	return command.PrettyPrint(ctx.Node.CreatedAt)
}

func (ctx *nodeInspectContext) StatusState() string {
	return command.PrettyPrint(ctx.Node.Status.State)
}

func (ctx *nodeInspectContext) HasStatusMessage() bool {
	return ctx.Node.Status.Message != ""
}

func (ctx *nodeInspectContext) StatusMessage() string {
	return command.PrettyPrint(ctx.Node.Status.Message)
}

func (ctx *nodeInspectContext) SpecAvailability() string {
	return command.PrettyPrint(ctx.Node.Spec.Availability)
}

func (ctx *nodeInspectContext) HasStatusAddr() bool {
	return ctx.Node.Status.Addr != ""
}

func (ctx *nodeInspectContext) StatusAddr() string {
	return ctx.Node.Status.Addr
}

func (ctx *nodeInspectContext) HasManagerStatus() bool {
	return ctx.Node.ManagerStatus != nil
}

func (ctx *nodeInspectContext) ManagerStatusAddr() string {
	return ctx.Node.ManagerStatus.Addr
}

func (ctx *nodeInspectContext) ManagerStatusReachability() string {
	return command.PrettyPrint(ctx.Node.ManagerStatus.Reachability)
}

func (ctx *nodeInspectContext) IsManagerStatusLeader() bool {
	return ctx.Node.ManagerStatus.Leader
}

func (ctx *nodeInspectContext) PlatformOS() string {
	return ctx.Node.Description.Platform.OS
}

func (ctx *nodeInspectContext) PlatformArchitecture() string {
	return ctx.Node.Description.Platform.Architecture
}

func (ctx *nodeInspectContext) ResourceNanoCPUs() int {
	if ctx.Node.Description.Resources.NanoCPUs == 0 {
		return int(0)
	}
	return int(ctx.Node.Description.Resources.NanoCPUs) / 1e9
}

func (ctx *nodeInspectContext) ResourceMemory() string {
	if ctx.Node.Description.Resources.MemoryBytes == 0 {
		return ""
	}
	return units.BytesSize(float64(ctx.Node.Description.Resources.MemoryBytes))
}

func (ctx *nodeInspectContext) HasEnginePlugins() bool {
	return len(ctx.Node.Description.Engine.Plugins) > 0
}

func (ctx *nodeInspectContext) EnginePlugins() map[string]string {
	pluginMap := map[string][]string{}
	for _, p := range ctx.Node.Description.Engine.Plugins {
		pluginMap[p.Type] = append(pluginMap[p.Type], p.Name)
	}

	pluginNamesByType := map[string]string{}
	for k, v := range pluginMap {
		pluginNamesByType[k] = strings.Join(v, ", ")
	}
	return pluginNamesByType
}

func (ctx *nodeInspectContext) EngineLabels() map[string]string {
	return ctx.Node.Description.Engine.Labels
}

func (ctx *nodeInspectContext) EngineVersion() string {
	return ctx.Node.Description.Engine.EngineVersion
}

func (ctx *nodeInspectContext) HasTLSInfo() bool {
	tlsInfo := ctx.Node.Description.TLSInfo
	return !reflect.DeepEqual(tlsInfo, swarm.TLSInfo{})
}

func (ctx *nodeInspectContext) TLSInfoTrustRoot() string {
	return ctx.Node.Description.TLSInfo.TrustRoot
}

func (ctx *nodeInspectContext) TLSInfoCertIssuerPublicKey() string {
	return base64.StdEncoding.EncodeToString(ctx.Node.Description.TLSInfo.CertIssuerPublicKey)
}

func (ctx *nodeInspectContext) TLSInfoCertIssuerSubject() string {
	return base64.StdEncoding.EncodeToString(ctx.Node.Description.TLSInfo.CertIssuerSubject)
}
