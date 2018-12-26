FROM golang:1.11.2

ENV VERSION=develop

WORKDIR /go/src/github.com/exantech/monero-fastsync

CMD go build -ldflags "-X main.version=$VERSION" -o fsd github.com/exantech/monero-fastsync/cmd/fsd && go build -ldflags "-X main.version=$VERSION" -o syncer github.com/exantech/monero-fastsync/cmd/syncer
