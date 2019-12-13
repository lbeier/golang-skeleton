FROM golang:1.13-alpine AS base
RUN apk add bash ca-certificates git gcc g++ libc-dev
WORKDIR /go/src/github.com/tutabeier/service
COPY go.mod .
COPY go.sum .
RUN go mod download

FROM base AS builder
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY migrations/ migrations/
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/service ./cmd/service

FROM alpine AS service
WORKDIR /go
COPY --from=builder /go/src/github.com/tutabeier/service/migrations/ /go/migrations/
COPY --from=builder /go/src/github.com/tutabeier/service/build/service /go/service
EXPOSE 80
ENTRYPOINT ["/go/service"]