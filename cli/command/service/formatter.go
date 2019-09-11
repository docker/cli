package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	mounttypes "github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/stringid"
	units "github.com/docker/go-units"
	"github.com/pkg/errors"
)

const serviceInspectPrettyTemplate formatter.Format = `
ID:		{{.ID}}
Name:		{{.Name}}
{{- if .Labels }}
Labels:
{{- range $k, $v := .Labels }}
 {{ $k }}{{if $v }}={{ $v }}{{ end }}
{{- end }}{{ end }}
Service Mode:
{{- if .IsModeGlobal }}	Global
{{- else if .IsModeReplicated }}	Replicated
{{- if .ModeReplicatedReplicas }}
 Replicas:	{{ .ModeReplicatedReplicas }}
{{- end }}{{ end }}
{{- if .HasUpdateStatus }}
UpdateStatus:
 State:		{{ .UpdateStatusState }}
{{- if .HasUpdateStatusStarted }}
 Started:	{{ .UpdateStatusStarted }}
{{- end }}
{{- if .UpdateIsCompleted }}
 Completed:	{{ .UpdateStatusCompleted }}
{{- end }}
 Message:	{{ .UpdateStatusMessage }}
{{- end }}
Placement:
{{- if .TaskPlacementConstraints }}
 Constraints:	{{ .TaskPlacementConstraints }}
{{- end }}
{{- if .TaskPlacementPreferences }}
 Preferences:   {{ .TaskPlacementPreferences }}
{{- end }}
{{- if .MaxReplicas }}
 Max Replicas Per Node:   {{ .MaxReplicas }}
{{- end }}
{{- if .HasUpdateConfig }}
UpdateConfig:
 Parallelism:	{{ .UpdateParallelism }}
{{- if .HasUpdateDelay}}
 Delay:		{{ .UpdateDelay }}
{{- end }}
 On failure:	{{ .UpdateOnFailure }}
{{- if .HasUpdateMonitor}}
 Monitoring Period: {{ .UpdateMonitor }}
{{- end }}
 Max failure ratio: {{ .UpdateMaxFailureRatio }}
 Update order:      {{ .UpdateOrder }}
{{- end }}
{{- if .HasRollbackConfig }}
RollbackConfig:
 Parallelism:	{{ .RollbackParallelism }}
{{- if .HasRollbackDelay}}
 Delay:		{{ .RollbackDelay }}
{{- end }}
 On failure:	{{ .RollbackOnFailure }}
{{- if .HasRollbackMonitor}}
 Monitoring Period: {{ .RollbackMonitor }}
{{- end }}
 Max failure ratio: {{ .RollbackMaxFailureRatio }}
 Rollback order:    {{ .RollbackOrder }}
{{- end }}
ContainerSpec:
 Image:		{{ .ContainerImage }}
{{- if .ContainerArgs }}
 Args:		{{ range $arg := .ContainerArgs }}{{ $arg }} {{ end }}
{{- end -}}
{{- if .ContainerEnv }}
 Env:		{{ range $env := .ContainerEnv }}{{ $env }} {{ end }}
{{- end -}}
{{- if .ContainerWorkDir }}
 Dir:		{{ .ContainerWorkDir }}
{{- end -}}
{{- if .HasContainerInit }}
 Init:		{{ .ContainerInit }}
{{- end -}}
{{- if .ContainerUser }}
 User: {{ .ContainerUser }}
{{- end }}
{{- if .ContainerSysCtls }}
SysCtls:
{{- range $k, $v := .ContainerSysCtls }}
 {{ $k }}{{if $v }}: {{ $v }}{{ end }}
{{- end }}{{ end }}
{{- if .ContainerMounts }}
Mounts:
{{- end }}
{{- range $mount := .ContainerMounts }}
 Target:	{{ $mount.Target }}
  Source:	{{ $mount.Source }}
  ReadOnly:	{{ $mount.ReadOnly }}
  Type:		{{ $mount.Type }}
{{- end -}}
{{- if .Configs}}
Configs:
{{- range $config := .Configs }}
 Target:	{{$config.File.Name}}
  Source:	{{$config.ConfigName}}
{{- end }}{{ end }}
{{- if .Secrets }}
Secrets:
{{- range $secret := .Secrets }}
 Target:	{{$secret.File.Name}}
  Source:	{{$secret.SecretName}}
{{- end }}{{ end }}
{{- if .HasResources }}
Resources:
{{- if .HasResourceReservations }}
 Reservations:
{{- if gt .ResourceReservationNanoCPUs 0.0 }}
  CPU:		{{ .ResourceReservationNanoCPUs }}
{{- end }}
{{- if .ResourceReservationMemory }}
  Memory:	{{ .ResourceReservationMemory }}
{{- end }}{{ end }}
{{- if .HasResourceLimits }}
 Limits:
{{- if gt .ResourceLimitsNanoCPUs 0.0 }}
  CPU:		{{ .ResourceLimitsNanoCPUs }}
{{- end }}
{{- if .ResourceLimitMemory }}
  Memory:	{{ .ResourceLimitMemory }}
{{- end }}{{ end }}{{ end }}
{{- if .Networks }}
Networks:
{{- range $network := .Networks }} {{ $network }}{{ end }} {{ end }}
Endpoint Mode:	{{ .EndpointMode }}
{{- if .Ports }}
Ports:
{{- range $port := .Ports }}
 PublishedPort = {{ $port.PublishedPort }}
  Protocol = {{ $port.Protocol }}
  TargetPort = {{ $port.TargetPort }}
  PublishMode = {{ $port.PublishMode }}
{{- end }} {{ end -}}
{{- if .Healthcheck }}
 Healthcheck:
  Interval = {{ .Healthcheck.Interval }}
  Retries = {{ .Healthcheck.Retries }}
  StartPeriod =	{{ .Healthcheck.StartPeriod }}
  Timeout =	{{ .Healthcheck.Timeout }}
  {{- if .Healthcheck.Test }}
  Tests:
	{{- range $test := .Healthcheck.Test }}
	 Test = {{ $test }}
  {{- end }} {{ end -}}
{{- end }}
`

