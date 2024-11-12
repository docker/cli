// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/google/shlex"
	"github.com/moby/swarmkit/v2/api"
	"github.com/moby/swarmkit/v2/api/defaults"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type int64Value interface {
	Value() int64
}

// Uint64Opt represents a uint64.
type Uint64Opt struct {
	value *uint64
}

// Set a new value on the option
func (i *Uint64Opt) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 64)
	i.value = &v
	return err
}

// Type returns the type of this option, which will be displayed in `--help` output
func (i *Uint64Opt) Type() string {
	return "uint"
}

// String returns a string repr of this option
func (i *Uint64Opt) String() string {
	if i.value != nil {
		return strconv.FormatUint(*i.value, 10)
	}
	return ""
}

// Value returns the uint64
func (i *Uint64Opt) Value() *uint64 {
	return i.value
}

type floatValue float32

func (f *floatValue) Set(s string) error {
	v, err := strconv.ParseFloat(s, 32)
	*f = floatValue(v)
	return err
}

func (f *floatValue) Type() string {
	return "float"
}

func (f *floatValue) String() string {
	return strconv.FormatFloat(float64(*f), 'g', -1, 32)
}

func (f *floatValue) Value() float32 {
	return float32(*f)
}

// placementPrefOpts holds a list of placement preferences.
type placementPrefOpts struct {
	prefs   []swarm.PlacementPreference
	strings []string
}

func (o *placementPrefOpts) String() string {
	if len(o.strings) == 0 {
		return ""
	}
	return fmt.Sprintf("%v", o.strings)
}

// Set validates the input value and adds it to the internal slices.
// Note: in the future strategies other than "spread", may be supported,
// as well as additional comma-separated options.
func (o *placementPrefOpts) Set(value string) error {
	strategy, arg, ok := strings.Cut(value, "=")
	if !ok || strategy == "" {
		return errors.New(`placement preference must be of the format "<strategy>=<arg>"`)
	}
	if strategy != "spread" {
		return errors.Errorf("unsupported placement preference %s (only spread is supported)", strategy)
	}

	o.prefs = append(o.prefs, swarm.PlacementPreference{
		Spread: &swarm.SpreadOver{
			SpreadDescriptor: arg,
		},
	})
	o.strings = append(o.strings, value)
	return nil
}

// Type returns a string name for this Option type
func (o *placementPrefOpts) Type() string {
	return "pref"
}

// ShlexOpt is a flag Value which parses a string as a list of shell words
type ShlexOpt []string

// Set the value
func (s *ShlexOpt) Set(value string) error {
	valueSlice, err := shlex.Split(value)
	if err != nil {
		return err
	}
	*s = valueSlice
	return nil
}

// Type returns the type of the value
func (s *ShlexOpt) Type() string {
	return "command"
}

func (s *ShlexOpt) String() string {
	if len(*s) == 0 {
		return ""
	}
	return fmt.Sprint(*s)
}

// Value returns the value as a string slice
func (s *ShlexOpt) Value() []string {
	return []string(*s)
}

type updateOptions struct {
	parallelism     uint64
	delay           time.Duration
	monitor         time.Duration
	onFailure       string
	maxFailureRatio floatValue
	order           string
}

func updateConfigFromDefaults(defaultUpdateConfig *api.UpdateConfig) *swarm.UpdateConfig {
	defaultFailureAction := strings.ToLower(api.UpdateConfig_FailureAction_name[int32(defaultUpdateConfig.FailureAction)])
	defaultMonitor, _ := gogotypes.DurationFromProto(defaultUpdateConfig.Monitor)
	return &swarm.UpdateConfig{
		Parallelism:     defaultUpdateConfig.Parallelism,
		Delay:           defaultUpdateConfig.Delay,
		Monitor:         defaultMonitor,
		FailureAction:   defaultFailureAction,
		MaxFailureRatio: defaultUpdateConfig.MaxFailureRatio,
		Order:           defaultOrder(defaultUpdateConfig.Order),
	}
}

func (o updateOptions) updateConfig(flags *pflag.FlagSet) *swarm.UpdateConfig {
	if !anyChanged(flags, flagUpdateParallelism, flagUpdateDelay, flagUpdateMonitor, flagUpdateFailureAction, flagUpdateMaxFailureRatio, flagUpdateOrder) {
		return nil
	}

	updateConfig := updateConfigFromDefaults(defaults.Service.Update)

	if flags.Changed(flagUpdateParallelism) {
		updateConfig.Parallelism = o.parallelism
	}
	if flags.Changed(flagUpdateDelay) {
		updateConfig.Delay = o.delay
	}
	if flags.Changed(flagUpdateMonitor) {
		updateConfig.Monitor = o.monitor
	}
	if flags.Changed(flagUpdateFailureAction) {
		updateConfig.FailureAction = o.onFailure
	}
	if flags.Changed(flagUpdateMaxFailureRatio) {
		updateConfig.MaxFailureRatio = o.maxFailureRatio.Value()
	}
	if flags.Changed(flagUpdateOrder) {
		updateConfig.Order = o.order
	}

	return updateConfig
}

