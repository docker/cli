//go:build windows && arm64

//go:generate goversioninfo -arm=true -64=true -o=./winresources/resource.syso -icon=internal/assets/docker.ico -manifest=internal/assets/docker.exe.manifest ./winresources/versioninfo.json

package main

import _ "github.com/docker/cli/cmd/docker/winresources"