// NewFormat returns a Format for rendering using a Context
func NewFormat(source string) formatter.Format {
	switch source {
	case formatter.PrettyFormatKey:
		return serviceInspectPrettyTemplate
	default:
		return formatter.Format(strings.TrimPrefix(source, formatter.RawFormatKey))
	}
}

func resolveNetworks(service swarm.Service, getNetwork inspect.GetRefFunc) map[string]string {
	networkNames := make(map[string]string)
	for _, network := range service.Spec.TaskTemplate.Networks {
		if resolved, _, err := getNetwork(network.Target); err == nil {
			if resolvedNetwork, ok := resolved.(types.NetworkResource); ok {
				networkNames[resolvedNetwork.ID] = resolvedNetwork.Name
			}
		}
	}
	return networkNames
}

// InspectFormatWrite renders the context for a list of services
func InspectFormatWrite(ctx formatter.Context, refs []string, getRef, getNetwork inspect.GetRefFunc) error {
	if ctx.Format != serviceInspectPrettyTemplate {
		return inspect.Inspect(ctx.Output, refs, string(ctx.Format), getRef)
	}
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, ref := range refs {
			serviceI, _, err := getRef(ref)
			if err != nil {
				return err
			}
			service, ok := serviceI.(swarm.Service)
			if !ok {
				return errors.Errorf("got wrong object to inspect")
			}
			if err := format(&serviceInspectContext{Service: service, networkNames: resolveNetworks(service, getNetwork)}); err != nil {
				return err
			}
		}
		return nil
	}
	return ctx.Write(&serviceInspectContext{}, render)
}

type serviceInspectContext struct {
	swarm.Service
	formatter.SubContext

	// networkNames is a map from network IDs (as found in
	// Networks[x].Target) to network names.
	networkNames map[string]string
}

func (ctx *serviceInspectContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(ctx)
}

func (ctx *serviceInspectContext) ID() string {
	return ctx.Service.ID
}

