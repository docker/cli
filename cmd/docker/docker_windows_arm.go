//go:build windows && arm
// +build windows,arm

//go:generate goversioninfo -arm=true -o=../../cli/winresources/resource.syso -icon=winresources/docker.ico -manifest=winresources/docker.exe.manifest ../../cli/winresources/versioninfo.json

package main

import _ "github.com/docker/cli/cli/winresources"
