FROM dockercore/golang-cross:1.13.15

RUN apt-get update && apt-get install -y \
	curl \
	clang \
	file \
	libsqlite3-dev \
	patch \
	tar \
	xz-utils \
	python \
	python-pip \
	--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

RUN useradd -ms /bin/bash notary \
    && pip install codecov

ENV GO111MODULE=on

# Locked go cyclo on this commit as newer commits depend on Golang 1.16 io/fs
RUN go get golang.org/x/lint/golint \
    github.com/client9/misspell/cmd/misspell \
    github.com/gordonklaus/ineffassign \
    github.com/securego/gosec/cmd/gosec/... \
    github.com/fzipp/gocyclo@ffe36aa317dcbb421a536de071660261136174dd

ENV GOFLAGS=-mod=vendor \
    NOTARYDIR=/go/src/github.com/theupdateframework/notary

COPY . ${NOTARYDIR}
RUN chmod -R a+rw /go

WORKDIR ${NOTARYDIR}

# Note this cannot use alpine because of the MacOSX Cross SDK: the cctools there uses sys/cdefs.h and that cannot be used in alpine: http://wiki.musl-libc.org/wiki/FAQ#Q:_I.27m_trying_to_compile_something_against_musl_and_I_get_error_messages_about_sys.2Fcdefs.h