func (ctx *serviceInspectContext) Name() string {
	return ctx.Service.Spec.Name
}

func (ctx *serviceInspectContext) Labels() map[string]string {
	return ctx.Service.Spec.Labels
}

func (ctx *serviceInspectContext) Configs() []*swarm.ConfigReference {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.Configs
}

func (ctx *serviceInspectContext) Secrets() []*swarm.SecretReference {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.Secrets
}

func (ctx *serviceInspectContext) Healthcheck() *container.HealthConfig {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.Healthcheck
}

func (ctx *serviceInspectContext) IsModeGlobal() bool {
	return ctx.Service.Spec.Mode.Global != nil
}

func (ctx *serviceInspectContext) IsModeReplicated() bool {
	return ctx.Service.Spec.Mode.Replicated != nil
}

func (ctx *serviceInspectContext) ModeReplicatedReplicas() *uint64 {
	return ctx.Service.Spec.Mode.Replicated.Replicas
}

func (ctx *serviceInspectContext) HasUpdateStatus() bool {
	return ctx.Service.UpdateStatus != nil && ctx.Service.UpdateStatus.State != ""
}

func (ctx *serviceInspectContext) UpdateStatusState() swarm.UpdateState {
	return ctx.Service.UpdateStatus.State
}

func (ctx *serviceInspectContext) HasUpdateStatusStarted() bool {
	return ctx.Service.UpdateStatus.StartedAt != nil
}

func (ctx *serviceInspectContext) UpdateStatusStarted() string {
	return units.HumanDuration(time.Since(*ctx.Service.UpdateStatus.StartedAt)) + " ago"
}

func (ctx *serviceInspectContext) UpdateIsCompleted() bool {
	return ctx.Service.UpdateStatus.State == swarm.UpdateStateCompleted && ctx.Service.UpdateStatus.CompletedAt != nil
}

func (ctx *serviceInspectContext) UpdateStatusCompleted() string {
	return units.HumanDuration(time.Since(*ctx.Service.UpdateStatus.CompletedAt)) + " ago"
}

func (ctx *serviceInspectContext) UpdateStatusMessage() string {
	return ctx.Service.UpdateStatus.Message
}

func (ctx *serviceInspectContext) TaskPlacementConstraints() []string {
	if ctx.Service.Spec.TaskTemplate.Placement != nil {
		return ctx.Service.Spec.TaskTemplate.Placement.Constraints
	}
	return nil
}

func (ctx *serviceInspectContext) TaskPlacementPreferences() []string {
	if ctx.Service.Spec.TaskTemplate.Placement == nil {
		return nil
	}
	var strings []string
	for _, pref := range ctx.Service.Spec.TaskTemplate.Placement.Preferences {
		if pref.Spread != nil {
			strings = append(strings, "spread="+pref.Spread.SpreadDescriptor)
		}
	}
	return strings
}

func (ctx *serviceInspectContext) MaxReplicas() uint64 {
	if ctx.Service.Spec.TaskTemplate.Placement != nil {
		return ctx.Service.Spec.TaskTemplate.Placement.MaxReplicas
	}
	return 0
}

func (ctx *serviceInspectContext) HasUpdateConfig() bool {
	return ctx.Service.Spec.UpdateConfig != nil
}

func (ctx *serviceInspectContext) UpdateParallelism() uint64 {
	return ctx.Service.Spec.UpdateConfig.Parallelism
}

func (ctx *serviceInspectContext) HasUpdateDelay() bool {
	return ctx.Service.Spec.UpdateConfig.Delay.Nanoseconds() > 0
}

func (ctx *serviceInspectContext) UpdateDelay() time.Duration {
	return ctx.Service.Spec.UpdateConfig.Delay
}

func (ctx *serviceInspectContext) UpdateOnFailure() string {
	return ctx.Service.Spec.UpdateConfig.FailureAction
}

func (ctx *serviceInspectContext) UpdateOrder() string {
	return ctx.Service.Spec.UpdateConfig.Order
}