func (o updateOptions) rollbackConfig(flags *pflag.FlagSet) *swarm.UpdateConfig {
	if !anyChanged(flags, flagRollbackParallelism, flagRollbackDelay, flagRollbackMonitor, flagRollbackFailureAction, flagRollbackMaxFailureRatio, flagRollbackOrder) {
		return nil
	}

	updateConfig := updateConfigFromDefaults(defaults.Service.Rollback)

	if flags.Changed(flagRollbackParallelism) {
		updateConfig.Parallelism = o.parallelism
	}
	if flags.Changed(flagRollbackDelay) {
		updateConfig.Delay = o.delay
	}
	if flags.Changed(flagRollbackMonitor) {
		updateConfig.Monitor = o.monitor
	}
	if flags.Changed(flagRollbackFailureAction) {
		updateConfig.FailureAction = o.onFailure
	}
	if flags.Changed(flagRollbackMaxFailureRatio) {
		updateConfig.MaxFailureRatio = o.maxFailureRatio.Value()
	}
	if flags.Changed(flagRollbackOrder) {
		updateConfig.Order = o.order
	}

	return updateConfig
}

type resourceOptions struct {
	limitCPU            opts.NanoCPUs
	limitMemBytes       opts.MemBytes
	limitPids           int64
	resCPU              opts.NanoCPUs
	resMemBytes         opts.MemBytes
	resGenericResources []string
}

func (r *resourceOptions) ToResourceRequirements() (*swarm.ResourceRequirements, error) {
	generic, err := ParseGenericResources(r.resGenericResources)
	if err != nil {
		return nil, err
	}

	return &swarm.ResourceRequirements{
		Limits: &swarm.Limit{
			NanoCPUs:    r.limitCPU.Value(),
			MemoryBytes: r.limitMemBytes.Value(),
			Pids:        r.limitPids,
		},
		Reservations: &swarm.Resources{
			NanoCPUs:         r.resCPU.Value(),
			MemoryBytes:      r.resMemBytes.Value(),
			GenericResources: generic,
		},
	}, nil
}

type restartPolicyOptions struct {
	condition   string
	delay       opts.DurationOpt
	maxAttempts Uint64Opt
	window      opts.DurationOpt
}

func defaultRestartPolicy() *swarm.RestartPolicy {
	defaultMaxAttempts := defaults.Service.Task.Restart.MaxAttempts
	rp := &swarm.RestartPolicy{
		MaxAttempts: &defaultMaxAttempts,
	}

	if defaults.Service.Task.Restart.Delay != nil {
		defaultRestartDelay, _ := gogotypes.DurationFromProto(defaults.Service.Task.Restart.Delay)
		rp.Delay = &defaultRestartDelay
	}
	if defaults.Service.Task.Restart.Window != nil {
		defaultRestartWindow, _ := gogotypes.DurationFromProto(defaults.Service.Task.Restart.Window)
		rp.Window = &defaultRestartWindow
	}
	rp.Condition = defaultRestartCondition()

	return rp
}

func defaultRestartCondition() swarm.RestartPolicyCondition {
	switch defaults.Service.Task.Restart.Condition {
	case api.RestartOnNone:
		return "none"
	case api.RestartOnFailure:
		return "on-failure"
	case api.RestartOnAny:
		return "any"
	default:
		return ""
	}
}

func defaultOrder(order api.UpdateConfig_UpdateOrder) string {
	switch order {
	case api.UpdateConfig_STOP_FIRST:
		return "stop-first"
	case api.UpdateConfig_START_FIRST:
		return "start-first"
	default:
		return ""
	}
}

func (r *restartPolicyOptions) ToRestartPolicy(flags *pflag.FlagSet) *swarm.RestartPolicy {
	if !anyChanged(flags, flagRestartDelay, flagRestartMaxAttempts, flagRestartWindow, flagRestartCondition) {
		return nil
	}

	restartPolicy := defaultRestartPolicy()

	if flags.Changed(flagRestartDelay) {
		restartPolicy.Delay = r.delay.Value()
	}
	if flags.Changed(flagRestartCondition) {
		restartPolicy.Condition = swarm.RestartPolicyCondition(r.condition)
	}
	if flags.Changed(flagRestartMaxAttempts) {
		restartPolicy.MaxAttempts = r.maxAttempts.Value()
	}
	if flags.Changed(flagRestartWindow) {
		restartPolicy.Window = r.window.Value()
	}

	return restartPolicy
}

type credentialSpecOpt struct {
	value  *swarm.CredentialSpec
	source string
}

func (c *credentialSpecOpt) Set(value string) error {
	c.source = value
	c.value = &swarm.CredentialSpec{}
	switch {
	case strings.HasPrefix(value, "config://"):
		// NOTE(dperny): we allow the user to specify the value of
		// CredentialSpec Config using the Name of the config, but the API
		// requires the ID of the config. For simplicity, we will parse
		// whatever value is provided into the "Config" field, but before
		// making API calls, we may need to swap the Config Name for the ID.
		// Therefore, this isn't the definitive location for the value of
		// Config that is passed to the API.
		c.value.Config = strings.TrimPrefix(value, "config://")
	case strings.HasPrefix(value, "file://"):
		c.value.File = strings.TrimPrefix(value, "file://")
	case strings.HasPrefix(value, "registry://"):
		c.value.Registry = strings.TrimPrefix(value, "registry://")
	case value == "":
		// if the value of the flag is an empty string, that means there is no
		// CredentialSpec needed. This is useful for removing a CredentialSpec
		// during a service update.
	default:
		return errors.New(`invalid credential spec: value must be prefixed with "config://", "file://", or "registry://"`)
	}

	return nil
}

func (c *credentialSpecOpt) Type() string {
	return "credential-spec"
}

