package swarm

import (
	"encoding/csv"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

const (
	defaultListenAddr = "0.0.0.0:2377"

	flagCertExpiry                = "cert-expiry"
	flagDispatcherHeartbeat       = "dispatcher-heartbeat"
	flagListenAddr                = "listen-addr"
	flagAdvertiseAddr             = "advertise-addr"
	flagDataPathAddr              = "data-path-addr"
	flagDataPathPort              = "data-path-port"
	flagDefaultAddrPool           = "default-addr-pool"
	flagDefaultAddrPoolMaskLength = "default-addr-pool-mask-length"
	flagQuiet                     = "quiet"
	flagRotate                    = "rotate"
	flagToken                     = "token"
	flagTaskHistoryLimit          = "task-history-limit"
	flagExternalCA                = "external-ca"
	flagMaxSnapshots              = "max-snapshots"
	flagSnapshotInterval          = "snapshot-interval"
	flagAutolock                  = "autolock"
	flagAvailability              = "availability"
	flagCACert                    = "ca-cert"
	flagCAKey                     = "ca-key"
)

type swarmOptions struct {
	swarmCAOptions
	taskHistoryLimit    int64
	dispatcherHeartbeat time.Duration
	maxSnapshots        uint64
	snapshotInterval    uint64
	autolock            bool
}

// NodeAddrOption is a pflag.Value for listening addresses
type NodeAddrOption struct {
	addr string
}

// String prints the representation of this flag
func (a *NodeAddrOption) String() string {
	return a.Value()
}

// Set the value for this flag
func (a *NodeAddrOption) Set(value string) error {
	addr, err := opts.ParseTCPAddr(value, a.addr)
	if err != nil {
		return err
	}
	a.addr = addr
	return nil
}

// Type returns the type of this flag
func (a *NodeAddrOption) Type() string {
	return "node-addr"
}

// Value returns the value of this option as addr:port
func (a *NodeAddrOption) Value() string {
	return strings.TrimPrefix(a.addr, "tcp://")
}

// NewNodeAddrOption returns a new node address option
func NewNodeAddrOption(addr string) NodeAddrOption {
	return NodeAddrOption{addr}
}

// NewListenAddrOption returns a NodeAddrOption with default values
func NewListenAddrOption() NodeAddrOption {
	return NewNodeAddrOption(defaultListenAddr)
}

// ExternalCAOption is a Value type for parsing external CA specifications.
type ExternalCAOption struct {
	values []*swarm.ExternalCA
}

// Set parses an external CA option.
func (m *ExternalCAOption) Set(value string) error {
	parsed, err := parseExternalCA(value)
	if err != nil {
		return err
	}

	m.values = append(m.values, parsed)
	return nil
}

// Type returns the type of this option.
func (m *ExternalCAOption) Type() string {
	return "external-ca"
}

// String returns a string repr of this option.
func (m *ExternalCAOption) String() string {
	externalCAs := []string{}
	for _, externalCA := range m.values {
		repr := fmt.Sprintf("%s: %s", externalCA.Protocol, externalCA.URL)
		externalCAs = append(externalCAs, repr)
	}
	return strings.Join(externalCAs, ", ")
}

// Value returns the external CAs
func (m *ExternalCAOption) Value() []*swarm.ExternalCA {
	return m.values
}

// PEMFile represents the path to a pem-formatted file
type PEMFile struct {
	path, contents string
}

// Type returns the type of this option.
func (p *PEMFile) Type() string {
	return "pem-file"
}

// String returns the path to the pem file
func (p *PEMFile) String() string {
	return p.path
}

// Set parses a root rotation option
func (p *PEMFile) Set(value string) error {
	contents, err := os.ReadFile(value)
	if err != nil {
		return err
	}
	if pemBlock, _ := pem.Decode(contents); pemBlock == nil {
		return errors.New("file contents must be in PEM format")
	}
	p.contents, p.path = string(contents), value
	return nil
}

// Contents returns the contents of the PEM file
func (p *PEMFile) Contents() string {
	return p.contents
}

// parseExternalCA parses an external CA specification from the command line,
// such as protocol=cfssl,url=https://example.com.
func parseExternalCA(caSpec string) (*swarm.ExternalCA, error) {
	csvReader := csv.NewReader(strings.NewReader(caSpec))
	fields, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	externalCA := swarm.ExternalCA{
		Options: make(map[string]string),
	}

	var (
		hasProtocol bool
		hasURL      bool
	)

	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			return nil, errors.Errorf("invalid field '%s' must be a key=value pair", field)
		}

		// TODO(thaJeztah): these options should not be case-insensitive.
		switch strings.ToLower(key) {
		case "protocol":
			hasProtocol = true
			if strings.ToLower(value) == string(swarm.ExternalCAProtocolCFSSL) {
				externalCA.Protocol = swarm.ExternalCAProtocolCFSSL
			} else {
				return nil, errors.Errorf("unrecognized external CA protocol %s", value)
			}
		case "url":
			hasURL = true
			externalCA.URL = value
		case "cacert":
			cacontents, err := os.ReadFile(value)
			if err != nil {
				return nil, errors.Wrap(err, "unable to read CA cert for external CA")
			}
			if pemBlock, _ := pem.Decode(cacontents); pemBlock == nil {
				return nil, errors.New("CA cert for external CA must be in PEM format")
			}
			externalCA.CACert = string(cacontents)
		default:
			externalCA.Options[key] = value
		}
	}

	if !hasProtocol {
		return nil, errors.New("the external-ca option needs a protocol= parameter")
	}
	if !hasURL {
		return nil, errors.New("the external-ca option needs a url= parameter")
	}

	return &externalCA, nil
}

