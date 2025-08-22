package trust

import (
	"sort"
	"strings"

	"github.com/docker/cli/cli/command/formatter"
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

// signedTagInfo represents all formatted information needed to describe a signed tag:
// Name: name of the signed tag
// Digest: hex encoded digest of the contents
// Signers: list of entities who signed the tag
type signedTagInfo struct {
	Name    string
	Digest  string
	Signers []string
}

// signerInfo represents all formatted information needed to describe a signer:
// Name: name of the signer role
// Keys: the keys associated with the signer
type signerInfo struct {
	Name string
	Keys []string
}

// tagWrite writes the context
func tagWrite(fmtCtx formatter.Context, signedTagInfoList []signedTagInfo) error {
	trustTagCtx := &trustTagContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"SignedTag": signedTagNameHeader,
				"Digest":    trustedDigestHeader,
				"Signers":   signersHeader,
			},
		},
	}
	return fmtCtx.Write(trustTagCtx, func(format func(subContext formatter.SubContext) error) error {
		for _, signedTag := range signedTagInfoList {
			if err := format(&trustTagContext{s: signedTag}); err != nil {
				return err
			}
		}
		return nil
	})
}

type trustTagContext struct {
	formatter.HeaderContext
	s signedTagInfo
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

// signerInfoWrite writes the context.
func signerInfoWrite(fmtCtx formatter.Context, signerInfoList []signerInfo) error {
	signerInfoCtx := &signerInfoContext{
		HeaderContext: formatter.HeaderContext{
			Header: formatter.SubHeaderContext{
				"Signer": signerNameHeader,
				"Keys":   keysHeader,
			},
		},
	}
	return fmtCtx.Write(signerInfoCtx, func(format func(subContext formatter.SubContext) error) error {
		for _, info := range signerInfoList {
			if err := format(&signerInfoContext{
				trunc: fmtCtx.Trunc,
				s:     info,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

type signerInfoContext struct {
	formatter.HeaderContext
	trunc bool
	s     signerInfo
}

// Keys returns the sorted list of keys associated with the signer
func (c *signerInfoContext) Keys() string {
	sort.Strings(c.s.Keys)
	truncatedKeys := []string{}
	if c.trunc {
		for _, keyID := range c.s.Keys {
			truncatedKeys = append(truncatedKeys, formatter.TruncateID(keyID))
		}
		return strings.Join(truncatedKeys, ", ")
	}
	return strings.Join(c.s.Keys, ", ")
}

// Signer returns the name of the signer
func (c *signerInfoContext) Signer() string {
	return c.s.Name
}
