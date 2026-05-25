package proto

//go:generate protoc --go_out=. --go_opt=paths=source_relative "--go_opt=Mdownloader.proto=github.com/kingbenny101/kbarr/shared/proto;proto" --go-grpc_out=. --go-grpc_opt=paths=source_relative "--go-grpc_opt=Mdownloader.proto=github.com/kingbenny101/kbarr/shared/proto;proto" anidb.proto prowlarr.proto downloader.proto
