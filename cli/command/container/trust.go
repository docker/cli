package container

import (
	"context"
	"fmt"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/command"
)

// tagTrusted tags a trusted ref. It is a shallow wrapper around APIClient.ImageTag
// that updates the given image references to their familiar format for printing.
func tagTrusted(ctx context.Context, cli command.Cli, trustedRef reference.Canonical, ref reference.NamedTagged) error {
	_, _ = fmt.Fprintf(cli.Err(), "Tagging %s as %s\n", reference.FamiliarString(trustedRef), reference.FamiliarString(ref))
	return cli.Client().ImageTag(ctx, trustedRef.String(), ref.String())
}
