# Stage 1: Build React frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS backend-builder

ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./internal/api/frontend
RUN CGO_ENABLED=1 GOOS=linux \
    go build \
    -ldflags "-X 'github.com/kingbenny101/kbarr/internal/version.GitCommit=$GIT_COMMIT' -X 'github.com/kingbenny101/kbarr/internal/version.BuildTime=$BUILD_TIME'" \
    -o kbarr .

# Stage 3: Final image
FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=backend-builder /app/kbarr .
EXPOSE 8282
CMD ["./kbarr"]