package notary

import (
	"github.com/docker/cli/cli/trust"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/client/changelist"
	"github.com/theupdateframework/notary/cryptoservice"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/storage"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/signed"
)

// GetOfflineNotaryRepository returns a OfflineNotaryRepository
func GetOfflineNotaryRepository(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
	return OfflineNotaryRepository{}, nil
}

// OfflineNotaryRepository is a mock Notary repository that is offline
type OfflineNotaryRepository struct{}

// Initialize creates a new repository by using rootKey as the root Key for the
// TUF repository.
func (o OfflineNotaryRepository) Initialize(rootKeyIDs []string, serverManagedRoles ...data.RoleName) error {
	return storage.ErrOffline{}
}

// InitializeWithCertificate initializes the repository with root keys and their corresponding certificates
func (o OfflineNotaryRepository) InitializeWithCertificate(rootKeyIDs []string, rootCerts []data.PublicKey, serverManagedRoles ...data.RoleName) error {
	return storage.ErrOffline{}
}

// Publish pushes the local changes in signed material to the remote notary-server
// Conceptually it performs an operation similar to a `git rebase`
func (o OfflineNotaryRepository) Publish() error {
	return storage.ErrOffline{}
}

// AddTarget creates new changelist entries to add a target to the given roles
// in the repository when the changelist gets applied at publish time.
func (o OfflineNotaryRepository) AddTarget(target *client.Target, roles ...data.RoleName) error {
	return nil
}

// RemoveTarget creates new changelist entries to remove a target from the given
// roles in the repository when the changelist gets applied at publish time.
func (o OfflineNotaryRepository) RemoveTarget(targetName string, roles ...data.RoleName) error {
	return nil
}

// ListTargets lists all targets for the current repository. The list of
// roles should be passed in order from highest to lowest priority.
func (o OfflineNotaryRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	return nil, storage.ErrOffline{}
}

// GetTargetByName returns a target by the given name.
func (o OfflineNotaryRepository) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	return nil, storage.ErrOffline{}
}

// GetAllTargetMetadataByName searches the entire delegation role tree to find the specified target by name for all
// roles, and returns a list of TargetSignedStructs for each time it finds the specified target.
func (o OfflineNotaryRepository) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	return nil, storage.ErrOffline{}
}

// GetChangelist returns the list of the repository's unpublished changes
func (o OfflineNotaryRepository) GetChangelist() (changelist.Changelist, error) {
	return changelist.NewMemChangelist(), nil
}

// ListRoles returns a list of RoleWithSignatures objects for this repo
func (o OfflineNotaryRepository) ListRoles() ([]client.RoleWithSignatures, error) {
	return nil, storage.ErrOffline{}
}

// GetDelegationRoles returns the keys and roles of the repository's delegations
func (o OfflineNotaryRepository) GetDelegationRoles() ([]data.Role, error) {
	return nil, storage.ErrOffline{}
}

// AddDelegation creates changelist entries to add provided delegation public keys and paths.
func (o OfflineNotaryRepository) AddDelegation(name data.RoleName, delegationKeys []data.PublicKey, paths []string) error {
	return nil
}

// AddDelegationRoleAndKeys creates a changelist entry to add provided delegation public keys.
func (o OfflineNotaryRepository) AddDelegationRoleAndKeys(name data.RoleName, delegationKeys []data.PublicKey) error {
	return nil
}

// AddDelegationPaths creates a changelist entry to add provided paths to an existing delegation.
func (o OfflineNotaryRepository) AddDelegationPaths(name data.RoleName, paths []string) error {
	return nil
}

// RemoveDelegationKeysAndPaths creates changelist entries to remove provided delegation key IDs and paths.
func (o OfflineNotaryRepository) RemoveDelegationKeysAndPaths(name data.RoleName, keyIDs, paths []string) error {
	return nil
}

// RemoveDelegationRole creates a changelist to remove all paths and keys from a role, and delete the role in its entirety.
func (o OfflineNotaryRepository) RemoveDelegationRole(name data.RoleName) error {
	return nil
}

