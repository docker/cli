# trust key generate

<!---MARKER_GEN_START-->
Generate and load a signing key-pair

### Options

| Name    | Type     | Default | Description                                                 |
|:--------|:---------|:--------|:------------------------------------------------------------|
| `--dir` | `string` |         | Directory to generate key in, defaults to current directory |


<!---MARKER_GEN_END-->

## Description

`docker trust key generate` generates a key-pair to be used with signing,
 and loads the private key into the local Docker trust keystore.

## Examples

### Generate a key-pair

```console
$ docker trust key generate alice

Generating key for alice...
Enter passphrase for new alice key with ID 17acf3c:
Repeat passphrase for new alice key with ID 17acf3c:
Successfully generated and loaded private key. Corresponding public key available: alice.pub
$ ls
alice.pub
```

The private signing key is encrypted by the passphrase and loaded into the Docker trust keystore.
All passphrase requests to sign with the key will be referred to by the provided `NAME`.

The public key component `alice.pub` will be available in the current working directory, and can
be used directly by `docker trust signer add`.

Provide the `--dir` argument to specify a directory to generate the key in:

```console
$ docker trust key generate alice --dir /foo

Generating key for alice...
Enter passphrase for new alice key with ID 17acf3c:
Repeat passphrase for new alice key with ID 17acf3c:
Successfully generated and loaded private key. Corresponding public key available: alice.pub
$ ls /foo
alice.pub
```