func (c *credentialSpecOpt) String() string {
	return c.source
}

func (c *credentialSpecOpt) Value() *swarm.CredentialSpec {
	return c.value
}

func resolveNetworkID(ctx context.Context, apiClient client.NetworkAPIClient, networkIDOrName string) (string, error) {
	nw, err := apiClient.NetworkInspect(ctx, networkIDOrName, network.InspectOptions{Scope: "swarm"})
	return nw.ID, err
}

func convertNetworks(networks opts.NetworkOpt) []swarm.NetworkAttachmentConfig {
	nws := networks.Value()
	netAttach := make([]swarm.NetworkAttachmentConfig, 0, len(nws))
	for _, net := range nws {
		netAttach = append(netAttach, swarm.NetworkAttachmentConfig{
			Target:     net.Target,
			Aliases:    net.Aliases,
			DriverOpts: net.DriverOpts,
		})
	}
	return netAttach
}

type endpointOptions struct {
	mode         string
	publishPorts opts.PortOpt
}

func (e *endpointOptions) ToEndpointSpec() *swarm.EndpointSpec {
	return &swarm.EndpointSpec{
		Mode:  swarm.ResolutionMode(strings.ToLower(e.mode)),
		Ports: e.publishPorts.Value(),
	}
}

type logDriverOptions struct {
	name string
	opts opts.ListOpts
}

func newLogDriverOptions() logDriverOptions {
	return logDriverOptions{opts: opts.NewListOpts(opts.ValidateEnv)}
}

func (ldo *logDriverOptions) toLogDriver() *swarm.Driver {
	if ldo.name == "" {
		return nil
	}

	// set the log driver only if specified.
	return &swarm.Driver{
		Name:    ldo.name,
		Options: opts.ConvertKVStringsToMap(ldo.opts.GetAll()),
	}
}

type healthCheckOptions struct {
	cmd           string
	interval      opts.PositiveDurationOpt
	timeout       opts.PositiveDurationOpt
	retries       int
	startPeriod   opts.PositiveDurationOpt
	startInterval opts.PositiveDurationOpt
	noHealthcheck bool
}

func (o *healthCheckOptions) toHealthConfig() (*container.HealthConfig, error) {
	var healthConfig *container.HealthConfig
	haveHealthSettings := o.cmd != "" ||
		o.interval.Value() != nil ||
		o.timeout.Value() != nil ||
		o.startPeriod.Value() != nil ||
		o.startInterval.Value() != nil ||
		o.retries != 0
	if o.noHealthcheck {
		if haveHealthSettings {
			return nil, errors.Errorf("--%s conflicts with --health-* options", flagNoHealthcheck)
		}
		healthConfig = &container.HealthConfig{Test: []string{"NONE"}}
	} else if haveHealthSettings {
		var test []string
		if o.cmd != "" {
			test = []string{"CMD-SHELL", o.cmd}
		}
		var interval, timeout, startPeriod, startInterval time.Duration
		if ptr := o.interval.Value(); ptr != nil {
			interval = *ptr
		}
		if ptr := o.timeout.Value(); ptr != nil {
			timeout = *ptr
		}
		if ptr := o.startPeriod.Value(); ptr != nil {
			startPeriod = *ptr
		}
		if ptr := o.startInterval.Value(); ptr != nil {
			startInterval = *ptr
		}
		healthConfig = &container.HealthConfig{
			Test:          test,
			Interval:      interval,
			Timeout:       timeout,
			Retries:       o.retries,
			StartPeriod:   startPeriod,
			StartInterval: startInterval,
		}
	}
	return healthConfig, nil
}

// convertExtraHostsToSwarmHosts converts an array of extra hosts in cli
//
//	<host>:<ip>
//
// into a swarmkit host format:
//
//	IP_address canonical_hostname [aliases...]
//
// This assumes input value (<host>:<ip>) has already been validated
func convertExtraHostsToSwarmHosts(extraHosts []string) []string {
	hosts := make([]string, 0, len(extraHosts))
	for _, extraHost := range extraHosts {
		host, ip, ok := strings.Cut(extraHost, ":")
		if ok {
			hosts = append(hosts, ip+" "+host)
		}
	}
	return hosts
}

type serviceOptions struct {
	detach bool
	quiet  bool

	name            string
	labels          opts.ListOpts
	containerLabels opts.ListOpts
	image           string
	entrypoint      ShlexOpt
	args            []string
	hostname        string
	env             opts.ListOpts
	envFile         opts.ListOpts
	workdir         string
	user            string
	groups          opts.ListOpts
	credentialSpec  credentialSpecOpt
	init            bool
	stopSignal      string
	tty             bool
	readOnly        bool
	mounts          opts.MountOpt
	dns             opts.ListOpts
	dnsSearch       opts.ListOpts
	dnsOption       opts.ListOpts
	hosts           opts.ListOpts
	sysctls         opts.ListOpts
	capAdd          opts.ListOpts
	capDrop         opts.ListOpts
	ulimits         opts.UlimitOpt
	oomScoreAdj     int64

	resources resourceOptions
	stopGrace opts.DurationOpt

	replicas      Uint64Opt
	mode          string
	maxConcurrent Uint64Opt

	restartPolicy  restartPolicyOptions
	constraints    opts.ListOpts
	placementPrefs placementPrefOpts
	maxReplicas    uint64
	update         updateOptions
	rollback       updateOptions
	networks       opts.NetworkOpt
	endpoint       endpointOptions

	registryAuth   bool
	noResolveImage bool

	logDriver logDriverOptions

	healthcheck healthCheckOptions
	secrets     opts.SecretOpt
	configs     opts.ConfigOpt

	isolation string
}

