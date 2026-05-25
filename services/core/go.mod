module github.com/kingbenny101/kbarr/services/core

go 1.25.7

replace github.com/kingbenny101/kbarr/shared/logger => ../../shared/logger

replace github.com/kingbenny101/kbarr/shared/models => ../../shared/models

replace github.com/kingbenny101/kbarr/shared/config => ../../shared/config

replace github.com/kingbenny101/kbarr/shared/proto => ../../shared/proto

require (
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-chi/cors v1.2.2
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/jackc/pgx/v5 v5.6.0
	github.com/kingbenny101/kbarr/shared/config v0.0.0
	github.com/kingbenny101/kbarr/shared/logger v0.0.0
	github.com/kingbenny101/kbarr/shared/models v0.0.0
	github.com/kingbenny101/kbarr/shared/proto v0.0.0
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/lib/pq v1.10.9 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
)
