package formatter

import (
	"bytes"
	"testing"

	"github.com/docker/docker/pkg/stringid"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestTrustTag(t *testing.T) {
	digest := stringid.GenerateRandomID()
	trustedTag := "tag"

	var ctx trustTagContext

	cases := []struct {
		trustTagCtx trustTagContext
		expValue    string
		call        func() string
	}{
		{
			trustTagContext{
				s: SignedTagInfo{Name: trustedTag,
					Digest:  digest,
					Signers: nil,
				},
			},
			digest,
			ctx.Digest,
		},
		{
			trustTagContext{
				s: SignedTagInfo{Name: trustedTag,
					Digest:  digest,
					Signers: nil,
				},
			},
			trustedTag,
			ctx.SignedTag,
		},
		// Empty signers makes a row with empty string
		{
			trustTagContext{
				s: SignedTagInfo{Name: trustedTag,
					Digest:  digest,
					Signers: nil,
				},
			},
			"",
			ctx.Signers,
		},
		{
			trustTagContext{
				s: SignedTagInfo{Name: trustedTag,
					Digest:  digest,
					Signers: []string{"alice", "bob", "claire"},
				},
			},
			"alice, bob, claire",
			ctx.Signers,
		},
		// alphabetic signing on Signers
		{
			trustTagContext{
				s: SignedTagInfo{Name: trustedTag,
					Digest:  digest,
					Signers: []string{"claire", "bob", "alice"},
				},
			},
			"alice, bob, claire",
			ctx.Signers,
		},
	}

	for _, c := range cases {
		ctx = c.trustTagCtx
		v := c.call()
		if v != c.expValue {
			t.Fatalf("Expected %s, was %s\n", c.expValue, v)
		}
	}
}

func TestTrustTagContextWrite(t *testing.T) {

	cases := []struct {
		context  Context
		expected string
	}{
		// Errors
		{
			Context{
				Format: "{{InvalidFunction}}",
			},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			Context{
				Format: "{{nil}}",
			},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table Format
		{
			Context{
				Format: NewTrustTagFormat(),
			},
			`SIGNED TAG          DIGEST              SIGNERS
tag1                deadbeef            alice
tag2                aaaaaaaa            alice, bob
tag3                bbbbbbbb            
`,
		},
	}

	for _, testcase := range cases {
		signedTags := []SignedTagInfo{
			{Name: "tag1", Digest: "deadbeef", Signers: []string{"alice"}},
			{Name: "tag2", Digest: "aaaaaaaa", Signers: []string{"alice", "bob"}},
			{Name: "tag3", Digest: "bbbbbbbb", Signers: []string{}},
		}
		out := bytes.NewBufferString("")
		testcase.context.Output = out
		err := TrustTagWrite(testcase.context, signedTags)
		if err != nil {
			assert.Error(t, err, testcase.expected)
		} else {
			assert.Check(t, is.Equal(testcase.expected, out.String()))
		}
	}
}

// With no trust data, the TrustTagWrite will print an empty table:
// it's up to the caller to decide whether or not to print this versus an error
func TestTrustTagContextEmptyWrite(t *testing.T) {

	emptyCase := struct {
		context  Context
		expected string
	}{
		Context{
			Format: NewTrustTagFormat(),
		},
		`SIGNED TAG          DIGEST              SIGNERS
`,
	}

	emptySignedTags := []SignedTagInfo{}
	out := bytes.NewBufferString("")
	emptyCase.context.Output = out
	err := TrustTagWrite(emptyCase.context, emptySignedTags)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(emptyCase.expected, out.String()))
}

func TestSignerInfoContextEmptyWrite(t *testing.T) {
	emptyCase := struct {
		context  Context
		expected string
	}{
		Context{
			Format: NewSignerInfoFormat(),
		},
		`SIGNER              KEYS
`,
	}
	emptySignerInfo := []SignerInfo{}
	out := bytes.NewBufferString("")
	emptyCase.context.Output = out
	err := SignerInfoWrite(emptyCase.context, emptySignerInfo)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(emptyCase.expected, out.String()))
}

func TestSignerInfoContextWrite(t *testing.T) {
	cases := []struct {
		context  Context
		expected string
	}{
		// Errors
		{
			Context{
				Format: "{{InvalidFunction}}",
			},
			`Template parsing error: template: :1: function "InvalidFunction" not defined
`,
		},
		{
			Context{
				Format: "{{nil}}",
			},
			`Template parsing error: template: :1:2: executing "" at <nil>: nil is not a command
`,
		},
		// Table Format
		{
			Context{
				Format: NewSignerInfoFormat(),
				Trunc:  true,
			},
			`SIGNER              KEYS
alice               key11, key12
bob                 key21
eve                 foobarbazqux, key31, key32
`,
		},
		// No truncation
		{
			Context{
				Format: NewSignerInfoFormat(),
			},
			`SIGNER              KEYS
alice               key11, key12
bob                 key21
eve                 foobarbazquxquux, key31, key32
`,
		},
	}

	for _, testcase := range cases {
		signerInfo := SignerInfoList{
			{Name: "alice", Keys: []string{"key11", "key12"}},
			{Name: "bob", Keys: []string{"key21"}},
			{Name: "eve", Keys: []string{"key31", "key32", "foobarbazquxquux"}},
		}
		out := bytes.NewBufferString("")
		testcase.context.Output = out
		err := SignerInfoWrite(testcase.context, signerInfo)
		if err != nil {
			assert.Error(t, err, testcase.expected)
		} else {
			assert.Check(t, is.Equal(testcase.expected, out.String()))
		}
	}
}
