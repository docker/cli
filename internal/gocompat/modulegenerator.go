//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"golang.org/x/mod/modfile"
)

func main() {
	if err := generateApp(); err != nil {
		log.Fatal(err)
	}
	if err := generateModule(); err != nil {
		log.Fatal(err)
	}
}

func generateApp() error {
	cmd := exec.Command("go", "list", "-find", "-f", `{{- if ne .Name "main"}}{{if .GoFiles}}{{.ImportPath}}{{end}}{{end -}}`, "../../...")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	var pkgs []string
	for _, p := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(p) == "" || strings.Contains(p, "/internal") {
			continue
		}
		pkgs = append(pkgs, p)
	}
	tmpl, err := template.New("main").Parse(appTemplate)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, appContext{Generator: cmd.String(), Packages: pkgs})
	if err != nil {
		return err
	}

	return os.WriteFile("main_test.go", buf.Bytes(), 0o644)
}

func generateModule() error {
	content, err := os.ReadFile("../../go.mod")
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		content = []byte("module github.com/docker/cli\n")
		if err := os.WriteFile("../../go.mod", content, 0o644); err != nil {
			return err
		}
		// Let's be nice, and remove the go.mod if we created it.
		// FIXME(thaJeztah): we need to clean up the go.mod after running the test, but need to know if we created it (or if it was an existing go.mod)
		// defer os.Remove("../../go.mod")
	} else {
		log.Println("WARN: go.mod exists in the repository root!")
		log.Println("WARN: Using your go.mod instead of our generated version -- this may misbehave!")
	}
	mod, err := modfile.Parse("../../go.mod", content, nil)
	if err != nil {
		return err
	}
	if mod.Go != nil && mod.Go.Version != "" {
		return fmt.Errorf("main go.mod must not contain a go version")
	}
	content, err = os.ReadFile("../../vendor.mod")
	if err != nil {
		return err
	}
	mod, err = modfile.Parse("../../vendor.mod", content, nil)
	if err != nil {
		return err
	}
	if err := mod.AddModuleStmt("gocompat"); err != nil {
		return err
	}
	if err := mod.AddReplace("github.com/docker/cli", "", "../../", ""); err != nil {
		return err
	}
	if err := mod.AddGoStmt("1.21"); err != nil {
		return err
	}
	out, err := mod.Format()
	if err != nil {
		return err
	}
	tmpl, err := template.New("mod").Parse(modTemplate)
	if err != nil {
		return err
	}

	gen, _ := os.Executable()

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, appContext{Generator: gen, Dependencies: string(out)})
	if err != nil {
		return err
	}

	return os.WriteFile("go.mod", buf.Bytes(), 0o644)
}

type appContext struct {
	Generator    string
	Packages     []string
	Dependencies string
}

const appTemplate = `// Code generated by "{{ .Generator }}". DO NOT EDIT.

package main_test

import (
	"testing"

	// Import all importable packages, i.e., packages that:
	//
	// - are not applications ("main")
	// - are not internal
	// - and that have non-test go-files
{{- range .Packages }}
 	_ "{{ . }}"
{{- end}}
)

// This file import all "importable" packages, i.e., packages that:
//
// - are not applications ("main")
// - are not internal
// - and that have non-test go-files
//
// We do this to verify that our code can be consumed as a dependency
// in "module mode". When using a dependency that does not have a go.mod
// (i.e.; is not a "module"), go implicitly generates a go.mod. Lacking
// information from the dependency itself, it assumes "go1.16" language
// (see [DefaultGoModVersion]). Starting with Go1.21, go downgrades the
// language version used for such dependencies, which means that any
// language feature used that is not supported by go1.16 results in a
// compile error;
//
//	# github.com/docker/cli/cli/context/store
//	/go/pkg/mod/github.com/docker/cli@v25.0.0-beta.2+incompatible/cli/context/store/storeconfig.go:6:24: predeclared any requires go1.18 or later (-lang was set to go1.16; check go.mod)
//	/go/pkg/mod/github.com/docker/cli@v25.0.0-beta.2+incompatible/cli/context/store/store.go:74:12: predeclared any requires go1.18 or later (-lang was set to go1.16; check go.mod)
// 
// These errors do NOT occur when using GOPATH mode, nor do they occur
// when using "pseudo module mode" (the "-mod=mod -modfile=vendor.mod"
// approach used in this repository).
//
// As a workaround for this situation, we must include "//go:build" comments
// in any file that uses newer go-language features (such as the "any" type
// or the "min()", "max()" builtins).
//
// From the go toolchain docs (https://go.dev/doc/toolchain):
//
// > The go line for each module sets the language version the compiler enforces
// > when compiling packages in that module. The language version can be changed
// > on a per-file basis by using a build constraint.
// > 
// > For example, a module containing code that uses the Go 1.21 language version
// > should have a go.mod file with a go line such as go 1.21 or go 1.21.3.
// > If a specific source file should be compiled only when using a newer Go
// > toolchain, adding //go:build go1.22 to that source file both ensures that
// > only Go 1.22 and newer toolchains will compile the file and also changes
// > the language version in that file to Go 1.22.
//
// This file is a generated module that imports all packages provided in
// the repository, which replicates an external consumer using our code
// as a dependency in go-module mode, and verifies all files in those
// packages have the correct "//go:build <go language version>" set.
//
// [DefaultGoModVersion]: https://github.com/golang/go/blob/58c28ba286dd0e98fe4cca80f5d64bbcb824a685/src/cmd/go/internal/gover/version.go#L15-L24
// [2]: https://go.dev/doc/toolchain
func TestModuleCompatibllity(t *testing.T) {
	t.Log("all packages have the correct go version specified through //go:build")
}
`

const modTemplate = `// Code generated by "{{ .Generator }}". DO NOT EDIT.

{{.Dependencies}}
`
