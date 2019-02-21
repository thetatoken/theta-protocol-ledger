FROM golang:1.11

VOLUME ["/go"]

WORKDIR /go/src/github.com/thetatoken/theta/

ENV GOPATH=/go

ENV CGO_ENABLED=1 

CMD ["/go/src/github.com/thetatoken/theta/integration/build/start.sh"]



