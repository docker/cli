variable "GO_VERSION" {
    default = "1.18.0"
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

group "default" {
    targets = ["binary"]
}

target "binary" {
    inherits = ["_common"]
    target = "binary"
    platforms = ["local"]
    output = ["build"]
    args = {
        BASE_VARIANT = USE_GLIBC != "" ? "bullseye" : "alpine"
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
        BASE_VARIANT = USE_GLIBC != "" ? "bullseye" : "alpine"
        VERSION = VERSION
        GO_STRIP = STRIP_TARGET
    }
}

target "platforms" {
    platforms = concat(["linux/amd64", "linux/386", "linux/arm64", "linux/arm", "linux/ppc64le", "linux/s390x", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm", "windows/386"], USE_GLIBC!=""?[]:["windows/arm64"])
}

target "cross" {
    inherits = ["binary", "platforms"]
}

target "dynbinary-cross" {
    inherits = ["dynbinary", "platforms"]
}

target "plugins-cross" {
    inherits = ["plugins", "platforms"]
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
    args = {
        // used to invalidate cache (more info https://github.com/moby/buildkit/issues/1213)
        UUID = uuidv4()
    }
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
        BASE_VARIANT = USE_GLIBC != "" ? "bullseye" : "alpine"
        VERSION = VERSION
    }
}