func (ctx *serviceInspectContext) HasUpdateMonitor() bool {
	return ctx.Service.Spec.UpdateConfig.Monitor.Nanoseconds() > 0
}

func (ctx *serviceInspectContext) UpdateMonitor() time.Duration {
	return ctx.Service.Spec.UpdateConfig.Monitor
}

func (ctx *serviceInspectContext) UpdateMaxFailureRatio() float32 {
	return ctx.Service.Spec.UpdateConfig.MaxFailureRatio
}

func (ctx *serviceInspectContext) HasRollbackConfig() bool {
	return ctx.Service.Spec.RollbackConfig != nil
}

func (ctx *serviceInspectContext) RollbackParallelism() uint64 {
	return ctx.Service.Spec.RollbackConfig.Parallelism
}

func (ctx *serviceInspectContext) HasRollbackDelay() bool {
	return ctx.Service.Spec.RollbackConfig.Delay.Nanoseconds() > 0
}

func (ctx *serviceInspectContext) RollbackDelay() time.Duration {
	return ctx.Service.Spec.RollbackConfig.Delay
}

func (ctx *serviceInspectContext) RollbackOnFailure() string {
	return ctx.Service.Spec.RollbackConfig.FailureAction
}

func (ctx *serviceInspectContext) HasRollbackMonitor() bool {
	return ctx.Service.Spec.RollbackConfig.Monitor.Nanoseconds() > 0
}

func (ctx *serviceInspectContext) RollbackMonitor() time.Duration {
	return ctx.Service.Spec.RollbackConfig.Monitor
}

func (ctx *serviceInspectContext) RollbackMaxFailureRatio() float32 {
	return ctx.Service.Spec.RollbackConfig.MaxFailureRatio
}

func (ctx *serviceInspectContext) RollbackOrder() string {
	return ctx.Service.Spec.RollbackConfig.Order
}

func (ctx *serviceInspectContext) ContainerImage() string {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.Image
}

func (ctx *serviceInspectContext) ContainerArgs() []string {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.Args
}

func (ctx *serviceInspectContext) ContainerEnv() []string {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.Env
}

func (ctx *serviceInspectContext) ContainerWorkDir() string {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.Dir
}

func (ctx *serviceInspectContext) ContainerUser() string {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.User
}

func (ctx *serviceInspectContext) HasContainerInit() bool {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.Init != nil
}

func (ctx *serviceInspectContext) ContainerInit() bool {
	return *ctx.Service.Spec.TaskTemplate.ContainerSpec.Init
}

func (ctx *serviceInspectContext) ContainerMounts() []mounttypes.Mount {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.Mounts
}

func (ctx *serviceInspectContext) ContainerSysCtls() map[string]string {
	return ctx.Service.Spec.TaskTemplate.ContainerSpec.Sysctls
}

func (ctx *serviceInspectContext) HasContainerSysCtls() bool {
	return len(ctx.Service.Spec.TaskTemplate.ContainerSpec.Sysctls) > 0
}

func (ctx *serviceInspectContext) HasResources() bool {
	return ctx.Service.Spec.TaskTemplate.Resources != nil
}

func (ctx *serviceInspectContext) HasResourceReservations() bool {
	if ctx.Service.Spec.TaskTemplate.Resources == nil || ctx.Service.Spec.TaskTemplate.Resources.Reservations == nil {
		return false
	}
	return ctx.Service.Spec.TaskTemplate.Resources.Reservations.NanoCPUs > 0 || ctx.Service.Spec.TaskTemplate.Resources.Reservations.MemoryBytes > 0
}

func (ctx *serviceInspectContext) ResourceReservationNanoCPUs() float64 {
	if ctx.Service.Spec.TaskTemplate.Resources.Reservations.NanoCPUs == 0 {
		return float64(0)
	}
	return float64(ctx.Service.Spec.TaskTemplate.Resources.Reservations.NanoCPUs) / 1e9
}

