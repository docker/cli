package trust

import (
	"context"
	"fmt"
	"io"

	"github.com/distribution/reference"
	"github.com/docker/docker/client"
)

type APIClientProvider interface {
	Client() client.APIClient
}

// TagTrusted tags a trusted ref. It is a shallow wrapper around [client.Client.ImageTag]
// that updates the given image references to their familiar format for tagging
// and printing.
func TagTrusted(ctx context.Context, cli APIClientProvider, out io.Writer, trustedRef reference.Canonical, ref reference.NamedTagged) error {
	// Use familiar references when interacting with client and output
	_, _ = fmt.Fprintf(out, "Tagging %s as %s\n", reference.FamiliarString(trustedRef), reference.FamiliarString(ref))
	return cli.Client().ImageTag(ctx, trustedRef.String(), ref.String())
}
