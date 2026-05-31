#!/bin/bash
tmux new-session -d -s kbarr
tmux rename-window -t kbarr:0 'db'
tmux send-keys -t kbarr:0 'make db' C-m

tmux new-window -t kbarr -n 'indexer'
tmux send-keys -t kbarr:indexer 'make run-indexer' C-m

tmux new-window -t kbarr -n 'metadata'
tmux send-keys -t kbarr:metadata 'make run-metadata' C-m

tmux new-window -t kbarr -n 'downloader'
tmux send-keys -t kbarr:downloader 'make run-downloader' C-m

tmux new-window -t kbarr -n 'core'
tmux send-keys -t kbarr:core 'make run-core' C-m

tmux new-window -t kbarr -n 'frontend'
tmux send-keys -t kbarr:frontend 'make run-frontend' C-m

tmux attach -t kbarr