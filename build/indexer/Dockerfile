FROM golang:1.26-alpine AS builder

WORKDIR /workspace
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY shared ./shared
COPY cmd/indexer ./cmd/indexer
COPY internal/indexer ./internal/indexer

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/kbarr-indexer ./cmd/indexer

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/kbarr-indexer .
EXPOSE 8082
CMD ["./kbarr-indexer"]
