FROM golang:1.25-alpine AS builder

WORKDIR /workspace
RUN apk add --no-cache ca-certificates

COPY shared ./shared
COPY services/indexer ./services/indexer

WORKDIR /workspace/services/indexer
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/kbarr-indexer ./cmd

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates python3 py3-pip && \
    pip3 install --no-cache-dir guessit --break-system-packages

COPY --from=builder /out/kbarr-indexer .

EXPOSE 8082
CMD ["./kbarr-indexer"]