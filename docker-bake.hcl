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

target "_all_platforms" {
    platforms = concat(["linux/amd64", "linux/386", "linux/arm64", "linux/arm", "linux/ppc64le", "linux/s390x", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm", "windows/386"], USE_GLIBC!=""?[]:["windows/arm64"])
}

target "cross" {
    inherits = ["binary", "_all_platforms"]
}

target "dynbinary-cross" {
    inherits = ["dynbinary", "_all_platforms"]
}
