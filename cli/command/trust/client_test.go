package trust

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

// Sample mock CLI interfaces

func getOfflineNotaryRepository(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
	return OfflineNotaryRepository{}, nil
}

// OfflineNotaryRepository is a mock Notary repository that is offline
type OfflineNotaryRepository struct{}

func (o OfflineNotaryRepository) Initialize(rootKeyIDs []string, serverManagedRoles ...data.RoleName) error {
	return storage.ErrOffline{}
}

func (o OfflineNotaryRepository) InitializeWithCertificate(rootKeyIDs []string, rootCerts []data.PublicKey, serverManagedRoles ...data.RoleName) error {
	return storage.ErrOffline{}
}
func (o OfflineNotaryRepository) Publish() error {
	return storage.ErrOffline{}
}

func (o OfflineNotaryRepository) AddTarget(target *client.Target, roles ...data.RoleName) error {
	return nil
}
func (o OfflineNotaryRepository) RemoveTarget(targetName string, roles ...data.RoleName) error {
	return nil
}
func (o OfflineNotaryRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	return nil, storage.ErrOffline{}
}

func (o OfflineNotaryRepository) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	return nil, storage.ErrOffline{}
}

func (o OfflineNotaryRepository) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	return nil, storage.ErrOffline{}
}

func (o OfflineNotaryRepository) GetChangelist() (changelist.Changelist, error) {
	return changelist.NewMemChangelist(), nil
}

func (o OfflineNotaryRepository) ListRoles() ([]client.RoleWithSignatures, error) {
	return nil, storage.ErrOffline{}
}

func (o OfflineNotaryRepository) GetDelegationRoles() ([]data.Role, error) {
	return nil, storage.ErrOffline{}
}

func (o OfflineNotaryRepository) AddDelegation(name data.RoleName, delegationKeys []data.PublicKey, paths []string) error {
	return nil
}

func (o OfflineNotaryRepository) AddDelegationRoleAndKeys(name data.RoleName, delegationKeys []data.PublicKey) error {
	return nil
}

func (o OfflineNotaryRepository) AddDelegationPaths(name data.RoleName, paths []string) error {
	return nil
}

func (o OfflineNotaryRepository) RemoveDelegationKeysAndPaths(name data.RoleName, keyIDs, paths []string) error {
	return nil
}

func (o OfflineNotaryRepository) RemoveDelegationRole(name data.RoleName) error {
	return nil
}

func (o OfflineNotaryRepository) RemoveDelegationPaths(name data.RoleName, paths []string) error {
	return nil
}

func (o OfflineNotaryRepository) RemoveDelegationKeys(name data.RoleName, keyIDs []string) error {
	return nil
}

func (o OfflineNotaryRepository) ClearDelegationPaths(name data.RoleName) error {
	return nil
}

func (o OfflineNotaryRepository) Witness(roles ...data.RoleName) ([]data.RoleName, error) {
	return nil, nil
}

func (o OfflineNotaryRepository) RotateKey(role data.RoleName, serverManagesKey bool, keyList []string) error {
	return storage.ErrOffline{}
}

func (o OfflineNotaryRepository) GetCryptoService() signed.CryptoService {
	return nil
}

func (o OfflineNotaryRepository) SetLegacyVersions(version int) {}

func (o OfflineNotaryRepository) GetGUN() data.GUN {
	return data.GUN("gun")
}

func getUninitializedNotaryRepository(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
	return UninitializedNotaryRepository{}, nil
}

// UninitializedNotaryRepository is a mock Notary repository that is uninintialized
// it builds on top of the OfflineNotaryRepository, instead returning ErrRepositoryNotExist
// for any online operation
type UninitializedNotaryRepository struct {
	OfflineNotaryRepository
}

func (u UninitializedNotaryRepository) Initialize(rootKeyIDs []string, serverManagedRoles ...data.RoleName) error {
	return client.ErrRepositoryNotExist{}
}

func (u UninitializedNotaryRepository) InitializeWithCertificate(rootKeyIDs []string, rootCerts []data.PublicKey, serverManagedRoles ...data.RoleName) error {
	return client.ErrRepositoryNotExist{}
}
func (u UninitializedNotaryRepository) Publish() error {
	return client.ErrRepositoryNotExist{}
}

func (u UninitializedNotaryRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	return nil, client.ErrRepositoryNotExist{}
}

func (u UninitializedNotaryRepository) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	return nil, client.ErrRepositoryNotExist{}
}

func (u UninitializedNotaryRepository) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	return nil, client.ErrRepositoryNotExist{}
}

func (u UninitializedNotaryRepository) ListRoles() ([]client.RoleWithSignatures, error) {
	return nil, client.ErrRepositoryNotExist{}
}