func newServiceOptions() *serviceOptions {
	return &serviceOptions{
		labels:          opts.NewListOpts(opts.ValidateLabel),
		constraints:     opts.NewListOpts(nil),
		containerLabels: opts.NewListOpts(opts.ValidateLabel),
		env:             opts.NewListOpts(opts.ValidateEnv),
		envFile:         opts.NewListOpts(nil),
		groups:          opts.NewListOpts(nil),
		logDriver:       newLogDriverOptions(),
		dns:             opts.NewListOpts(opts.ValidateIPAddress),
		dnsOption:       opts.NewListOpts(nil),
		dnsSearch:       opts.NewListOpts(opts.ValidateDNSSearch),
		hosts:           opts.NewListOpts(opts.ValidateExtraHost),
		sysctls:         opts.NewListOpts(nil),
		capAdd:          opts.NewListOpts(nil),
		capDrop:         opts.NewListOpts(nil),
		ulimits:         *opts.NewUlimitOpt(nil),
	}
}

func (options *serviceOptions) ToServiceMode() (swarm.ServiceMode, error) {
	serviceMode := swarm.ServiceMode{}
	switch options.mode {
	case "global":
		if options.replicas.Value() != nil {
			return serviceMode, errors.Errorf("replicas can only be used with replicated or replicated-job mode")
		}

		if options.maxReplicas > 0 {
			return serviceMode, errors.New("replicas-max-per-node can only be used with replicated or replicated-job mode")
		}
		if options.maxConcurrent.Value() != nil {
			return serviceMode, errors.New("max-concurrent can only be used with replicated-job mode")
		}

		serviceMode.Global = &swarm.GlobalService{}
	case "replicated":
		if options.maxConcurrent.Value() != nil {
			return serviceMode, errors.New("max-concurrent can only be used with replicated-job mode")
		}

		serviceMode.Replicated = &swarm.ReplicatedService{
			Replicas: options.replicas.Value(),
		}
	case "replicated-job":
		concurrent := options.maxConcurrent.Value()
		if concurrent == nil {
			concurrent = options.replicas.Value()
		}
		serviceMode.ReplicatedJob = &swarm.ReplicatedJob{
			MaxConcurrent:    concurrent,
			TotalCompletions: options.replicas.Value(),
		}
	case "global-job":
		if options.maxReplicas > 0 {
			return serviceMode, errors.New("replicas-max-per-node can only be used with replicated or replicated-job mode")
		}
		if options.maxConcurrent.Value() != nil {
			return serviceMode, errors.New("max-concurrent can only be used with replicated-job mode")
		}
		if options.replicas.Value() != nil {
			return serviceMode, errors.Errorf("replicas can only be used with replicated or replicated-job mode")
		}
		serviceMode.GlobalJob = &swarm.GlobalJob{}
	default:
		return serviceMode, errors.Errorf("Unknown mode: %s, only replicated and global supported", options.mode)
	}
	return serviceMode, nil
}

func (options *serviceOptions) ToStopGracePeriod(flags *pflag.FlagSet) *time.Duration {
	if flags.Changed(flagStopGracePeriod) {
		return options.stopGrace.Value()
	}
	return nil
}

// makeEnv gets the environment variables from the command line options and
// returns a slice of strings to use in the service spec when doing ToService
func (options *serviceOptions) makeEnv() ([]string, error) {
	envVariables, err := opts.ReadKVEnvStrings(options.envFile.GetAll(), options.env.GetAll())
	if err != nil {
		return nil, err
	}
	currentEnv := make([]string, 0, len(envVariables))
	for _, env := range envVariables { // need to process each var, in order
		k, _, _ := strings.Cut(env, "=")
		for i, current := range currentEnv { // remove duplicates
			if current == env {
				continue // no update required, may hide this behind flag to preserve order of envVariables
			}
			if strings.HasPrefix(current, k+"=") {
				currentEnv = append(currentEnv[:i], currentEnv[i+1:]...)
			}
		}
		currentEnv = append(currentEnv, env)
	}

	return currentEnv, nil
}