func (ctx *serviceInspectContext) ResourceReservationMemory() string {
	if ctx.Service.Spec.TaskTemplate.Resources.Reservations.MemoryBytes == 0 {
		return ""
	}
	return units.BytesSize(float64(ctx.Service.Spec.TaskTemplate.Resources.Reservations.MemoryBytes))
}

func (ctx *serviceInspectContext) HasResourceLimits() bool {
	if ctx.Service.Spec.TaskTemplate.Resources == nil || ctx.Service.Spec.TaskTemplate.Resources.Limits == nil {
		return false
	}
	return ctx.Service.Spec.TaskTemplate.Resources.Limits.NanoCPUs > 0 || ctx.Service.Spec.TaskTemplate.Resources.Limits.MemoryBytes > 0
}

func (ctx *serviceInspectContext) ResourceLimitsNanoCPUs() float64 {
	return float64(ctx.Service.Spec.TaskTemplate.Resources.Limits.NanoCPUs) / 1e9
}

func (ctx *serviceInspectContext) ResourceLimitMemory() string {
	if ctx.Service.Spec.TaskTemplate.Resources.Limits.MemoryBytes == 0 {
		return ""
	}
	return units.BytesSize(float64(ctx.Service.Spec.TaskTemplate.Resources.Limits.MemoryBytes))
}

func (ctx *serviceInspectContext) Networks() []string {
	var out []string
	for _, n := range ctx.Service.Spec.TaskTemplate.Networks {
		if name, ok := ctx.networkNames[n.Target]; ok {
			out = append(out, name)
		} else {
			out = append(out, n.Target)
		}
	}
	return out
}

func (ctx *serviceInspectContext) EndpointMode() string {
	if ctx.Service.Spec.EndpointSpec == nil {
		return ""
	}

	return string(ctx.Service.Spec.EndpointSpec.Mode)
}

func (ctx *serviceInspectContext) Ports() []swarm.PortConfig {
	return ctx.Service.Endpoint.Ports
}

const (
	defaultServiceTableFormat = "table {{.ID}}\t{{.Name}}\t{{.Mode}}\t{{.Replicas}}\t{{.Image}}\t{{.Ports}}"

	serviceIDHeader = "ID"
	modeHeader      = "MODE"
	replicasHeader  = "REPLICAS"
)

// NewListFormat returns a Format for rendering using a service Context
func NewListFormat(source string, quiet bool) formatter.Format {
	switch source {
	case formatter.TableFormatKey:
		if quiet {
			return formatter.DefaultQuietFormat
		}
		return defaultServiceTableFormat
	case formatter.RawFormatKey:
		if quiet {
			return `id: {{.ID}}`
		}
		return `id: {{.ID}}\nname: {{.Name}}\nmode: {{.Mode}}\nreplicas: {{.Replicas}}\nimage: {{.Image}}\nports: {{.Ports}}\n`
	}
	return formatter.Format(source)
}

// ListInfo stores the information about mode and replicas to be used by template
type ListInfo struct {
	Mode     string
	Replicas string
}

// ListFormatWrite writes the context
func ListFormatWrite(ctx formatter.Context, services []swarm.Service, info map[string]ListInfo) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, service := range services {
			serviceCtx := &serviceContext{service: service, mode: info[service.ID].Mode, replicas: info[service.ID].Replicas}
			if err := format(serviceCtx); err != nil {
				return err
			}
		}
		return nil
	}
	serviceCtx := serviceContext{}
	serviceCtx.Header = formatter.SubHeaderContext{
		"ID":       serviceIDHeader,
		"Name":     formatter.NameHeader,
		"Mode":     modeHeader,
		"Replicas": replicasHeader,
		"Image":    formatter.ImageHeader,
		"Ports":    formatter.PortsHeader,
	}
	return ctx.Write(&serviceCtx, render)
}

type serviceContext struct {
	formatter.HeaderContext
	service  swarm.Service
	mode     string
	replicas string
}

func (c *serviceContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *serviceContext) ID() string {
	return stringid.TruncateID(c.service.ID)
}

func (c *serviceContext) Name() string {
	return c.service.Spec.Name
}

func (c *serviceContext) Mode() string {
	return c.mode
}