// RemoveDelegationPaths creates a changelist entry to remove provided paths from an existing delegation.
func (o OfflineNotaryRepository) RemoveDelegationPaths(name data.RoleName, paths []string) error {
	return nil
}

// RemoveDelegationKeys creates a changelist entry to remove provided keys from an existing delegation.
func (o OfflineNotaryRepository) RemoveDelegationKeys(name data.RoleName, keyIDs []string) error {
	return nil
}

// ClearDelegationPaths creates a changelist entry to remove all paths from an existing delegation.
func (o OfflineNotaryRepository) ClearDelegationPaths(name data.RoleName) error {
	return nil
}

// Witness creates change objects to witness (i.e. re-sign) the given
// roles on the next publish. One change is created per role
func (o OfflineNotaryRepository) Witness(roles ...data.RoleName) ([]data.RoleName, error) {
	return nil, nil
}

// RotateKey rotates a private key and returns the public component from the remote server
func (o OfflineNotaryRepository) RotateKey(role data.RoleName, serverManagesKey bool, keyList []string) error {
	return storage.ErrOffline{}
}

// GetCryptoService is the getter for the repository's CryptoService
func (o OfflineNotaryRepository) GetCryptoService() signed.CryptoService {
	return nil
}

// SetLegacyVersions allows the number of legacy versions of the root
// to be inspected for old signing keys to be configured.
func (o OfflineNotaryRepository) SetLegacyVersions(version int) {}

// GetGUN is a getter for the GUN object from a Repository
func (o OfflineNotaryRepository) GetGUN() data.GUN {
	return data.GUN("gun")
}

// GetUninitializedNotaryRepository returns an UninitializedNotaryRepository
func GetUninitializedNotaryRepository(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
	return UninitializedNotaryRepository{}, nil
}

// UninitializedNotaryRepository is a mock Notary repository that is uninintialized
// it builds on top of the OfflineNotaryRepository, instead returning ErrRepositoryNotExist
// for any online operation
type UninitializedNotaryRepository struct {
	OfflineNotaryRepository
}

// Initialize creates a new repository by using rootKey as the root Key for the
// TUF repository.
func (u UninitializedNotaryRepository) Initialize(rootKeyIDs []string, serverManagedRoles ...data.RoleName) error {
	return client.ErrRepositoryNotExist{}
}

// InitializeWithCertificate initializes the repository with root keys and their corresponding certificates
func (u UninitializedNotaryRepository) InitializeWithCertificate(rootKeyIDs []string, rootCerts []data.PublicKey, serverManagedRoles ...data.RoleName) error {
	return client.ErrRepositoryNotExist{}
}

// Publish pushes the local changes in signed material to the remote notary-server
// Conceptually it performs an operation similar to a `git rebase`
func (u UninitializedNotaryRepository) Publish() error {
	return client.ErrRepositoryNotExist{}
}

// ListTargets lists all targets for the current repository. The list of
// roles should be passed in order from highest to lowest priority.
func (u UninitializedNotaryRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	return nil, client.ErrRepositoryNotExist{}
}

// GetTargetByName returns a target by the given name.
func (u UninitializedNotaryRepository) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	return nil, client.ErrRepositoryNotExist{}
}

// GetAllTargetMetadataByName searches the entire delegation role tree to find the specified target by name for all
// roles, and returns a list of TargetSignedStructs for each time it finds the specified target.
func (u UninitializedNotaryRepository) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	return nil, client.ErrRepositoryNotExist{}
}

// ListRoles returns a list of RoleWithSignatures objects for this repo
func (u UninitializedNotaryRepository) ListRoles() ([]client.RoleWithSignatures, error) {
	return nil, client.ErrRepositoryNotExist{}
}

// GetDelegationRoles returns the keys and roles of the repository's delegations
func (u UninitializedNotaryRepository) GetDelegationRoles() ([]data.Role, error) {
	return nil, client.ErrRepositoryNotExist{}
}