// ToService takes the set of flags passed to the command and converts them
// into a service spec.
//
// Takes an API client as the second argument in order to resolve network names
// from the flags into network IDs.
//
// Returns an error if any flags are invalid or contradictory.
func (options *serviceOptions) ToService(ctx context.Context, apiClient client.NetworkAPIClient, flags *pflag.FlagSet) (swarm.ServiceSpec, error) {
	var service swarm.ServiceSpec

	currentEnv, err := options.makeEnv()
	if err != nil {
		return service, err
	}

	healthConfig, err := options.healthcheck.toHealthConfig()
	if err != nil {
		return service, err
	}

	serviceMode, err := options.ToServiceMode()
	if err != nil {
		return service, err
	}

	updateConfig := options.update.updateConfig(flags)
	rollbackConfig := options.rollback.rollbackConfig(flags)

	// update and rollback configuration is not supported for jobs. If these
	// flags are not set, then the values will be nil. If they are non-nil,
	// then return an error.
	if (serviceMode.ReplicatedJob != nil || serviceMode.GlobalJob != nil) && (updateConfig != nil || rollbackConfig != nil) {
		return service, errors.Errorf("update and rollback configuration is not supported for jobs")
	}

	networks := convertNetworks(options.networks)
	for i, net := range networks {
		nwID, err := resolveNetworkID(ctx, apiClient, net.Target)
		if err != nil {
			return service, err
		}
		networks[i].Target = nwID
	}
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Target < networks[j].Target
	})

	resources, err := options.resources.ToResourceRequirements()
	if err != nil {
		return service, err
	}

	capAdd, capDrop := opts.EffectiveCapAddCapDrop(options.capAdd.GetAll(), options.capDrop.GetAll())

	service = swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name:   options.name,
			Labels: opts.ConvertKVStringsToMap(options.labels.GetAll()),
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:      options.image,
				Args:       options.args,
				Command:    options.entrypoint.Value(),
				Env:        currentEnv,
				Hostname:   options.hostname,
				Labels:     opts.ConvertKVStringsToMap(options.containerLabels.GetAll()),
				Dir:        options.workdir,
				User:       options.user,
				Groups:     options.groups.GetAll(),
				StopSignal: options.stopSignal,
				TTY:        options.tty,
				ReadOnly:   options.readOnly,
				Mounts:     options.mounts.Value(),
				Init:       &options.init,
				DNSConfig: &swarm.DNSConfig{
					Nameservers: options.dns.GetAll(),
					Search:      options.dnsSearch.GetAll(),
					Options:     options.dnsOption.GetAll(),
				},
				Hosts:           convertExtraHostsToSwarmHosts(options.hosts.GetAll()),
				StopGracePeriod: options.ToStopGracePeriod(flags),
				Healthcheck:     healthConfig,
				Isolation:       container.Isolation(options.isolation),
				Sysctls:         opts.ConvertKVStringsToMap(options.sysctls.GetAll()),
				CapabilityAdd:   capAdd,
				CapabilityDrop:  capDrop,
				Ulimits:         options.ulimits.GetList(),
				OomScoreAdj:     options.oomScoreAdj,
			},
			Networks:      networks,
			Resources:     resources,
			RestartPolicy: options.restartPolicy.ToRestartPolicy(flags),
			Placement: &swarm.Placement{
				Constraints: options.constraints.GetAll(),
				Preferences: options.placementPrefs.prefs,
				MaxReplicas: options.maxReplicas,
			},
			LogDriver: options.logDriver.toLogDriver(),
		},
		Mode:           serviceMode,
		UpdateConfig:   updateConfig,
		RollbackConfig: rollbackConfig,
		EndpointSpec:   options.endpoint.ToEndpointSpec(),
	}

	if options.credentialSpec.String() != "" && options.credentialSpec.Value() != nil {
		service.TaskTemplate.ContainerSpec.Privileges = &swarm.Privileges{
			CredentialSpec: options.credentialSpec.Value(),
		}
	}

	return service, nil
}

type flagDefaults map[string]any

func (fd flagDefaults) getUint64(flagName string) uint64 {
	if val, ok := fd[flagName].(uint64); ok {
		return val
	}
	return 0
}

func (fd flagDefaults) getString(flagName string) string {
	if val, ok := fd[flagName].(string); ok {
		return val
	}
	return ""
}

func buildServiceDefaultFlagMapping() flagDefaults {
	defaultFlagValues := make(map[string]any)

	defaultFlagValues[flagStopGracePeriod], _ = gogotypes.DurationFromProto(defaults.Service.Task.GetContainer().StopGracePeriod)
	defaultFlagValues[flagRestartCondition] = `"` + defaultRestartCondition() + `"`
	defaultFlagValues[flagRestartDelay], _ = gogotypes.DurationFromProto(defaults.Service.Task.Restart.Delay)

	if defaults.Service.Task.Restart.MaxAttempts != 0 {
		defaultFlagValues[flagRestartMaxAttempts] = defaults.Service.Task.Restart.MaxAttempts
	}

	defaultRestartWindow, _ := gogotypes.DurationFromProto(defaults.Service.Task.Restart.Window)
	if defaultRestartWindow != 0 {
		defaultFlagValues[flagRestartWindow] = defaultRestartWindow
	}

	defaultFlagValues[flagUpdateParallelism] = defaults.Service.Update.Parallelism
	defaultFlagValues[flagUpdateDelay] = defaults.Service.Update.Delay
	defaultFlagValues[flagUpdateMonitor], _ = gogotypes.DurationFromProto(defaults.Service.Update.Monitor)
	defaultFlagValues[flagUpdateFailureAction] = `"` + strings.ToLower(api.UpdateConfig_FailureAction_name[int32(defaults.Service.Update.FailureAction)]) + `"`
	defaultFlagValues[flagUpdateMaxFailureRatio] = defaults.Service.Update.MaxFailureRatio
	defaultFlagValues[flagUpdateOrder] = `"` + defaultOrder(defaults.Service.Update.Order) + `"`

	defaultFlagValues[flagRollbackParallelism] = defaults.Service.Rollback.Parallelism
	defaultFlagValues[flagRollbackDelay] = defaults.Service.Rollback.Delay
	defaultFlagValues[flagRollbackMonitor], _ = gogotypes.DurationFromProto(defaults.Service.Rollback.Monitor)
	defaultFlagValues[flagRollbackFailureAction] = `"` + strings.ToLower(api.UpdateConfig_FailureAction_name[int32(defaults.Service.Rollback.FailureAction)]) + `"`
	defaultFlagValues[flagRollbackMaxFailureRatio] = defaults.Service.Rollback.MaxFailureRatio
	defaultFlagValues[flagRollbackOrder] = `"` + defaultOrder(defaults.Service.Rollback.Order) + `"`

	defaultFlagValues[flagEndpointMode] = "vip"

	return defaultFlagValues
}

