#!/usr/bin/env bash

set -Eeuo pipefail

backend_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
frontend_root="${FRONTEND_DIR:-$backend_root/../stock-quant-console}"
compose_file="$backend_root/build/dev/docker-compose.yml"
backend_address="${HTTP_ADDRESS:-:8357}"
backend_port="${backend_address##*:}"
frontend_port="${FRONTEND_PORT:-4173}"

export APP_ENV="${APP_ENV:-development}"
export HTTP_ADDRESS="$backend_address"
export DB_HOST="${DB_HOST:-127.0.0.1}"
export DB_PORT="${DB_PORT:-3307}"
export MYSQL_HOST_PORT="${MYSQL_HOST_PORT:-$DB_PORT}"
export DB_NAME="${DB_NAME:-stock_quant_dev}"
export DB_USER="${DB_USER:-stock_quant}"
export DB_PASSWORD="${DB_PASSWORD:-stock_quant_dev}"
export VITE_API_PROXY_TARGET="http://127.0.0.1:${backend_port}"

backend_pid=""
frontend_pid=""

cleanup() {
	set +e
	if [[ -n "$backend_pid" ]]; then
		kill "$backend_pid" 2>/dev/null
		wait "$backend_pid" 2>/dev/null
	fi
	if [[ -n "$frontend_pid" ]]; then
		kill "$frontend_pid" 2>/dev/null
		wait "$frontend_pid" 2>/dev/null
	fi
	docker compose -f "$compose_file" stop mysql >/dev/null 2>&1
}
trap cleanup EXIT INT TERM

if [[ ! -d "$frontend_root" ]]; then
	echo "frontend directory not found: $frontend_root" >&2
	exit 1
fi

cd "$backend_root"

docker compose -f "$compose_file" up -d --wait mysql
go run ./cmd/seed

go run ./cmd/server &
backend_pid=$!

for _ in $(seq 1 60); do
	if curl --fail --silent "http://127.0.0.1:${backend_port}/api/v1/health" >/dev/null; then
		break
	fi
	sleep 1
done
if ! curl --fail --silent "http://127.0.0.1:${backend_port}/api/v1/health" >/dev/null; then
	echo "backend health check failed at http://127.0.0.1:${backend_port}/api/v1/health" >&2
	exit 1
fi

(
	cd "$frontend_root"
	npm run dev -- --host 127.0.0.1 --port "$frontend_port"
) &
frontend_pid=$!

for _ in $(seq 1 60); do
	if curl --fail --silent "http://127.0.0.1:${frontend_port}/" >/dev/null; then
		break
	fi
	sleep 1
done
if ! curl --fail --silent "http://127.0.0.1:${frontend_port}/" >/dev/null; then
	echo "frontend health check failed at http://127.0.0.1:${frontend_port}/" >&2
	exit 1
fi

echo "MySQL: 127.0.0.1:${DB_PORT}"
echo "Backend: http://127.0.0.1:${backend_port}"
echo "Frontend: http://127.0.0.1:${frontend_port}"
echo "VITE_API_PROXY_TARGET: ${VITE_API_PROXY_TARGET}"
wait -n "$backend_pid" "$frontend_pid"