func (u UninitializedNotaryRepository) GetDelegationRoles() ([]data.Role, error) {
	return nil, client.ErrRepositoryNotExist{}
}

func (u UninitializedNotaryRepository) RotateKey(role data.RoleName, serverManagesKey bool, keyList []string) error {
	return client.ErrRepositoryNotExist{}
}

func getEmptyTargetsNotaryRepository(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
	return EmptyTargetsNotaryRepository{}, nil
}

// EmptyTargetsNotaryRepository is a mock Notary repository that is initialized
// but does not have any signed targets
type EmptyTargetsNotaryRepository struct {
	OfflineNotaryRepository
}

func (e EmptyTargetsNotaryRepository) Initialize(rootKeyIDs []string, serverManagedRoles ...data.RoleName) error {
	return nil
}

func (e EmptyTargetsNotaryRepository) InitializeWithCertificate(rootKeyIDs []string, rootCerts []data.PublicKey, serverManagedRoles ...data.RoleName) error {
	return nil
}
func (e EmptyTargetsNotaryRepository) Publish() error {
	return nil
}

func (e EmptyTargetsNotaryRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	return []*client.TargetWithRole{}, nil
}

func (e EmptyTargetsNotaryRepository) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	return nil, client.ErrNoSuchTarget(name)
}

func (e EmptyTargetsNotaryRepository) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	return nil, client.ErrNoSuchTarget(name)
}

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

func (e EmptyTargetsNotaryRepository) GetDelegationRoles() ([]data.Role, error) {
	return []data.Role{}, nil
}

func (e EmptyTargetsNotaryRepository) RotateKey(role data.RoleName, serverManagesKey bool, keyList []string) error {
	return nil
}

func getLoadedNotaryRepository(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
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

func (l LoadedNotaryRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	filteredTargets := []*client.TargetWithRole{}
	for _, tgt := range loadedTargets {
		if len(roles) == 0 || (len(roles) > 0 && roles[0] == tgt.Role.Name) {
			filteredTargets = append(filteredTargets, &client.TargetWithRole{Target: tgt.Target, Role: tgt.Role.Name})
		}
	}
	return filteredTargets, nil
}

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

func (l LoadedNotaryRepository) GetGUN() data.GUN {
	return data.GUN("signed-repo")
}

func (l LoadedNotaryRepository) GetDelegationRoles() ([]data.Role, error) {
	return loadedDelegationRoles, nil
}

func (l LoadedNotaryRepository) GetCryptoService() signed.CryptoService {
	if l.statefulCryptoService == nil {
		// give it an in-memory cryptoservice with a root key and targets key
		l.statefulCryptoService = cryptoservice.NewCryptoService(trustmanager.NewKeyMemoryStore(passphrase.ConstantRetriever("password")))
		l.statefulCryptoService.AddKey(data.CanonicalRootRole, l.GetGUN(), nil)
		l.statefulCryptoService.AddKey(data.CanonicalTargetsRole, l.GetGUN(), nil)
	}
	return l.statefulCryptoService
}

func getLoadedWithNoSignersNotaryRepository(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (client.Repository, error) {
	return LoadedWithNoSignersNotaryRepository{}, nil
}

// LoadedWithNoSignersNotaryRepository is a mock Notary repository that is loaded with targets but no delegations
// it only contains the green target
type LoadedWithNoSignersNotaryRepository struct {
	LoadedNotaryRepository
}

func (l LoadedWithNoSignersNotaryRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	filteredTargets := []*client.TargetWithRole{}
	for _, tgt := range loadedTargets {
		if len(roles) == 0 || (len(roles) > 0 && roles[0] == tgt.Role.Name) {
			filteredTargets = append(filteredTargets, &client.TargetWithRole{Target: tgt.Target, Role: tgt.Role.Name})
		}
	}
	return filteredTargets, nil
}

func (l LoadedWithNoSignersNotaryRepository) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	if name == "" || name == loadedGreenTarget.Name {
		return &client.TargetWithRole{Target: loadedGreenTarget, Role: data.CanonicalTargetsRole}, nil
	}
	return nil, client.ErrNoSuchTarget(name)
}

func (l LoadedWithNoSignersNotaryRepository) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	if name == "" || name == loadedGreenTarget.Name {
		return []client.TargetSignedStruct{{Target: loadedGreenTarget, Role: loadedTargetsRole}}, nil
	}
	return nil, client.ErrNoSuchTarget(name)
}

func (l LoadedWithNoSignersNotaryRepository) GetDelegationRoles() ([]data.Role, error) {
	return []data.Role{}, nil
}
