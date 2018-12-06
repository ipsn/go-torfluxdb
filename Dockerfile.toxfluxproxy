# Build the Tor proxy in a stock Go builder container
FROM golang:alpine as builder

RUN apk update && apk add --no-cache git gcc musl-dev linux-headers

WORKDIR /go/src/github.com/ipsn/go-torfluxdb
ADD . .

RUN \
  go get -v ./cmd/torfluxproxy && \
  go install ./cmd/torfluxproxy

# Pull the Tor proxy into a second stage deploy container
FROM alpine:latest

COPY --from=builder /go/bin/torfluxproxy /usr/local/bin/

ENTRYPOINT ["torfluxproxy"]
