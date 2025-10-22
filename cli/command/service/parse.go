package service

import (
	"context"
	"fmt"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

// ParseSecrets retrieves the secrets with the requested names and fills
// secret IDs into the secret references.
func ParseSecrets(ctx context.Context, apiClient client.SecretAPIClient, requestedSecrets []*swarm.SecretReference) ([]*swarm.SecretReference, error) {
	if len(requestedSecrets) == 0 {
		return []*swarm.SecretReference{}, nil
	}

	secretRefs := make(map[string]*swarm.SecretReference)

	for _, secret := range requestedSecrets {
		if _, exists := secretRefs[secret.File.Name]; exists {
			return nil, fmt.Errorf("duplicate secret target for %s not allowed", secret.SecretName)
		}
		secretRef := new(swarm.SecretReference)
		*secretRef = *secret
		secretRefs[secret.File.Name] = secretRef
	}

	args := make(client.Filters)
	for _, s := range secretRefs {
		args.Add("name", s.SecretName)
	}

	res, err := apiClient.SecretList(ctx, client.SecretListOptions{
		Filters: args,
	})
	if err != nil {
		return nil, err
	}

	foundSecrets := make(map[string]string)
	for _, secret := range res.Items {
		foundSecrets[secret.Spec.Annotations.Name] = secret.ID
	}

	addedSecrets := make([]*swarm.SecretReference, 0, len(secretRefs))
	for _, ref := range secretRefs {
		id, ok := foundSecrets[ref.SecretName]
		if !ok {
			return nil, fmt.Errorf("secret not found: %s", ref.SecretName)
		}

		// set the id for the ref to properly assign in swarm
		// since swarm needs the ID instead of the name
		ref.SecretID = id
		addedSecrets = append(addedSecrets, ref)
	}

	return addedSecrets, nil
}

// ParseConfigs retrieves the configs from the requested names and converts
// them to config references to use with the spec
func ParseConfigs(ctx context.Context, apiClient client.ConfigAPIClient, requestedConfigs []*swarm.ConfigReference) ([]*swarm.ConfigReference, error) {
	if len(requestedConfigs) == 0 {
		return []*swarm.ConfigReference{}, nil
	}

	// the configRefs map has two purposes: it prevents duplication of config
	// target filenames. It is used to get all configs, so we can resolve
	// their IDs. unfortunately, there are other targets for ConfigReferences,
	// besides just a File; specifically, the Runtime target, which is used for
	// CredentialSpecs. Therefore, we need to have a list of ConfigReferences
	// that are not File targets as well. at this time of writing, the only use
	// for Runtime targets is CredentialSpecs. However, to future-proof this
	// functionality, we should handle the case where multiple Runtime targets
	// are in use for the same Config, and we should deduplicate
	// such ConfigReferences, as no matter how many times the Config is used,
	// it is only needed to be referenced once.
	configRefs := make(map[string]*swarm.ConfigReference)
	runtimeRefs := make(map[string]*swarm.ConfigReference)

	for _, config := range requestedConfigs {
		// copy the config, so we don't mutate the args
		configRef := new(swarm.ConfigReference)
		*configRef = *config

		if config.Runtime != nil {
			// by assigning to a map based on ConfigName, if the same Config
			// is required as a Runtime target for multiple purposes, we only
			// include it once in the final set of configs.
			runtimeRefs[config.ConfigName] = config
			// continue, so we skip the logic below for handling file-type
			// configs
			continue
		}

		if _, exists := configRefs[config.File.Name]; exists {
			return nil, fmt.Errorf("duplicate config target for %s not allowed", config.ConfigName)
		}

		configRefs[config.File.Name] = configRef
	}

	args := make(client.Filters)
	for _, s := range configRefs {
		args.Add("name", s.ConfigName)
	}
	for _, s := range runtimeRefs {
		args.Add("name", s.ConfigName)
	}

	res, err := apiClient.ConfigList(ctx, client.ConfigListOptions{
		Filters: args,
	})
	if err != nil {
		return nil, err
	}

	foundConfigs := make(map[string]string)
	for _, config := range res.Items {
		foundConfigs[config.Spec.Annotations.Name] = config.ID
	}

	addedConfigs := make([]*swarm.ConfigReference, 0, len(configRefs))
	for _, ref := range configRefs {
		id, ok := foundConfigs[ref.ConfigName]
		if !ok {
			return nil, fmt.Errorf("config not found: %s", ref.ConfigName)
		}

		// set the id for the ref to properly assign in swarm
		// since swarm needs the ID instead of the name
		ref.ConfigID = id
		addedConfigs = append(addedConfigs, ref)
	}

	// unfortunately, because the key of configRefs and runtimeRefs is different
	// values that may collide, we can't just do some fancy trickery to
	// concat maps, we need to do two separate loops
	for _, ref := range runtimeRefs {
		id, ok := foundConfigs[ref.ConfigName]
		if !ok {
			return nil, fmt.Errorf("config not found: %s", ref.ConfigName)
		}

		ref.ConfigID = id
		addedConfigs = append(addedConfigs, ref)
	}

	return addedConfigs, nil
}
