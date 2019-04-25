FROM golang:1.12-alpine AS base
RUN apk add bash ca-certificates git gcc g++ libc-dev
WORKDIR /go/src/github.com/tutabeier/golang-skeleton
ENV GO111MODULE=on
COPY go.mod .
COPY go.sum .
RUN go mod download

FROM base AS builder
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go install -a -tags netgo -ldflags '-w -extldflags "-static"' ./cmd/golang-skeleton

FROM alpine AS service
COPY --from=builder /go/bin/golang-skeleton /bin/service
EXPOSE 80
ENTRYPOINT ["/bin/service"]