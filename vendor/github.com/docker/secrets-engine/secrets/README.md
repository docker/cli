# secrets

`github.com/docker/secrets-engine/secrets` holds the core types of the Secrets
Engine: secret IDs and patterns, resolved envelopes, and the `Secret` and
`Store` interfaces.

It depends only on the standard library, so clients, plugins and store
backends can import it without pulling in the engine runtime or the keychain
bindings. Its API is stable and follows semantic versioning from `v1.0.0`.

```shell
go get github.com/docker/secrets-engine/secrets
```

## Identifiers and patterns

An `ID` contains a secret identifier. Valid secret identifiers must match the
format `^[A-Za-z0-9._:-]+(?:/[A-Za-z0-9._:-]+)*$`:

- Components separated by `/`
- Each component is non-empty
- Only characters `A-Z`, `a-z`, `0-9`, `.`, `_`, `-` or `:`
- No leading, trailing, or double slashes

For storage, we don't really differentiate much about the ID format but by
convention we do simple, slash-separated management, providing a groupable
access control system for management across plugins.

A `Pattern` matches secret IDs. It follows the ID validation rules, except
that `*` matches one component and `**` matches zero or more.

```go
id := secrets.MustParseID("docker/auth/hub/joe")
pattern := secrets.MustParsePattern("docker/auth/**")
pattern.Match(id) // true

if _, err := secrets.ParsePattern(userInput); errors.Is(err, secrets.ErrInvalidPattern) {
	// reject the input
}
```

## Implementing a secret

Store backends such as [`store/keychain`](../store/keychain) and
[`store/posixage`](../store/posixage) persist any type that implements
`secrets.Secret`, and take a `secrets.Factory` that creates an empty one per
ID:

```go
type token struct {
	value    []byte
	metadata map[string]string
}

func (t *token) Marshal() ([]byte, error) { return t.value, nil }

// Copy the data: backends zero the buffer after Unmarshal returns.
func (t *token) Unmarshal(data []byte) error {
	t.value = bytes.Clone(data)
	return nil
}

func (t *token) Metadata() map[string]string { return t.metadata }

func (t *token) SetMetadata(m map[string]string) error {
	t.metadata = m
	return nil
}

var factory secrets.Factory[*token] = func(context.Context, secrets.ID) *token {
	return &token{}
}
```

## Migrating from `x/secrets` and `store`

`github.com/docker/secrets-engine/x/secrets` and the type aliases in the root
`github.com/docker/secrets-engine/store` package are deprecated aliases of
this module. They carry `//go:fix inline` directives, so `go fix ./...`
rewrites callers to import `secrets` directly. `x/secrets` will be removed in a
future release of `x`.

`x/secrets.Resolver`, `x/secrets.Authorizer` and `x/secrets.AuthorizeResponse`
are the exception: `go fix` leaves them alone, and their replacements live in
`github.com/docker/secrets-engine/client/api/resolver`.

| Deprecated                                                  | Replacement                 |
|-------------------------------------------------------------|-----------------------------|
| `x/secrets.*`                                               | `secrets.*`                 |
| `x/secrets.Resolver`, `x/secrets.Authorizer`, `x/secrets.AuthorizeResponse` | `client/api/resolver.Resolver`, `client/api/resolver.Authorizer`, `client/api/resolver.AuthorizeResponse` |
| `store.ID`, `store.Pattern`, `store.Secret`, `store.Store`  | `secrets.ID`, `secrets.Pattern`, `secrets.Secret`, `secrets.Store` |
| `store.Factory[T]`                                          | `secrets.Factory[T]`        |
| `store.ParseID`, `store.MustParseID`                        | `secrets.ParseID`, `secrets.MustParseID` |
| `store.ParsePattern`, `store.MustParsePattern`              | `secrets.ParsePattern`, `secrets.MustParsePattern` |
| `store.ErrCredentialNotFound`                               | `secrets.ErrNotFound`       |
| `store.ErrEmptySecret`                                      | `secrets.ErrEmptySecret`    |
