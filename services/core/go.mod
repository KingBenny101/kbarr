module github.com/kingbenny101/kbarr/services/core

go 1.25.7

replace github.com/kingbenny101/kbarr/shared/logger => ../../shared/logger

replace github.com/kingbenny101/kbarr/shared/models => ../../shared/models

replace github.com/kingbenny101/kbarr/shared/config => ../../shared/config

replace github.com/kingbenny101/kbarr/shared/proto => ../../shared/proto

require (
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-chi/cors v1.2.2
	github.com/kingbenny101/kbarr/shared/config v0.0.0
	github.com/kingbenny101/kbarr/shared/logger v0.0.0
	github.com/kingbenny101/kbarr/shared/models v0.0.0
	github.com/kingbenny101/kbarr/shared/proto v0.0.0
	github.com/uptrace/bun v1.2.18
	github.com/uptrace/bun/dialect/pgdialect v1.2.18
	github.com/uptrace/bun/driver/pgdriver v1.2.18
	github.com/uptrace/bun/extra/bundebug v1.2.18
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.opentelemetry.io/otel v1.40.0 // indirect
	go.opentelemetry.io/otel/trace v1.40.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	mellium.im/sasl v0.3.2 // indirect
)
