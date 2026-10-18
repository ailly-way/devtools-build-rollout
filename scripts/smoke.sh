#!/bin/sh
set -eu

: "${INFRAI_API_KEY:?set INFRAI_API_KEY first}"

go run ./cmd/devtools-service &
service_pid=$!
trap 'kill "$service_pid" 2>/dev/null || true' EXIT
sleep 1

curl --fail-with-body --request POST http://localhost:8080/build-events \
  --header 'Content-Type: application/json' \
  --data '{"build_id":"build-184","developer":"dev-27","status":"failed","repository":"etl-runner"}'
