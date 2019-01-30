FROM golang:1.11.2

ENV VERSION=develop

WORKDIR /go/src/github.com/exantech/monero-fastsync

ENV GO111MODULE=on

RUN mkdir -p /.cache/go-build
RUN chmod -R 777 /.cache

CMD go build -ldflags "-X main.version=$VERSION" -o fsd ./cmd/fsd && go build -ldflags "-X main.version=$VERSION" -o syncer ./cmd/syncer
