package builders

import (
	"time"

	"github.com/moby/moby/api/types/swarm"
)

// Secret creates a secret with default values.
// Any number of secret builder functions can be passed to augment it.
func Secret(builders ...func(secret *swarm.Secret)) *swarm.Secret {
	secret := &swarm.Secret{}

	for _, builder := range builders {
		builder(secret)
	}

	return secret
}

// SecretLabels sets the secret's labels
func SecretLabels(labels map[string]string) func(secret *swarm.Secret) {
	return func(secret *swarm.Secret) {
		secret.Spec.Labels = labels
	}
}

// SecretName sets the secret's name
func SecretName(name string) func(secret *swarm.Secret) {
	return func(secret *swarm.Secret) {
		secret.Spec.Name = name
	}
}

// SecretDriver sets the secret's driver name
func SecretDriver(driver string) func(secret *swarm.Secret) {
	return func(secret *swarm.Secret) {
		secret.Spec.Driver = &swarm.Driver{
			Name: driver,
		}
	}
}

// SecretID sets the secret's ID
func SecretID(id string) func(secret *swarm.Secret) {
	return func(secret *swarm.Secret) {
		secret.ID = id
	}
}

// SecretVersion sets the version for the secret
func SecretVersion(v swarm.Version) func(*swarm.Secret) {
	return func(secret *swarm.Secret) {
		secret.Version = v
	}
}

// SecretCreatedAt sets the creation time for the secret
func SecretCreatedAt(t time.Time) func(*swarm.Secret) {
	return func(secret *swarm.Secret) {
		secret.CreatedAt = t
	}
}

// SecretUpdatedAt sets the update time for the secret
func SecretUpdatedAt(t time.Time) func(*swarm.Secret) {
	return func(secret *swarm.Secret) {
		secret.UpdatedAt = t
	}
}