// RotateKey rotates a private key and returns the public component from the remote server
func (u UninitializedNotaryRepository) RotateKey(role data.RoleName, serverManagesKey bool, keyList []string) error {
	return client.ErrRepositoryNotExist{}
}

// GetEmptyTargetsNotaryRepository returns an EmptyTargetsNotaryRepository
func GetEmptyTargetsNotaryRepository(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
	return EmptyTargetsNotaryRepository{}, nil
}

// EmptyTargetsNotaryRepository is a mock Notary repository that is initialized
// but does not have any signed targets
type EmptyTargetsNotaryRepository struct {
	OfflineNotaryRepository
}

// Initialize creates a new repository by using rootKey as the root Key for the
// TUF repository.
func (e EmptyTargetsNotaryRepository) Initialize(rootKeyIDs []string, serverManagedRoles ...data.RoleName) error {
	return nil
}

// InitializeWithCertificate initializes the repository with root keys and their corresponding certificates
func (e EmptyTargetsNotaryRepository) InitializeWithCertificate(rootKeyIDs []string, rootCerts []data.PublicKey, serverManagedRoles ...data.RoleName) error {
	return nil
}

// Publish pushes the local changes in signed material to the remote notary-server
// Conceptually it performs an operation similar to a `git rebase`
func (e EmptyTargetsNotaryRepository) Publish() error {
	return nil
}

// ListTargets lists all targets for the current repository. The list of
// roles should be passed in order from highest to lowest priority.
func (e EmptyTargetsNotaryRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	return []*client.TargetWithRole{}, nil
}

// GetTargetByName returns a target by the given name.
func (e EmptyTargetsNotaryRepository) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	return nil, client.ErrNoSuchTarget(name)
}

// GetAllTargetMetadataByName searches the entire delegation role tree to find the specified target by name for all
// roles, and returns a list of TargetSignedStructs for each time it finds the specified target.
func (e EmptyTargetsNotaryRepository) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	return nil, client.ErrNoSuchTarget(name)
}

// ListRoles returns a list of RoleWithSignatures objects for this repo
func (e EmptyTargetsNotaryRepository) ListRoles() ([]client.RoleWithSignatures, error) {
	rootRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs:    []string{"rootID"},
			Threshold: 1,
		},
		Name: data.CanonicalRootRole,
	}

	targetsRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs:    []string{"targetsID"},
			Threshold: 1,
		},
		Name: data.CanonicalTargetsRole,
	}
	return []client.RoleWithSignatures{
		{Role: rootRole},
		{Role: targetsRole}}, nil
}

// GetDelegationRoles returns the keys and roles of the repository's delegations
func (e EmptyTargetsNotaryRepository) GetDelegationRoles() ([]data.Role, error) {
	return []data.Role{}, nil
}

// RotateKey rotates a private key and returns the public component from the remote server
func (e EmptyTargetsNotaryRepository) RotateKey(role data.RoleName, serverManagesKey bool, keyList []string) error {
	return nil
}

// GetLoadedNotaryRepository returns a LoadedNotaryRepository
func GetLoadedNotaryRepository(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
	return LoadedNotaryRepository{}, nil
}

// LoadedNotaryRepository is a mock Notary repository that is loaded with targets, delegations, and keys
type LoadedNotaryRepository struct {
	EmptyTargetsNotaryRepository
	statefulCryptoService signed.CryptoService
}

