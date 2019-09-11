package trust

import (
	"sort"
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/pkg/stringid"
)

const (
	defaultTrustTagTableFormat   = "table {{.SignedTag}}\t{{.Digest}}\t{{.Signers}}"
	signedTagNameHeader          = "SIGNED TAG"
	trustedDigestHeader          = "DIGEST"
	signersHeader                = "SIGNERS"
	defaultSignerInfoTableFormat = "table {{.Signer}}\t{{.Keys}}"
	signerNameHeader             = "SIGNER"
	keysHeader                   = "KEYS"
)

// SignedTagInfo represents all formatted information needed to describe a signed tag:
// Name: name of the signed tag
// Digest: hex encoded digest of the contents
// Signers: list of entities who signed the tag
type SignedTagInfo struct {
	Name    string
	Digest  string
	Signers []string
}

// SignerInfo represents all formatted information needed to describe a signer:
// Name: name of the signer role
// Keys: the keys associated with the signer
type SignerInfo struct {
	Name string
	Keys []string
}

// NewTrustTagFormat returns a Format for rendering using a trusted tag Context
func NewTrustTagFormat() formatter.Format {
	return defaultTrustTagTableFormat
}

// NewSignerInfoFormat returns a Format for rendering a signer role info Context
func NewSignerInfoFormat() formatter.Format {
	return defaultSignerInfoTableFormat
}

// TagWrite writes the context
func TagWrite(ctx formatter.Context, signedTagInfoList []SignedTagInfo) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, signedTag := range signedTagInfoList {
			if err := format(&trustTagContext{s: signedTag}); err != nil {
				return err
			}
		}
		return nil
	}
	trustTagCtx := trustTagContext{}
	trustTagCtx.Header = formatter.SubHeaderContext{
		"SignedTag": signedTagNameHeader,
		"Digest":    trustedDigestHeader,
		"Signers":   signersHeader,
	}
	return ctx.Write(&trustTagCtx, render)
}

type trustTagContext struct {
	formatter.HeaderContext
	s SignedTagInfo
}

// SignedTag returns the name of the signed tag
func (c *trustTagContext) SignedTag() string {
	return c.s.Name
}

// Digest returns the hex encoded digest associated with this signed tag
func (c *trustTagContext) Digest() string {
	return c.s.Digest
}

// Signers returns the sorted list of entities who signed this tag
func (c *trustTagContext) Signers() string {
	sort.Strings(c.s.Signers)
	return strings.Join(c.s.Signers, ", ")
}

// SignerInfoWrite writes the context
func SignerInfoWrite(ctx formatter.Context, signerInfoList []SignerInfo) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		for _, signerInfo := range signerInfoList {
			if err := format(&signerInfoContext{
				trunc: ctx.Trunc,
				s:     signerInfo,
			}); err != nil {
				return err
			}
		}
		return nil
	}
	signerInfoCtx := signerInfoContext{}
	signerInfoCtx.Header = formatter.SubHeaderContext{
		"Signer": signerNameHeader,
		"Keys":   keysHeader,
	}
	return ctx.Write(&signerInfoCtx, render)
}

type signerInfoContext struct {
	formatter.HeaderContext
	trunc bool
	s     SignerInfo
}

// Keys returns the sorted list of keys associated with the signer
func (c *signerInfoContext) Keys() string {
	sort.Strings(c.s.Keys)
	truncatedKeys := []string{}
	if c.trunc {
		for _, keyID := range c.s.Keys {
			truncatedKeys = append(truncatedKeys, stringid.TruncateID(keyID))
		}
		return strings.Join(truncatedKeys, ", ")
	}
	return strings.Join(c.s.Keys, ", ")
}

// Signer returns the name of the signer
func (c *signerInfoContext) Signer() string {
	return c.s.Name
}
