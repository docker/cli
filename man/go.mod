module github.com/docker/cli/man

// dummy go.mod to avoid dealing with dependencies specific
// to manpages generation and not really part of the project.

go 1.16

//require (
//	github.com/docker/cli v0.0.0+incompatible
//	github.com/cpuguy83/go-md2man/v2 v2.0.3
//	github.com/spf13/cobra v1.2.1
//	github.com/spf13/pflag v1.0.5
//)
//
//replace github.com/docker/cli v0.0.0+incompatible => ../
