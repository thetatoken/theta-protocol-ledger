FROM golang:1.21-bullseye

# GNU cross-compiler for x86_64.
# Using Debian Bullseye (glibc 2.31) rather than Bookworm (glibc 2.36) is
# intentional: Bullseye still has libpthread as a separate library, so pthread
# symbols are versioned at GLIBC_2.0/2.1 rather than the GLIBC_2.34 that
# appeared after the libpthread-into-libc merge, making the binary compatible
# with Ubuntu 18.04 (glibc 2.27).
RUN apt-get update && apt-get install -y gcc-x86-64-linux-gnu g++-x86-64-linux-gnu

VOLUME ["/go"]

WORKDIR /go/src/github.com/thetatoken/theta/

ENV GOPATH=/go
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=1
ENV CC=x86_64-linux-gnu-gcc
ENV CGO_LDFLAGS="-lstdc++"
ENV GO111MODULE=on

CMD ["/go/src/github.com/thetatoken/theta/integration/docker/build/start.sh"]
