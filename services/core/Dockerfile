FROM golang:1.25-alpine AS builder

WORKDIR /workspace
RUN apk add --no-cache ca-certificates nodejs npm

COPY VERSION ./VERSION
COPY shared ./shared
COPY frontend ./frontend
COPY services/core ./services/core

WORKDIR /workspace/frontend
RUN npm ci
RUN npm run build
RUN mkdir -p /workspace/services/core/internal/api/frontend /workspace/services/core/frontend && cp -r /workspace/frontend/dist/. /workspace/services/core/internal/api/frontend/ && cp -r /workspace/frontend/dist/. /workspace/services/core/frontend/

WORKDIR /workspace/services/core
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/kbarr ./cmd

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/kbarr .
COPY VERSION .
EXPOSE 8080
CMD ["./kbarr"]
