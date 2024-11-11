package image

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/docker/cli/cli/version"
)

// FetchTokenResponse is response from fetching token with GET request
type FetchTokenResponse struct {
	Token        string    `json:"token"`
	AccessToken  string    `json:"access_token"`
	ExpiresIn    int       `json:"expires_in"`
	IssuedAt     time.Time `json:"issued_at"`
	RefreshToken string    `json:"refresh_token"`
}

type TokenOptions struct {
	Realm    string
	Service  string
	Scopes   []string
	Username string
	Secret   string

	// FetchRefreshToken enables fetching a refresh token (aka "identity token", "offline token") along with the bearer token.
	//
	// For HTTP GET mode (FetchToken), FetchRefreshToken sets `offline_token=true` in the request.
	// https://docs.docker.com/registry/spec/auth/token/#requesting-a-token
	//
	// For HTTP POST mode (FetchTokenWithOAuth), FetchRefreshToken sets `access_type=offline` in the request.
	// https://docs.docker.com/registry/spec/auth/oauth/#getting-a-token
	FetchRefreshToken bool
}

func FetchToken(ctx context.Context, client *http.Client, headers http.Header, to TokenOptions) (*FetchTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, to.Realm, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header[k] = append(req.Header[k], v...)
	}
	if len(req.Header.Get("User-Agent")) == 0 {
		req.Header.Set("User-Agent", "containerd/"+version.Version)
	}

	reqParams := req.URL.Query()

	if to.Service != "" {
		reqParams.Add("service", to.Service)
	}

	for _, scope := range to.Scopes {
		reqParams.Add("scope", scope)
	}

	if to.Secret != "" {
		req.SetBasicAuth(to.Username, to.Secret)
	}

	if to.FetchRefreshToken {
		reqParams.Add("offline_token", "true")
	}

	req.URL.RawQuery = reqParams.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, nil
	}

	decoder := json.NewDecoder(resp.Body)

	var tr FetchTokenResponse
	if err = decoder.Decode(&tr); err != nil {
		return nil, fmt.Errorf("unable to decode token response: %w", err)
	}

	// `access_token` is equivalent to `token` and if both are specified
	// the choice is undefined.  Canonicalize `access_token` by sticking
	// things in `token`.
	if tr.AccessToken != "" {
		tr.Token = tr.AccessToken
	}

	if tr.Token == "" {
		return nil, nil
	}

	return &tr, nil
}

func parseValueAndParams(header string) (value string, params map[string]string) {
	params = make(map[string]string)
	value, s := expectToken(header)
	if value == "" {
		return
	}
	value = strings.ToLower(value)
	for {
		var pkey string
		pkey, s = expectToken(skipSpace(s))
		if pkey == "" {
			return
		}
		if !strings.HasPrefix(s, "=") {
			return
		}
		var pvalue string
		pvalue, s = expectTokenOrQuoted(s[1:])
		pkey = strings.ToLower(pkey)
		params[pkey] = pvalue
		s = skipSpace(s)
		if !strings.HasPrefix(s, ",") {
			return
		}
		s = s[1:]
	}
}

// Octet types from RFC 2616.
type octetType byte

var octetTypes [256]octetType

const (
	isToken octetType = 1 << iota
	isSpace
)

func init() {
	// OCTET      = <any 8-bit sequence of data>
	// CHAR       = <any US-ASCII character (octets 0 - 127)>
	// CTL        = <any US-ASCII control character (octets 0 - 31) and DEL (127)>
	// CR         = <US-ASCII CR, carriage return (13)>
	// LF         = <US-ASCII LF, linefeed (10)>
	// SP         = <US-ASCII SP, space (32)>
	// HT         = <US-ASCII HT, horizontal-tab (9)>
	// <">        = <US-ASCII double-quote mark (34)>
	// CRLF       = CR LF
	// LWS        = [CRLF] 1*( SP | HT )
	// TEXT       = <any OCTET except CTLs, but including LWS>
	// separators = "(" | ")" | "<" | ">" | "@" | "," | ";" | ":" | "\" | <">
	//              | "/" | "[" | "]" | "?" | "=" | "{" | "}" | SP | HT
	// token      = 1*<any CHAR except CTLs or separators>
	// qdtext     = <any TEXT except <">>

	for c := 0; c < 256; c++ {
		var t octetType
		isCtl := c <= 31 || c == 127
		isChar := 0 <= c && c <= 127
		isSeparator := strings.ContainsRune(" \t\"(),/:;<=>?@[]\\{}", rune(c))
		if strings.ContainsRune(" \t\r\n", rune(c)) {
			t |= isSpace
		}
		if isChar && !isCtl && !isSeparator {
			t |= isToken
		}
		octetTypes[c] = t
	}
}

func skipSpace(s string) (rest string) {
	i := 0
	for ; i < len(s); i++ {
		if octetTypes[s[i]]&isSpace == 0 {
			break
		}
	}
	return s[i:]
}

func expectToken(s string) (token, rest string) {
	i := 0
	for ; i < len(s); i++ {
		if octetTypes[s[i]]&isToken == 0 {
			break
		}
	}
	return s[:i], s[i:]
}

func expectTokenOrQuoted(s string) (value string, rest string) {
	if !strings.HasPrefix(s, "\"") {
		return expectToken(s)
	}
	s = s[1:]
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			return s[:i], s[i+1:]
		case '\\':
			p := make([]byte, len(s)-1)
			j := copy(p, s[:i])
			escape := true
			for i = i + 1; i < len(s); i++ {
				b := s[i]
				switch {
				case escape:
					escape = false
					p[j] = b
					j++
				case b == '\\':
					escape = true
				case b == '"':
					return string(p[:j]), s[i+1:]
				default:
					p[j] = b
					j++
				}
			}
			return "", ""
		}
	}
	return "", ""
}