func addDetachFlag(flags *pflag.FlagSet, detach *bool) {
	flags.BoolVarP(detach, flagDetach, "d", false, "Exit immediately instead of waiting for the service to converge")
	flags.SetAnnotation(flagDetach, "version", []string{"1.29"})
}

// addServiceFlags adds all flags that are common to both `create` and `update`.
// Any flags that are not common are added separately in the individual command
func addServiceFlags(flags *pflag.FlagSet, options *serviceOptions, defaultFlagValues flagDefaults) {
	flagDesc := func(flagName string, desc string) string {
		if defaultValue, ok := defaultFlagValues[flagName]; ok {
			return fmt.Sprintf("%s (default %v)", desc, defaultValue)
		}
		return desc
	}

	addDetachFlag(flags, &options.detach)
	flags.BoolVarP(&options.quiet, flagQuiet, "q", false, "Suppress progress output")

	flags.StringVarP(&options.workdir, flagWorkdir, "w", "", "Working directory inside the container")
	flags.StringVarP(&options.user, flagUser, "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	flags.Var(&options.credentialSpec, flagCredentialSpec, "Credential spec for managed service account (Windows only)")
	flags.SetAnnotation(flagCredentialSpec, "version", []string{"1.29"})
	flags.StringVar(&options.hostname, flagHostname, "", "Container hostname")
	flags.SetAnnotation(flagHostname, "version", []string{"1.25"})
	flags.Var(&options.entrypoint, flagEntrypoint, "Overwrite the default ENTRYPOINT of the image")
	flags.Var(&options.capAdd, flagCapAdd, "Add Linux capabilities")
	flags.SetAnnotation(flagCapAdd, "version", []string{"1.41"})
	flags.Var(&options.capDrop, flagCapDrop, "Drop Linux capabilities")
	flags.SetAnnotation(flagCapDrop, "version", []string{"1.41"})

	flags.Var(&options.resources.limitCPU, flagLimitCPU, "Limit CPUs")
	flags.Var(&options.resources.limitMemBytes, flagLimitMemory, "Limit Memory")
	flags.Var(&options.resources.resCPU, flagReserveCPU, "Reserve CPUs")
	flags.Var(&options.resources.resMemBytes, flagReserveMemory, "Reserve Memory")
	flags.Int64Var(&options.resources.limitPids, flagLimitPids, 0, "Limit maximum number of processes (default 0 = unlimited)")
	flags.SetAnnotation(flagLimitPids, "version", []string{"1.41"})

	flags.Var(&options.stopGrace, flagStopGracePeriod, flagDesc(flagStopGracePeriod, "Time to wait before force killing a container (ns|us|ms|s|m|h)"))
	flags.Var(&options.replicas, flagReplicas, "Number of tasks")
	flags.Var(&options.maxConcurrent, flagConcurrent, "Number of job tasks to run concurrently (default equal to --replicas)")
	flags.SetAnnotation(flagConcurrent, "version", []string{"1.41"})
	flags.Uint64Var(&options.maxReplicas, flagMaxReplicas, defaultFlagValues.getUint64(flagMaxReplicas), "Maximum number of tasks per node (default 0 = unlimited)")
	flags.SetAnnotation(flagMaxReplicas, "version", []string{"1.40"})

	flags.StringVar(&options.restartPolicy.condition, flagRestartCondition, "", flagDesc(flagRestartCondition, `Restart when condition is met ("none", "on-failure", "any")`))
	flags.Var(&options.restartPolicy.delay, flagRestartDelay, flagDesc(flagRestartDelay, "Delay between restart attempts (ns|us|ms|s|m|h)"))
	flags.Var(&options.restartPolicy.maxAttempts, flagRestartMaxAttempts, flagDesc(flagRestartMaxAttempts, "Maximum number of restarts before giving up"))

	flags.Var(&options.restartPolicy.window, flagRestartWindow, flagDesc(flagRestartWindow, "Window used to evaluate the restart policy (ns|us|ms|s|m|h)"))

	flags.Uint64Var(&options.update.parallelism, flagUpdateParallelism, defaultFlagValues.getUint64(flagUpdateParallelism), "Maximum number of tasks updated simultaneously (0 to update all at once)")
	flags.DurationVar(&options.update.delay, flagUpdateDelay, 0, flagDesc(flagUpdateDelay, "Delay between updates (ns|us|ms|s|m|h)"))
	flags.DurationVar(&options.update.monitor, flagUpdateMonitor, 0, flagDesc(flagUpdateMonitor, "Duration after each task update to monitor for failure (ns|us|ms|s|m|h)"))
	flags.SetAnnotation(flagUpdateMonitor, "version", []string{"1.25"})
	flags.StringVar(&options.update.onFailure, flagUpdateFailureAction, "", flagDesc(flagUpdateFailureAction, `Action on update failure ("pause", "continue", "rollback")`))
	flags.Var(&options.update.maxFailureRatio, flagUpdateMaxFailureRatio, flagDesc(flagUpdateMaxFailureRatio, "Failure rate to tolerate during an update"))
	flags.SetAnnotation(flagUpdateMaxFailureRatio, "version", []string{"1.25"})
	flags.StringVar(&options.update.order, flagUpdateOrder, "", flagDesc(flagUpdateOrder, `Update order ("start-first", "stop-first")`))
	flags.SetAnnotation(flagUpdateOrder, "version", []string{"1.29"})

	flags.Uint64Var(&options.rollback.parallelism, flagRollbackParallelism, defaultFlagValues.getUint64(flagRollbackParallelism),
		"Maximum number of tasks rolled back simultaneously (0 to roll back all at once)")
	flags.SetAnnotation(flagRollbackParallelism, "version", []string{"1.28"})
	flags.DurationVar(&options.rollback.delay, flagRollbackDelay, 0, flagDesc(flagRollbackDelay, "Delay between task rollbacks (ns|us|ms|s|m|h)"))
	flags.SetAnnotation(flagRollbackDelay, "version", []string{"1.28"})
	flags.DurationVar(&options.rollback.monitor, flagRollbackMonitor, 0, flagDesc(flagRollbackMonitor, "Duration after each task rollback to monitor for failure (ns|us|ms|s|m|h)"))
	flags.SetAnnotation(flagRollbackMonitor, "version", []string{"1.28"})
	flags.StringVar(&options.rollback.onFailure, flagRollbackFailureAction, "", flagDesc(flagRollbackFailureAction, `Action on rollback failure ("pause", "continue")`))
	flags.SetAnnotation(flagRollbackFailureAction, "version", []string{"1.28"})
	flags.Var(&options.rollback.maxFailureRatio, flagRollbackMaxFailureRatio, flagDesc(flagRollbackMaxFailureRatio, "Failure rate to tolerate during a rollback"))
	flags.SetAnnotation(flagRollbackMaxFailureRatio, "version", []string{"1.28"})
	flags.StringVar(&options.rollback.order, flagRollbackOrder, "", flagDesc(flagRollbackOrder, `Rollback order ("start-first", "stop-first")`))
	flags.SetAnnotation(flagRollbackOrder, "version", []string{"1.29"})

	flags.StringVar(&options.endpoint.mode, flagEndpointMode, defaultFlagValues.getString(flagEndpointMode), "Endpoint mode (vip or dnsrr)")

	flags.BoolVar(&options.registryAuth, flagRegistryAuth, false, "Send registry authentication details to swarm agents")
	flags.BoolVar(&options.noResolveImage, flagNoResolveImage, false, "Do not query the registry to resolve image digest and supported platforms")
	flags.SetAnnotation(flagNoResolveImage, "version", []string{"1.30"})

	flags.StringVar(&options.logDriver.name, flagLogDriver, "", "Logging driver for service")
	flags.Var(&options.logDriver.opts, flagLogOpt, "Logging driver options")

	flags.StringVar(&options.healthcheck.cmd, flagHealthCmd, "", "Command to run to check health")
	flags.SetAnnotation(flagHealthCmd, "version", []string{"1.25"})
	flags.Var(&options.healthcheck.interval, flagHealthInterval, "Time between running the check (ms|s|m|h)")
	flags.SetAnnotation(flagHealthInterval, "version", []string{"1.25"})
	flags.Var(&options.healthcheck.timeout, flagHealthTimeout, "Maximum time to allow one check to run (ms|s|m|h)")
	flags.SetAnnotation(flagHealthTimeout, "version", []string{"1.25"})
	flags.IntVar(&options.healthcheck.retries, flagHealthRetries, 0, "Consecutive failures needed to report unhealthy")
	flags.SetAnnotation(flagHealthRetries, "version", []string{"1.25"})
	flags.Var(&options.healthcheck.startPeriod, flagHealthStartPeriod, "Start period for the container to initialize before counting retries towards unstable (ms|s|m|h)")
	flags.SetAnnotation(flagHealthStartPeriod, "version", []string{"1.29"})
	flags.Var(&options.healthcheck.startInterval, flagHealthStartInterval, "Time between running the check during the start period (ms|s|m|h)")
	flags.SetAnnotation(flagHealthStartInterval, "version", []string{"1.44"})
	flags.BoolVar(&options.healthcheck.noHealthcheck, flagNoHealthcheck, false, "Disable any container-specified HEALTHCHECK")
	flags.SetAnnotation(flagNoHealthcheck, "version", []string{"1.25"})

	flags.BoolVarP(&options.tty, flagTTY, "t", false, "Allocate a pseudo-TTY")
	flags.SetAnnotation(flagTTY, "version", []string{"1.25"})

	flags.BoolVar(&options.readOnly, flagReadOnly, false, "Mount the container's root filesystem as read only")
	flags.SetAnnotation(flagReadOnly, "version", []string{"1.28"})

	flags.StringVar(&options.stopSignal, flagStopSignal, "", "Signal to stop the container")
	flags.SetAnnotation(flagStopSignal, "version", []string{"1.28"})
	flags.StringVar(&options.isolation, flagIsolation, "", "Service container isolation mode")
	flags.SetAnnotation(flagIsolation, "version", []string{"1.35"})
}

const (
	flagCredentialSpec          = "credential-spec" //nolint:gosec // ignore G101: Potential hardcoded credentials
	flagPlacementPref           = "placement-pref"
	flagPlacementPrefAdd        = "placement-pref-add"
	flagPlacementPrefRemove     = "placement-pref-rm"
	flagConstraint              = "constraint"
	flagConstraintRemove        = "constraint-rm"
	flagConstraintAdd           = "constraint-add"
	flagContainerLabel          = "container-label"
	flagContainerLabelRemove    = "container-label-rm"
	flagContainerLabelAdd       = "container-label-add"
	flagDetach                  = "detach"
	flagDNS                     = "dns"
	flagDNSRemove               = "dns-rm"
	flagDNSAdd                  = "dns-add"
	flagDNSOption               = "dns-option"
	flagDNSOptionRemove         = "dns-option-rm"
	flagDNSOptionAdd            = "dns-option-add"
	flagDNSSearch               = "dns-search"
	flagDNSSearchRemove         = "dns-search-rm"
	flagDNSSearchAdd            = "dns-search-add"
	flagEndpointMode            = "endpoint-mode"
	flagEntrypoint              = "entrypoint"
	flagEnv                     = "env"
	flagEnvFile                 = "env-file"
	flagEnvRemove               = "env-rm"
	flagEnvAdd                  = "env-add"
	flagGenericResourcesRemove  = "generic-resource-rm"
	flagGenericResourcesAdd     = "generic-resource-add"
	flagGroup                   = "group"
	flagGroupAdd                = "group-add"
	flagGroupRemove             = "group-rm"
	flagHost                    = "host"
	flagHostAdd                 = "host-add"
	flagHostRemove              = "host-rm"
	flagHostname                = "hostname"
	flagLabel                   = "label"
	flagLabelRemove             = "label-rm"
	flagLabelAdd                = "label-add"
	flagLimitCPU                = "limit-cpu"
	flagLimitMemory             = "limit-memory"
	flagLimitPids               = "limit-pids"
	flagMaxReplicas             = "replicas-max-per-node"
	flagConcurrent              = "max-concurrent"
	flagMode                    = "mode"
	flagMount                   = "mount"
	flagMountRemove             = "mount-rm"
	flagMountAdd                = "mount-add"
	flagName                    = "name"
	flagNetwork                 = "network"
	flagNetworkAdd              = "network-add"
	flagNetworkRemove           = "network-rm"
	flagPublish                 = "publish"
	flagPublishRemove           = "publish-rm"
	flagPublishAdd              = "publish-add"
	flagQuiet                   = "quiet"
	flagReadOnly                = "read-only"
	flagReplicas                = "replicas"
	flagReserveCPU              = "reserve-cpu"
	flagReserveMemory           = "reserve-memory"
	flagRestartCondition        = "restart-condition"
	flagRestartDelay            = "restart-delay"
	flagRestartMaxAttempts      = "restart-max-attempts"
	flagRestartWindow           = "restart-window"
	flagRollback                = "rollback"
	flagRollbackDelay           = "rollback-delay"
	flagRollbackFailureAction   = "rollback-failure-action"
	flagRollbackMaxFailureRatio = "rollback-max-failure-ratio"
	flagRollbackMonitor         = "rollback-monitor"
	flagRollbackOrder           = "rollback-order"
	flagRollbackParallelism     = "rollback-parallelism"
	flagInit                    = "init"
	flagSysCtl                  = "sysctl"
	flagSysCtlAdd               = "sysctl-add"
	flagSysCtlRemove            = "sysctl-rm"
	flagStopGracePeriod         = "stop-grace-period"
	flagStopSignal              = "stop-signal"
	flagTTY                     = "tty"
	flagUpdateDelay             = "update-delay"
	flagUpdateFailureAction     = "update-failure-action"
	flagUpdateMaxFailureRatio   = "update-max-failure-ratio" // #nosec G101 -- ignoring: Potential hardcoded credentials (gosec)
	flagUpdateMonitor           = "update-monitor"
	flagUpdateOrder             = "update-order"
	flagUpdateParallelism       = "update-parallelism"
	flagUser                    = "user"
	flagWorkdir                 = "workdir"
	flagRegistryAuth            = "with-registry-auth"
	flagNoResolveImage          = "no-resolve-image"
	flagLogDriver               = "log-driver"
	flagLogOpt                  = "log-opt"
	flagHealthCmd               = "health-cmd"
	flagHealthInterval          = "health-interval"
	flagHealthRetries           = "health-retries"
	flagHealthTimeout           = "health-timeout"
	flagHealthStartPeriod       = "health-start-period"
	flagHealthStartInterval     = "health-start-interval"
	flagNoHealthcheck           = "no-healthcheck"
	flagSecret                  = "secret"
	flagSecretAdd               = "secret-add"
	flagSecretRemove            = "secret-rm"
	flagConfig                  = "config"
	flagConfigAdd               = "config-add"
	flagConfigRemove            = "config-rm"
	flagIsolation               = "isolation"
	flagCapAdd                  = "cap-add"
	flagCapDrop                 = "cap-drop"
	flagUlimit                  = "ulimit"
	flagUlimitAdd               = "ulimit-add"
	flagUlimitRemove            = "ulimit-rm"
	flagOomScoreAdj             = "oom-score-adj"
)

func validateAPIVersion(c swarm.ServiceSpec, serverAPIVersion string) error {
	for _, m := range c.TaskTemplate.ContainerSpec.Mounts {
		if err := command.ValidateMountWithAPIVersion(m, serverAPIVersion); err != nil {
			return err
		}
	}
	return nil
}