func addSwarmCAFlags(flags *pflag.FlagSet, options *swarmCAOptions) {
	flags.DurationVar(&options.nodeCertExpiry, flagCertExpiry, 90*24*time.Hour, "Validity period for node certificates (ns|us|ms|s|m|h)")
	flags.Var(&options.externalCA, flagExternalCA, "Specifications of one or more certificate signing endpoints")
}

func addSwarmFlags(flags *pflag.FlagSet, options *swarmOptions) {
	flags.Int64Var(&options.taskHistoryLimit, flagTaskHistoryLimit, 5, "Task history retention limit")
	flags.DurationVar(&options.dispatcherHeartbeat, flagDispatcherHeartbeat, 5*time.Second, "Dispatcher heartbeat period (ns|us|ms|s|m|h)")
	flags.Uint64Var(&options.maxSnapshots, flagMaxSnapshots, 0, "Number of additional Raft snapshots to retain")
	flags.SetAnnotation(flagMaxSnapshots, "version", []string{"1.25"})
	flags.Uint64Var(&options.snapshotInterval, flagSnapshotInterval, 10000, "Number of log entries between Raft snapshots")
	flags.SetAnnotation(flagSnapshotInterval, "version", []string{"1.25"})
	addSwarmCAFlags(flags, &options.swarmCAOptions)
}

func (o *swarmOptions) mergeSwarmSpec(spec *swarm.Spec, flags *pflag.FlagSet, caCert string) {
	if flags.Changed(flagTaskHistoryLimit) {
		spec.Orchestration.TaskHistoryRetentionLimit = &o.taskHistoryLimit
	}
	if flags.Changed(flagDispatcherHeartbeat) {
		spec.Dispatcher.HeartbeatPeriod = o.dispatcherHeartbeat
	}
	if flags.Changed(flagMaxSnapshots) {
		spec.Raft.KeepOldSnapshots = &o.maxSnapshots
	}
	if flags.Changed(flagSnapshotInterval) {
		spec.Raft.SnapshotInterval = o.snapshotInterval
	}
	if flags.Changed(flagAutolock) {
		spec.EncryptionConfig.AutoLockManagers = o.autolock
	}
	o.mergeSwarmSpecCAFlags(spec, flags, caCert)
}

type swarmCAOptions struct {
	nodeCertExpiry time.Duration
	externalCA     ExternalCAOption
}

func (o *swarmCAOptions) mergeSwarmSpecCAFlags(spec *swarm.Spec, flags *pflag.FlagSet, caCert string) {
	if flags.Changed(flagCertExpiry) {
		spec.CAConfig.NodeCertExpiry = o.nodeCertExpiry
	}
	if flags.Changed(flagExternalCA) {
		spec.CAConfig.ExternalCAs = o.externalCA.Value()
		for _, ca := range spec.CAConfig.ExternalCAs {
			ca.CACert = caCert
		}
	}
}

func (o *swarmOptions) ToSpec(flags *pflag.FlagSet) swarm.Spec {
	var spec swarm.Spec
	o.mergeSwarmSpec(&spec, flags, "")
	return spec
}
