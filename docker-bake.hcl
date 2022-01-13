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
variable "COMPANY_NAME" {
    default = ""
}

group "default" {
    targets = ["binary"]
}

target "binary" {
    target = "binary"
    platforms = ["local"]
    output = ["build"]
    args = {
        BASE_VARIANT = USE_GLIBC != "" ? "buster" : "alpine"
        VERSION = VERSION
        COMPANY_NAME = COMPANY_NAME
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
    target = "plugins"
    platforms = ["local"]
    output = ["build"]
    args = {
        BASE_VARIANT = USE_GLIBC != "" ? "buster" : "alpine"
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
    dockerfile = "./dockerfiles/Dockerfile.lint"
    target = "lint"
    output = ["type=cacheonly"]
}

target "shellcheck" {
    dockerfile = "./dockerfiles/Dockerfile.shellcheck"
    target = "shellcheck"
    output = ["type=cacheonly"]
}

target "validate-vendor" {
    dockerfile = "./dockerfiles/Dockerfile.vendor"
    target = "validate"
    output = ["type=cacheonly"]
}

target "update-vendor" {
    dockerfile = "./dockerfiles/Dockerfile.vendor"
    target = "update"
    output = ["."]
}

target "mod-outdated" {
    dockerfile = "./dockerfiles/Dockerfile.vendor"
    target = "outdated"
    args = {
        // used to invalidate cache (more info https://github.com/moby/buildkit/issues/1213)
        UUID = uuidv4()
    }
    output = ["type=cacheonly"]
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
        BASE_VARIANT = USE_GLIBC != "" ? "buster" : "alpine"
        VERSION = VERSION
    }
}
