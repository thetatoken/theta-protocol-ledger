FROM golang:1.11

VOLUME ["/go"]

WORKDIR /go/src/github.com/thetatoken/theta/

ENV GOPATH=/go

ENV CGO_ENABLED=1 

ENV GO111MODULE=on

CMD ["/go/src/github.com/thetatoken/theta/integration/docker/build/start.sh"]



