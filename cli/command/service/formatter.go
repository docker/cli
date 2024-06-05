package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/docker/docker/api/types/container"
	mounttypes "github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/stringid"
	units "github.com/docker/go-units"
	"github.com/fvbommel/sortorder"
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
{{- if .HasCapabilities }}
Capabilities:
{{- if .HasCapabilityAdd }}
 Add: {{ .CapabilityAdd }}
{{- end }}
{{- if .HasCapabilityDrop }}
 Drop: {{ .CapabilityDrop }}
{{- end }}
{{- end }}
{{- if .ContainerSysCtls }}
SysCtls:
{{- range $k, $v := .ContainerSysCtls }}
 {{ $k }}{{if $v }}: {{ $v }}{{ end }}
{{- end }}{{ end }}
{{- if .ContainerUlimits }}
Ulimits:
{{- range $k, $v := .ContainerUlimits }}
 {{ $k }}: {{ $v }}
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
{{- if .HasLogDriver }}
Log Driver:
{{- if .HasLogDriverName }}
 Name:		{{ .LogDriverName }}
{{- end }}
{{- if .LogOpts }}
 LogOpts:
{{- range $k, $v := .LogOpts }}
  {{ $k }}{{if $v }}:       {{ $v }}{{ end }}
{{- end }}{{ end }}
{{ end }}
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
{{- if gt .ResourceLimitPids 0 }}
  PIDs:		{{ .ResourceLimitPids }}
{{- end }}
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
	for _, nw := range service.Spec.TaskTemplate.Networks {
		if resolved, _, err := getNetwork(nw.Target); err == nil {
			if resolvedNetwork, ok := resolved.(network.Summary); ok {
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

func (ctx *serviceInspectContext) HasLogDriver() bool {
	return ctx.Service.Spec.TaskTemplate.LogDriver != nil
}

func (ctx *serviceInspectContext) HasLogDriverName() bool {
	return ctx.Service.Spec.TaskTemplate.LogDriver.Name != ""
}

func (ctx *serviceInspectContext) LogDriverName() string {
	return ctx.Service.Spec.TaskTemplate.LogDriver.Name
}

func (ctx *serviceInspectContext) LogOpts() map[string]string {
	return ctx.Service.Spec.TaskTemplate.LogDriver.Options
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
	var out []string
	for _, pref := range ctx.Service.Spec.TaskTemplate.Placement.Preferences {
		if pref.Spread != nil {
			out = append(out, "spread="+pref.Spread.SpreadDescriptor)
		}
	}
	return out
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

func (ctx *serviceInspectContext) ContainerUlimits() map[string]string {
	ulimits := map[string]string{}

	for _, u := range ctx.Service.Spec.TaskTemplate.ContainerSpec.Ulimits {
		ulimits[u.Name] = fmt.Sprintf("%d:%d", u.Soft, u.Hard)
	}

	return ulimits
}

func (ctx *serviceInspectContext) HasContainerUlimits() bool {
	return len(ctx.Service.Spec.TaskTemplate.ContainerSpec.Ulimits) > 0
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
	return ctx.Service.Spec.TaskTemplate.Resources.Limits.NanoCPUs > 0 || ctx.Service.Spec.TaskTemplate.Resources.Limits.MemoryBytes > 0 || ctx.Service.Spec.TaskTemplate.Resources.Limits.Pids > 0
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

func (ctx *serviceInspectContext) ResourceLimitPids() int64 {
	if ctx.Service.Spec.TaskTemplate.Resources == nil || ctx.Service.Spec.TaskTemplate.Resources.Limits == nil {
		return 0
	}
	return ctx.Service.Spec.TaskTemplate.Resources.Limits.Pids
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

func (ctx *serviceInspectContext) HasCapabilities() bool {
	return len(ctx.Service.Spec.TaskTemplate.ContainerSpec.CapabilityAdd) > 0 || len(ctx.Service.Spec.TaskTemplate.ContainerSpec.CapabilityDrop) > 0
}

func (ctx *serviceInspectContext) HasCapabilityAdd() bool {
	return len(ctx.Service.Spec.TaskTemplate.ContainerSpec.CapabilityAdd) > 0
}

func (ctx *serviceInspectContext) HasCapabilityDrop() bool {
	return len(ctx.Service.Spec.TaskTemplate.ContainerSpec.CapabilityDrop) > 0
}

func (ctx *serviceInspectContext) CapabilityAdd() string {
	return strings.Join(ctx.Service.Spec.TaskTemplate.ContainerSpec.CapabilityAdd, ", ")
}

func (ctx *serviceInspectContext) CapabilityDrop() string {
	return strings.Join(ctx.Service.Spec.TaskTemplate.ContainerSpec.CapabilityDrop, ", ")
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

// ListFormatWrite writes the context
func ListFormatWrite(ctx formatter.Context, services []swarm.Service) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		sort.Slice(services, func(i, j int) bool {
			return sortorder.NaturalLess(services[i].Spec.Name, services[j].Spec.Name)
		})
		for _, service := range services {
			serviceCtx := &serviceContext{service: service}
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
	service swarm.Service
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
	switch {
	case c.service.Spec.Mode.Global != nil:
		return "global"
	case c.service.Spec.Mode.Replicated != nil:
		return "replicated"
	case c.service.Spec.Mode.ReplicatedJob != nil:
		return "replicated job"
	case c.service.Spec.Mode.GlobalJob != nil:
		return "global job"
	default:
		return ""
	}
}

func (c *serviceContext) Replicas() string {
	s := &c.service

	var running, desired, completed uint64
	if s.ServiceStatus != nil {
		running = c.service.ServiceStatus.RunningTasks
		desired = c.service.ServiceStatus.DesiredTasks
		completed = c.service.ServiceStatus.CompletedTasks
	}
	// for jobs, we will not include the max per node, even if it is set. jobs
	// include instead the progress of the job as a whole, in addition to the
	// current running state. the system respects max per node, but if we
	// included it in the list output, the lines for jobs would be entirely too
	// long and make the UI look bad.
	if s.Spec.Mode.ReplicatedJob != nil {
		return fmt.Sprintf(
			"%d/%d (%d/%d completed)",
			running, desired, completed, *s.Spec.Mode.ReplicatedJob.TotalCompletions,
		)
	}
	if s.Spec.Mode.GlobalJob != nil {
		// for global jobs, we need to do a little math. desired tasks are only
		// the tasks that have not yet actually reached the Completed state.
		// Completed tasks have reached the completed state. the TOTAL number
		// of tasks to run is the sum of the tasks desired to still complete,
		// and the tasks actually completed.
		return fmt.Sprintf(
			"%d/%d (%d/%d completed)",
			running, desired, completed, desired+completed,
		)
	}
	if r := c.maxReplicas(); r > 0 {
		return fmt.Sprintf("%d/%d (max %d per node)", running, desired, r)
	}
	return fmt.Sprintf("%d/%d", running, desired)
}

func (c *serviceContext) maxReplicas() uint64 {
	if c.Mode() != "replicated" || c.service.Spec.TaskTemplate.Placement == nil {
		return 0
	}
	return c.service.Spec.TaskTemplate.Placement.MaxReplicas
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
		pub = strconv.FormatUint(uint64(pr.pStart), 10)
	}
	if pr.tEnd > pr.tStart {
		tgt = fmt.Sprintf("%d-%d", pr.tStart, pr.tEnd)
	} else {
		tgt = strconv.FormatUint(uint64(pr.tStart), 10)
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