func (c *serviceContext) Replicas() string {
	return c.replicas
}

func (c *serviceContext) Image() string {
	var image string
	if c.service.Spec.TaskTemplate.ContainerSpec != nil {
		image = c.service.Spec.TaskTemplate.ContainerSpec.Image
	}
	if ref, err := reference.ParseNormalizedNamed(image); err == nil {
		// update image string for display, (strips any digest)
		if nt, ok := ref.(reference.NamedTagged); ok {
			if namedTagged, err := reference.WithTag(reference.TrimNamed(nt), nt.Tag()); err == nil {
				image = reference.FamiliarString(namedTagged)
			}
		}
	}

	return image
}

type portRange struct {
	pStart   uint32
	pEnd     uint32
	tStart   uint32
	tEnd     uint32
	protocol swarm.PortConfigProtocol
}

func (pr portRange) String() string {
	var (
		pub string
		tgt string
	)

	if pr.pEnd > pr.pStart {
		pub = fmt.Sprintf("%d-%d", pr.pStart, pr.pEnd)
	} else {
		pub = fmt.Sprintf("%d", pr.pStart)
	}
	if pr.tEnd > pr.tStart {
		tgt = fmt.Sprintf("%d-%d", pr.tStart, pr.tEnd)
	} else {
		tgt = fmt.Sprintf("%d", pr.tStart)
	}
	return fmt.Sprintf("*:%s->%s/%s", pub, tgt, pr.protocol)
}

// Ports formats published ports on the ingress network for output.
//
// Where possible, ranges are grouped to produce a compact output:
// - multiple ports mapped to a single port (80->80, 81->80); is formatted as *:80-81->80
// - multiple consecutive ports on both sides; (80->80, 81->81) are formatted as: *:80-81->80-81
//
// The above should not be grouped together, i.e.:
// - 80->80, 81->81, 82->80 should be presented as : *:80-81->80-81, *:82->80
//
// TODO improve:
// - combine non-consecutive ports mapped to a single port (80->80, 81->80, 84->80, 86->80, 87->80); to be printed as *:80-81,84,86-87->80
// - combine tcp and udp mappings if their port-mapping is exactly the same (*:80-81->80-81/tcp+udp instead of *:80-81->80-81/tcp, *:80-81->80-81/udp)
func (c *serviceContext) Ports() string {
	if c.service.Endpoint.Ports == nil {
		return ""
	}

	pr := portRange{}
	ports := []string{}

	servicePorts := c.service.Endpoint.Ports
	sort.Slice(servicePorts, func(i, j int) bool {
		if servicePorts[i].Protocol == servicePorts[j].Protocol {
			return servicePorts[i].PublishedPort < servicePorts[j].PublishedPort
		}
		return servicePorts[i].Protocol < servicePorts[j].Protocol
	})

	for _, p := range c.service.Endpoint.Ports {
		if p.PublishMode == swarm.PortConfigPublishModeIngress {
			prIsRange := pr.tEnd != pr.tStart
			tOverlaps := p.TargetPort <= pr.tEnd

			// Start a new port-range if:
			// - the protocol is different from the current port-range
			// - published or target port are not consecutive to the current port-range
			// - the current port-range is a _range_, and the target port overlaps with the current range's target-ports
			if p.Protocol != pr.protocol || p.PublishedPort-pr.pEnd > 1 || p.TargetPort-pr.tEnd > 1 || prIsRange && tOverlaps {
				// start a new port-range, and print the previous port-range (if any)
				if pr.pStart > 0 {
					ports = append(ports, pr.String())
				}
				pr = portRange{
					pStart:   p.PublishedPort,
					pEnd:     p.PublishedPort,
					tStart:   p.TargetPort,
					tEnd:     p.TargetPort,
					protocol: p.Protocol,
				}
				continue
			}
			pr.pEnd = p.PublishedPort
			pr.tEnd = p.TargetPort
		}
	}
	if pr.pStart > 0 {
		ports = append(ports, pr.String())
	}
	return strings.Join(ports, ", ")
}
