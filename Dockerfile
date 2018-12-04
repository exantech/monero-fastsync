FROM golang:1.11.2-alpine

WORKDIR /go/src/github.com/exantech/monero-fastsync

CMD go build -o fsd github.com/exantech/monero-fastsync/cmd/fsd && go build -o syncer github.com/exantech/monero-fastsync/cmd/syncer