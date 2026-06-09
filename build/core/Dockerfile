FROM golang:1.26-alpine AS builder

WORKDIR /workspace
RUN apk add --no-cache ca-certificates nodejs npm

COPY go.mod go.sum ./
RUN go mod download

COPY VERSION ./VERSION
COPY internal ./internal
COPY web ./web
COPY cmd/core ./cmd/core

WORKDIR /workspace/web
RUN npm ci
RUN npm run build
RUN mkdir -p /workspace/internal/core/api/frontend && cp -r /workspace/web/dist/. /workspace/internal/core/api/frontend/

WORKDIR /workspace
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/kbarr ./cmd/core

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/kbarr .
COPY VERSION .
EXPOSE 8080
CMD ["./kbarr"]
