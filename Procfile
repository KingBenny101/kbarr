core:       air -build.cmd "go build -o tmp/core ./cmd/core" -build.bin "tmp/core" -build.include_ext "go" -build.exclude_dir "web,tmp,vendor" -build.delay "1000"
metadata:   air -build.cmd "go build -o tmp/metadata ./cmd/metadata" -build.bin "tmp/metadata" -build.include_ext "go" -build.exclude_dir "web,tmp,vendor" -build.delay "1000"
indexer:    air -build.cmd "go build -o tmp/indexer ./cmd/indexer" -build.bin "tmp/indexer" -build.include_ext "go" -build.exclude_dir "web,tmp,vendor" -build.delay "1000"
downloader: air -build.cmd "go build -o tmp/downloader ./cmd/downloader" -build.bin "tmp/downloader" -build.include_ext "go" -build.exclude_dir "web,tmp,vendor" -build.delay "1000"
web:        cd web && npm run dev
