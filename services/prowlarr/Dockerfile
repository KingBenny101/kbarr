FROM golang:1.25-alpine AS builder

WORKDIR /workspace
RUN apk add --no-cache ca-certificates

COPY shared ./shared
COPY services/prowlarr ./services/prowlarr

WORKDIR /workspace/services/prowlarr
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/kbarr-prowlarr ./cmd

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /out/kbarr-prowlarr .

EXPOSE 8082
CMD ["./kbarr-prowlarr"]
