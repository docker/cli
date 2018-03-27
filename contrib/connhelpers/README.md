# Connection Helpers

Connection helpers allow connecting to a remote daemon with custom connection method.

## Installation

You need to put the following to `~/.docker/config.json`:
```
{
  "connHelpers": {
    "ssh": "ssh",
    "dind": "dind"
  }
}
```

## docker-connection-ssh

Usage:
```
$ docker -H ssh://[user@]host[:port][socketpath]
```

Requirements:

- public key authentication is configured
- `socat` must be installed on the remote host

## docker-connection-dind

Usage:
```
$ docker -H dind://containername
```
