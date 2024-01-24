variable "GO_VERSION" {
    default = "1.20.13"
}
variable "VERSION" {
    default = ""
}
variable "USE_GLIBC" {
    default = ""
}
variable "STRIP_TARGET" {
    default = ""
}
variable "IMAGE_NAME" {
    default = "docker-cli"
}

# Sets the name of the company that produced the windows binary.
variable "PACKAGER_NAME" {
    default = ""
}

target "_common" {
    args = {
        GO_VERSION = GO_VERSION
        BUILDKIT_CONTEXT_KEEP_GIT_DIR = 1
    }
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
    args = {
        BASE_VARIANT = USE_GLIBC == "1" ? "bullseye" : "alpine"
        VERSION = VERSION
        PACKAGER_NAME = PACKAGER_NAME
        GO_STRIP = STRIP_TARGET
    }
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
    args = {
        BASE_VARIANT = USE_GLIBC == "1" ? "bullseye" : "alpine"
        VERSION = VERSION
        GO_STRIP = STRIP_TARGET
    }
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
    args = {
        BASE_VARIANT = USE_GLIBC == "1" ? "bullseye" : "alpine"
        VERSION = VERSION
    }
}

target "e2e-gencerts" {
    inherits = ["_common"]
    dockerfile = "./e2e/testdata/Dockerfile.gencerts"
    output = ["./e2e/testdata"]
}
