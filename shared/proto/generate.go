package proto

//go:generate protoc --go_out=./metadata --go_opt=paths=source_relative --go-grpc_out=./metadata --go-grpc_opt=paths=source_relative metadata.proto
//go:generate protoc --go_out=./indexer --go_opt=paths=source_relative --go-grpc_out=./indexer --go-grpc_opt=paths=source_relative indexer.proto
//go:generate protoc --go_out=./downloader --go_opt=paths=source_relative --go-grpc_out=./downloader --go-grpc_opt=paths=source_relative downloader.proto
