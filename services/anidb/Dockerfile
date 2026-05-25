FROM golang:1.25-alpine AS builder

WORKDIR /workspace
RUN apk add --no-cache ca-certificates

COPY shared ./shared
COPY services/anidb ./services/anidb

WORKDIR /workspace/services/anidb
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/kbarr-anidb ./cmd

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /out/kbarr-anidb .

EXPOSE 8081
CMD ["./kbarr-anidb"]
