variable "VERSION" {
    default = ""
}

variable "USE_GLIBC" {
    default = ""
}

variable "STRIP_TARGET" {
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
        GO_STRIP = STRIP_TARGET
    }
}

target "dynbinary" {
    inherits = ["binary"]
    args = {
        GO_LINKMODE = "dynamic"
    }
}

variable "GROUP_TOTAL" {
    default = "1"
}

variable "GROUP_INDEX" {
    default = "0"
}

function "platforms" {
    params = [USE_GLIBC]
    result = concat(["linux/amd64", "linux/386", "linux/arm64", "linux/arm", "linux/ppc64le", "linux/s390x", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm", "windows/386"], USE_GLIBC!=""?[]:["windows/arm64"])
}

function "glen" {
    params = [platforms, GROUP_TOTAL]
    result = ceil(length(platforms)/GROUP_TOTAL)
}

target "_all_platforms" {
    platforms = slice(platforms(USE_GLIBC), GROUP_INDEX*glen(platforms(USE_GLIBC), GROUP_TOTAL),min(length(platforms(USE_GLIBC)), (GROUP_INDEX+1)*glen(platforms(USE_GLIBC), GROUP_TOTAL)))
}

target "cross" {
    inherits = ["binary", "_all_platforms"]
}

target "dynbinary-cross" {
    inherits = ["dynbinary", "_all_platforms"]
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