// LoadedNotaryRepository has three delegations:
// - targets/releases: includes keys A and B
// - targets/alice: includes key A
// - targets/bob: includes key B
var loadedReleasesRole = data.DelegationRole{
	BaseRole: data.BaseRole{
		Name:      "targets/releases",
		Keys:      map[string]data.PublicKey{"A": nil, "B": nil},
		Threshold: 1,
	},
}
var loadedAliceRole = data.DelegationRole{
	BaseRole: data.BaseRole{
		Name:      "targets/alice",
		Keys:      map[string]data.PublicKey{"A": nil},
		Threshold: 1,
	},
}
var loadedBobRole = data.DelegationRole{
	BaseRole: data.BaseRole{
		Name:      "targets/bob",
		Keys:      map[string]data.PublicKey{"B": nil},
		Threshold: 1,
	},
}
var loadedDelegationRoles = []data.Role{
	{
		Name: loadedReleasesRole.Name,
		RootRole: data.RootRole{
			KeyIDs:    []string{"A", "B"},
			Threshold: 1,
		},
	},
	{
		Name: loadedAliceRole.Name,
		RootRole: data.RootRole{
			KeyIDs:    []string{"A"},
			Threshold: 1,
		},
	},
	{
		Name: loadedBobRole.Name,
		RootRole: data.RootRole{
			KeyIDs:    []string{"B"},
			Threshold: 1,
		},
	},
}
var loadedTargetsRole = data.DelegationRole{
	BaseRole: data.BaseRole{
		Name:      data.CanonicalTargetsRole,
		Keys:      map[string]data.PublicKey{"C": nil},
		Threshold: 1,
	},
}

// LoadedNotaryRepository has three targets:
// - red: signed by targets/releases, targets/alice, targets/bob
// - blue: signed by targets/releases, targets/alice
// - green: signed by targets/releases
var loadedRedTarget = client.Target{
	Name:   "red",
	Hashes: data.Hashes{"sha256": []byte("red-digest")},
}
var loadedBlueTarget = client.Target{
	Name:   "blue",
	Hashes: data.Hashes{"sha256": []byte("blue-digest")},
}
var loadedGreenTarget = client.Target{
	Name:   "green",
	Hashes: data.Hashes{"sha256": []byte("green-digest")},
}
var loadedTargets = []client.TargetSignedStruct{
	// red is signed by all three delegations
	{Target: loadedRedTarget, Role: loadedReleasesRole},
	{Target: loadedRedTarget, Role: loadedAliceRole},
	{Target: loadedRedTarget, Role: loadedBobRole},

	// blue is signed by targets/releases, targets/alice
	{Target: loadedBlueTarget, Role: loadedReleasesRole},
	{Target: loadedBlueTarget, Role: loadedAliceRole},

	// green is signed by targets/releases
	{Target: loadedGreenTarget, Role: loadedReleasesRole},
}

// ListRoles returns a list of RoleWithSignatures objects for this repo
func (l LoadedNotaryRepository) ListRoles() ([]client.RoleWithSignatures, error) {
	rootRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs:    []string{"rootID"},
			Threshold: 1,
		},
		Name: data.CanonicalRootRole,
	}

	targetsRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs:    []string{"targetsID"},
			Threshold: 1,
		},
		Name: data.CanonicalTargetsRole,
	}

	aliceRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs:    []string{"A"},
			Threshold: 1,
		},
		Name: data.RoleName("targets/alice"),
	}

	bobRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs:    []string{"B"},
			Threshold: 1,
		},
		Name: data.RoleName("targets/bob"),
	}

	releasesRole := data.Role{
		RootRole: data.RootRole{
			KeyIDs:    []string{"A", "B"},
			Threshold: 1,
		},
		Name: data.RoleName("targets/releases"),
	}
	// have releases only signed off by Alice last
	releasesSig := []data.Signature{{KeyID: "A"}}

	return []client.RoleWithSignatures{
		{Role: rootRole},
		{Role: targetsRole},
		{Role: aliceRole},
		{Role: bobRole},
		{Role: releasesRole, Signatures: releasesSig},
	}, nil
}

// ListTargets lists all targets for the current repository. The list of
// roles should be passed in order from highest to lowest priority.
func (l LoadedNotaryRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	filteredTargets := []*client.TargetWithRole{}
	for _, tgt := range loadedTargets {
		if len(roles) == 0 || (len(roles) > 0 && roles[0] == tgt.Role.Name) {
			filteredTargets = append(filteredTargets, &client.TargetWithRole{Target: tgt.Target, Role: tgt.Role.Name})
		}
	}
	return filteredTargets, nil
}

