#!/bin/bash
tmux new-session -d -s kbarr
tmux rename-window -t kbarr:0 'db'
tmux send-keys -t kbarr:0 'make db && make migrate' C-m

tmux new-window -t kbarr -n 'anidb'
tmux send-keys -t kbarr:anidb 'make run-anidb' C-m

tmux new-window -t kbarr -n 'prowlarr'
tmux send-keys -t kbarr:prowlarr 'make run-prowlarr' C-m

tmux new-window -t kbarr -n 'downloader'
tmux send-keys -t kbarr:downloader 'make run-downloader' C-m

tmux new-window -t kbarr -n 'core'
tmux send-keys -t kbarr:core 'make run-core' C-m

tmux new-window -t kbarr -n 'frontend'
tmux send-keys -t kbarr:frontend 'make run-frontend' C-m

tmux attach -t kbarr