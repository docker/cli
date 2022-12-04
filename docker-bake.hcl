variable "GO_VERSION" {}
variable "VERSION" {}
variable "USE_GLIBC" {}
variable "STRIP_TARGET" {}
variable "IMAGE_NAME" {
    default = "docker-cli"
}

# Sets the name of the company that produced the windows binary.
variable "PACKAGER_NAME" {}

variable "binary_args" {
    default = {}
}

variable "go_version" {
    default = GO_VERSION != "" ? { GO_VERSION = GO_VERSION } : {}
}

variable "go_strip" {
    default = STRIP_TARGET != "" ? { GO_STRIP = STRIP_TARGET } : {}
}

variable "variant" {
    default = USE_GLIBC != "" ? { BASE_VARIANT = "bullseye" } : {}
}

variable "version" {
    default = VERSION != "" ? { VERSION = VERSION } : {}
}

variable "packager_name" {
    default = PACKAGER_NAME != "" ? { PACKAGER_NAME = PACKAGER_NAME } : {}
}

target "_common" {
    args = merge(go_version, { BUILDKIT_CONTEXT_KEEP_GIT_DIR = 1 })
}

target "_platforms" {
    platforms = [
        "darwin/amd64",
        "darwin/arm64",
        "linux/amd64",
        "linux/arm/v6",
        "linux/arm/v7",
        "linux/arm64",
        "linux/ppc64le",
        "linux/riscv64",
        "linux/s390x",
        "windows/amd64",
        "windows/arm64"
    ]
}

group "default" {
    targets = ["binary"]
}

target "binary" {
    inherits = ["_common"]
    target = "binary"
    platforms = ["local"]
    output = ["build"]
    args = merge(go_strip, variant, version, packager_name)
}

target "dynbinary" {
    inherits = ["binary"]
    args = {
        GO_LINKMODE = "dynamic"
    }
}

target "plugins" {
    inherits = ["_common"]
    target = "plugins"
    platforms = ["local"]
    output = ["build"]
    args = merge(go_strip, variant, version)
}

target "cross" {
    inherits = ["binary", "_platforms"]
}

target "dynbinary-cross" {
    inherits = ["dynbinary", "_platforms"]
}

target "plugins-cross" {
    inherits = ["plugins", "_platforms"]
}

target "lint" {
    inherits = ["_common"]
    dockerfile = "./dockerfiles/Dockerfile.lint"
    target = "lint"
    output = ["type=cacheonly"]
}

target "shellcheck" {
    inherits = ["_common"]
    dockerfile = "./dockerfiles/Dockerfile.shellcheck"
    target = "shellcheck"
    output = ["type=cacheonly"]
}

target "validate-vendor" {
    inherits = ["_common"]
    dockerfile = "./dockerfiles/Dockerfile.vendor"
    target = "validate"
    output = ["type=cacheonly"]
}

target "update-vendor" {
    inherits = ["_common"]
    dockerfile = "./dockerfiles/Dockerfile.vendor"
    target = "update"
    output = ["."]
}

target "mod-outdated" {
    inherits = ["_common"]
    dockerfile = "./dockerfiles/Dockerfile.vendor"
    target = "outdated"
    no-cache-filter = ["outdated"]
    output = ["type=cacheonly"]
}

target "validate-authors" {
    inherits = ["_common"]
    dockerfile = "./dockerfiles/Dockerfile.authors"
    target = "validate"
    output = ["type=cacheonly"]
}

target "update-authors" {
    inherits = ["_common"]
    dockerfile = "./dockerfiles/Dockerfile.authors"
    target = "update"
    output = ["."]
}

target "test" {
    target = "test"
    output = ["type=cacheonly"]
}

target "test-coverage" {
    target = "test-coverage"
    output = ["build/coverage"]
}

target "e2e-image" {
    target = "e2e"
    output = ["type=docker"]
    tags = ["${IMAGE_NAME}"]
    args = merge(go_version, variant, version)
}
