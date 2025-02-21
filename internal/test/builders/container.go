package builders // import "docker.com/cli/v28/internal/test/builders"

import (
	"time"

	"github.com/docker/docker/api/types/container"
)

// Container creates a container with default values.
// Any number of container function builder can be passed to augment it.
func Container(name string, builders ...func(c *container.Summary)) *container.Summary {
	// now := time.Now()
	// onehourago := now.Add(-120 * time.Minute)
	ctr := &container.Summary{
		ID:      "container_id",
		Names:   []string{"/" + name},
		Command: "top",
		Image:   "busybox:latest",
		Status:  "Up 1 minute",
		Created: time.Now().Add(-1 * time.Minute).Unix(),
	}

	for _, builder := range builders {
		builder(ctr)
	}

	return ctr
}

// WithLabel adds a label to the container
func WithLabel(key, value string) func(*container.Summary) {
	return func(c *container.Summary) {
		if c.Labels == nil {
			c.Labels = map[string]string{}
		}
		c.Labels[key] = value
	}
}

// WithName adds a name to the container
func WithName(name string) func(*container.Summary) {
	return func(c *container.Summary) {
		c.Names = append(c.Names, "/"+name)
	}
}

// WithPort adds a port mapping to the container
func WithPort(privatePort, publicPort uint16, builders ...func(*container.Port)) func(*container.Summary) {
	return func(c *container.Summary) {
		if c.Ports == nil {
			c.Ports = []container.Port{}
		}
		port := &container.Port{
			PrivatePort: privatePort,
			PublicPort:  publicPort,
		}
		for _, builder := range builders {
			builder(port)
		}
		c.Ports = append(c.Ports, *port)
	}
}

// WithSize adds size in bytes to the container
func WithSize(size int64) func(*container.Summary) {
	return func(c *container.Summary) {
		if size >= 0 {
			c.SizeRw = size
		}
	}
}

// IP sets the ip of the port
func IP(ip string) func(*container.Port) {
	return func(p *container.Port) {
		p.IP = ip
	}
}

// TCP sets the port to tcp
func TCP(p *container.Port) {
	p.Type = "tcp"
}

// UDP sets the port to udp
func UDP(p *container.Port) {
	p.Type = "udp"
}
