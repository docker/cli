#!/usr/bin/env bash

set -eu

: "${GO_MD2MAN:=go-md2man}"

if ! command -v "$GO_MD2MAN" > /dev/null; then
  (
    set -x
    # note: this installs all tools defined in go.mod/vendor.mod
    GOBIN="$(pwd)/build/tools" go install -mod=vendor -modfile=vendor.mod tool
  )
  GO_MD2MAN="$(pwd)/build/tools/go-md2man"
fi

mkdir -p man/man1
(
  set -x
  go run -mod=vendor -modfile=vendor.mod -tags manpages ./man/generate.go --source "./man/src" --target "./man/man1"
)

(
  cd man
  for FILE in *.md; do
    base="$(basename "$FILE")"
    name="${base%.md}"
    num="${name##*.}"
    if [ -z "$num" ] || [ "$name" = "$num" ]; then
      # skip files that aren't of the format xxxx.N.md (like README.md)
      continue
    fi
    mkdir -p "./man${num}"
    (
      set -x ;
      "$GO_MD2MAN" -in "$FILE" -out "./man${num}/${name}"
    )
  done
)
