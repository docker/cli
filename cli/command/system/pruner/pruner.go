// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.23

// Package pruner registers "prune" functions to be included as part of
// "docker system prune".
package pruner

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
)

// ContentType is an identifier for content that can be pruned.
type ContentType string

// Pre-defined content-types to prune. Additional types can be registered,
// and will be pruned after the list of pre-defined types.
const (
	TypeContainer  ContentType = "container"
	TypeNetwork    ContentType = "network"
	TypeImage      ContentType = "image"
	TypeVolume     ContentType = "volume"
	TypeBuildCache ContentType = "buildcache"
)

// pruneOrder is the order in which ContentType must be pruned. The order
// in which pruning happens is important to make sure that resources are
// released before pruning (e.g., a "container" can use a "network" and
// "volume", so containers must be pruned before networks and volumes).
var pruneOrder = []ContentType{
	TypeContainer,
	TypeNetwork,
	TypeVolume,
	TypeImage,
	TypeBuildCache,
}

// PruneFunc is the signature for prune-functions. It returns details about
// the content pruned;
//
// - spaceReclaimed is the amount of data removed (in bytes).
// - details is arbitrary information about the content pruned.
type PruneFunc func(ctx context.Context, dockerCLI command.Cli, pruneOpts PruneOptions) (spaceReclaimed uint64, details string, _ error)

type PruneOptions struct {
	All    bool
	Filter opts.FilterOpt
}

// registered holds a map of PruneFunc functions registered through [Register].
// It is considered immutable after startup.
var registered map[ContentType]PruneFunc

// Register registers a [PruneFunc] under the given name to be included in
// "docker system prune". It is designed to be called in an init function
// and is not safe for concurrent use.
//
// For example:
//
//	 func init() {
//		// Register the prune command to run as part of "docker system prune".
//		if err := prune.Register(prune.TypeImage, prunerFn); err != nil {
//			panic(err)
//		}
//	}
func Register(name ContentType, pruneFunc PruneFunc) error {
	if name == "" {
		return errors.New("error registering pruner: invalid prune type: cannot be empty")
	}
	if pruneFunc == nil {
		return errors.New("error registering pruner: prune function is nil for " + string(name))
	}
	if registered == nil {
		registered = make(map[ContentType]PruneFunc)
	}
	if _, exists := registered[name]; exists {
		return fmt.Errorf("error registering pruner: content-type %s is already registered", name)
	}
	registered[name] = pruneFunc
	return nil
}

// List iterates over all registered pruners, starting with known pruners
// in their predefined order, followed by any others (sorted alphabetically).
func List() iter.Seq2[ContentType, PruneFunc] {
	all := maps.Clone(registered)
	ordered := make([]ContentType, 0, len(all))
	for _, ct := range pruneOrder {
		if _, ok := all[ct]; ok {
			ordered = append(ordered, ct)
			delete(all, ct)
		}
	}
	// append any remaining content-types (if any) that may be registered.
	if len(all) > 0 {
		ordered = append(ordered, slices.Sorted(maps.Keys(all))...)
	}

	return func(yield func(ContentType, PruneFunc) bool) {
		for _, ct := range ordered {
			if fn := registered[ct]; fn != nil {
				if !yield(ct, fn) {
					return
				}
			}
		}
	}
}
