//go:build windows && arm64
// +build windows,arm64

//go:generate goversioninfo -arm=true -64=true -o=../../cli/winresources/resource.syso -icon=winresources/docker.ico -manifest=winresources/docker.exe.manifest ../../cli/winresources/versioninfo.json

package main

import _ "github.com/docker/cli/cli/winresources"
