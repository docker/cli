package clientset

import (
	composev1beta1 "github.com/docker/cli/kubernetes/client/clientset/typed/compose/v1beta1"
	composev1beta2 "github.com/docker/cli/kubernetes/client/clientset/typed/compose/v1beta2"
	"github.com/golang/glog"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

// Interface defines the methods a compose kube client should have
// FIXME(vdemeester) is it required ?
type Interface interface {
	Discovery() discovery.DiscoveryInterface
	ComposeV1beta2() composev1beta2.ComposeV1beta2Interface
	ComposeV1beta1() composev1beta1.ComposeV1beta1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	*composev1beta2.ComposeV1beta2Client
	*composev1beta1.ComposeV1beta1Client
}

// ComposeV1beta2 retrieves the ComposeV1beta2Client
func (c *Clientset) ComposeV1beta2() composev1beta2.ComposeV1beta2Interface {
	if c == nil {
		return nil
	}
	return c.ComposeV1beta2Client
}

// ComposeV1beta1 retrieves the ComposeV1beta1Client
func (c *Clientset) ComposeV1beta1() composev1beta1.ComposeV1beta1Interface {
	if c == nil {
		return nil
	}
	return c.ComposeV1beta1Client
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.ComposeV1beta2Client, err = composev1beta2.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.ComposeV1beta1Client, err = composev1beta1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		glog.Errorf("failed to create the DiscoveryClient: %v", err)
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.ComposeV1beta2Client = composev1beta2.NewForConfigOrDie(c)
	cs.ComposeV1beta1Client = composev1beta1.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.ComposeV1beta2Client = composev1beta2.New(c)
	cs.ComposeV1beta1Client = composev1beta1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
