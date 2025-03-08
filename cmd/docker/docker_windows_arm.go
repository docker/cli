//go:build windows && arm

//go:generate goversioninfo -arm=true -o=./winresources/resource.syso -icon=internal/assets/docker.ico -manifest=internal/assets/docker.exe.manifest ./winresources/versioninfo.json

package main

import _ "github.com/docker/cli/cmd/docker/winresources"
