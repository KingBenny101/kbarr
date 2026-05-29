module github.com/kingbenny101/kbarr/services/prowlarr

go 1.25.7

replace github.com/kingbenny101/kbarr/shared/config => ../../shared/config

replace github.com/kingbenny101/kbarr/shared/logger => ../../shared/logger

replace github.com/kingbenny101/kbarr/shared/proto => ../../shared/proto

require (
	github.com/jackc/pgx/v5 v5.6.0
	github.com/kingbenny101/kbarr/shared/config v0.0.0
	github.com/kingbenny101/kbarr/shared/logger v0.0.0
	github.com/kingbenny101/kbarr/shared/proto v0.0.0
	google.golang.org/grpc v1.80.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/uptrace/bun v1.2.18 // indirect
	github.com/uptrace/bun/dialect/pgdialect v1.2.18 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
