#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
mkdir -p logs
nohup go run ./cmd/server -c config/config.yaml > logs/server_run.log 2>&1 &
echo $! > logs/server.pid
echo "Server started, PID: $(cat logs/server.pid), log: logs/server_run.log"