// GetTargetByName returns a target by the given name.
func (l LoadedNotaryRepository) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	for _, tgt := range loadedTargets {
		if name == tgt.Target.Name {
			if len(roles) == 0 || (len(roles) > 0 && roles[0] == tgt.Role.Name) {
				return &client.TargetWithRole{Target: tgt.Target, Role: tgt.Role.Name}, nil
			}
		}
	}
	return nil, client.ErrNoSuchTarget(name)
}

// GetAllTargetMetadataByName searches the entire delegation role tree to find the specified target by name for all
// roles, and returns a list of TargetSignedStructs for each time it finds the specified target.
func (l LoadedNotaryRepository) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	if name == "" {
		return loadedTargets, nil
	}
	filteredTargets := []client.TargetSignedStruct{}
	for _, tgt := range loadedTargets {
		if name == tgt.Target.Name {
			filteredTargets = append(filteredTargets, tgt)
		}
	}
	if len(filteredTargets) == 0 {
		return nil, client.ErrNoSuchTarget(name)
	}
	return filteredTargets, nil
}

// GetGUN is a getter for the GUN object from a Repository
func (l LoadedNotaryRepository) GetGUN() data.GUN {
	return data.GUN("signed-repo")
}

// GetDelegationRoles returns the keys and roles of the repository's delegations
func (l LoadedNotaryRepository) GetDelegationRoles() ([]data.Role, error) {
	return loadedDelegationRoles, nil
}

// GetCryptoService is the getter for the repository's CryptoService
func (l LoadedNotaryRepository) GetCryptoService() signed.CryptoService {
	if l.statefulCryptoService == nil {
		// give it an in-memory cryptoservice with a root key and targets key
		l.statefulCryptoService = cryptoservice.NewCryptoService(trustmanager.NewKeyMemoryStore(passphrase.ConstantRetriever("password")))
		l.statefulCryptoService.AddKey(data.CanonicalRootRole, l.GetGUN(), nil)
		l.statefulCryptoService.AddKey(data.CanonicalTargetsRole, l.GetGUN(), nil)
	}
	return l.statefulCryptoService
}

// GetLoadedWithNoSignersNotaryRepository returns a LoadedWithNoSignersNotaryRepository
func GetLoadedWithNoSignersNotaryRepository(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
	return LoadedWithNoSignersNotaryRepository{}, nil
}

// LoadedWithNoSignersNotaryRepository is a mock Notary repository that is loaded with targets but no delegations
// it only contains the green target
type LoadedWithNoSignersNotaryRepository struct {
	LoadedNotaryRepository
}

// ListTargets lists all targets for the current repository. The list of
// roles should be passed in order from highest to lowest priority.
func (l LoadedWithNoSignersNotaryRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	filteredTargets := []*client.TargetWithRole{}
	for _, tgt := range loadedTargets {
		if len(roles) == 0 || (len(roles) > 0 && roles[0] == tgt.Role.Name) {
			filteredTargets = append(filteredTargets, &client.TargetWithRole{Target: tgt.Target, Role: tgt.Role.Name})
		}
	}
	return filteredTargets, nil
}

// GetTargetByName returns a target by the given name.
func (l LoadedWithNoSignersNotaryRepository) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	if name == "" || name == loadedGreenTarget.Name {
		return &client.TargetWithRole{Target: loadedGreenTarget, Role: data.CanonicalTargetsRole}, nil
	}
	return nil, client.ErrNoSuchTarget(name)
}

// GetAllTargetMetadataByName searches the entire delegation role tree to find the specified target by name for all
// roles, and returns a list of TargetSignedStructs for each time it finds the specified target.
func (l LoadedWithNoSignersNotaryRepository) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	if name == "" || name == loadedGreenTarget.Name {
		return []client.TargetSignedStruct{{Target: loadedGreenTarget, Role: loadedTargetsRole}}, nil
	}
	return nil, client.ErrNoSuchTarget(name)
}

// GetDelegationRoles returns the keys and roles of the repository's delegations
func (l LoadedWithNoSignersNotaryRepository) GetDelegationRoles() ([]data.Role, error) {
	return []data.Role{}, nil
}
