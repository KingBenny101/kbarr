FROM golang:1.26-alpine AS builder

WORKDIR /workspace
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY internal ./internal
COPY cmd/downloader ./cmd/downloader

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/kbarr-downloader ./cmd/downloader

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/kbarr-downloader .
EXPOSE 8083
CMD ["./kbarr-downloader"]
