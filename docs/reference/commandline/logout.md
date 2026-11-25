# logout

<!---MARKER_GEN_START-->
Log out from a registry.
If no server is specified, log out of Docker Hub.


<!---MARKER_GEN_END-->

## Description

With no argument, `docker logout` removes credentials for Docker Hub
(`https://index.docker.io/v1/`). It does not consult the daemon for a
default registry. Pass a server to log out of a private registry.

## Examples

```console
$ docker logout
$ docker logout localhost:8080
```

## Related commands

* [login](login.md)
