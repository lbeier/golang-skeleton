FROM golang:1.12 AS builder

WORKDIR /go/src/github.com/tutabeier/golang-skeleton
COPY . ./

ENV GO111MODULE=on

RUN go mod tidy
RUN go build -o build/golang-skeleton -ldflags="-s -w" github.com/tutabeier/golang-skeleton/cmd/golang-skeleton

FROM ubuntu:bionic

COPY --from=builder /go/src/github.com/tutabeier/golang-skeleton/build/golang-skeleton /

EXPOSE 80
ENTRYPOINT ["/golang-skeleton"]