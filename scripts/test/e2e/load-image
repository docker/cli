#!/usr/bin/env bash
# Fetch images used for e2e testing
set -eu -o pipefail

alpine_src=alpine@sha256:f006ecbb824d87947d0b51ab8488634bf69fe4094959d935c0c103f4820a417d
alpine_dest=registry:5000/alpine:3.6

busybox_src=busybox@sha256:3e8fa85ddfef1af9ca85a5cfb714148956984e02f00bec3f7f49d3925a91e0e7
busybox_dest=registry:5000/busybox:1.27.2

fetch_tag_image() {
    docker pull "$1"
    docker tag "$1" "$2"
}

push_image() {
  docker push "$1"
}

cmd=${1-}
case "$cmd" in
    alpine)
        fetch_tag_image "$alpine_src" "$alpine_dest"
        push_image "$alpine_dest"
        exit
        ;;
    busybox)
        fetch_tag_image "$busybox_src" "$busybox_dest"
        push_image "$busybox_dest"
        exit
        ;;
    all|"")
        fetch_tag_image "$alpine_src" "$alpine_dest"
        push_image "$alpine_dest"
        fetch_tag_image "$busybox_src" "$busybox_dest"
        push_image "$busybox_dest"
        exit
        ;;
    fetch-only)
        fetch_tag_image "$alpine_src" "$alpine_dest"
        fetch_tag_image "$busybox_src" "$busybox_dest"
        exit
        ;;
    *)
        echo "Unknown command: $cmd"
        echo "Usage:"
        echo "    $0 [alpine | busybox | all | fetch-only]"
        exit 1
        ;;
esac
