# trust key load

<!---MARKER_GEN_START-->
Load a private key file for signing

### Options

| Name     | Type     | Default  | Description             |
|:---------|:---------|:---------|:------------------------|
| `--name` | `string` | `signer` | Name for the loaded key |


<!---MARKER_GEN_END-->

## Description

`docker trust key load` adds private keys to the local Docker trust keystore.

To add a signer to a repository use `docker trust signer add`.

## Examples

### Load a single private key

For a private key `alice.pem` with permissions `-rw-------`

```console
$ docker trust key load alice.pem

Loading key from "alice.pem"...
Enter passphrase for new signer key with ID f8097df:
Repeat passphrase for new signer key with ID f8097df:
Successfully imported key from alice.pem
```

To specify a name use the `--name` flag:

```console
$ docker trust key load --name alice-key alice.pem

Loading key from "alice.pem"...
Enter passphrase for new alice-key key with ID f8097df:
Repeat passphrase for new alice-key key with ID f8097df:
Successfully imported key from alice.pem
```
