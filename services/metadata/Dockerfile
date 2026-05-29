FROM golang:1.25-alpine AS builder

WORKDIR /workspace
RUN apk add --no-cache ca-certificates

COPY shared ./shared
COPY services/metadata ./services/metadata

WORKDIR /workspace/services/metadata
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/kbarr-metadata ./cmd

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /out/kbarr-metadata .

EXPOSE 8081
CMD ["./kbarr-metadata"]