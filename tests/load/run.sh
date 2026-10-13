#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."

export PRODUCTS="${PRODUCTS:-100000}" SEED="${SEED:-42}" STOCK="${STOCK:-1000000}"
export DISTRIBUTION="${DISTRIBUTION:-uniform}"
export WORKLOAD="${WORKLOAD:-search}" ACCESS="${ACCESS:-uniform}" RPS="${RPS:-10}"
export BASE_URL="${BASE_URL:-http://127.0.0.1:18082}" K6_NO_USAGE_REPORT=true
export POSTGRES_DB=j_load API_HOST_PORT=18082
export PROMETHEUS_HOST_PORT="${PROMETHEUS_HOST_PORT:-19092}" GRAFANA_HOST_PORT="${GRAFANA_HOST_PORT:-13002}"
LOAD_PROJECT="${LOAD_PROJECT:-j-load-100k-uniform}"
ENV_FILE="${ENV_FILE:-.env.local}"
K6_BIN="${K6_BIN:-k6}"
OUTPUT_DIR="${OUTPUT_DIR:-/tmp/j-load-results/$(date -u +%Y%m%dT%H%M%SZ)}"
if docker compose version >/dev/null 2>&1; then COMPOSE=(docker compose); else COMPOSE=(docker-compose); fi
compose() { "${COMPOSE[@]}" --env-file "$ENV_FILE" -p "$LOAD_PROJECT" "$@"; }
psql_load() { compose exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -p "$DB_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"' sh "$@"; }
command -v "$K6_BIN" >/dev/null
mkdir -p "$OUTPUT_DIR"
curl -fsS "$BASE_URL/health/ready" > "$OUTPUT_DIR/ready.json"
# Save configuration without printing connection credentials.
"$K6_BIN" version > "$OUTPUT_DIR/k6-version.txt"
uname -smr > "$OUTPUT_DIR/host-os.txt"
docker version --format '{{.Server.Version}}' > "$OUTPUT_DIR/docker-version.txt"
docker info --format 'CPUs={{.NCPU}} Memory={{.MemTotal}}' > "$OUTPUT_DIR/docker-resources.txt"
compose ps -q | xargs docker inspect --format '{{.Name}} image={{.Config.Image}} cpu={{.HostConfig.NanoCpus}} memory={{.HostConfig.Memory}}' > "$OUTPUT_DIR/containers-config.txt"

for trial in 1 2 3; do
  prefix="$OUTPUT_DIR/trial-$trial"
  psql_load -v "stock=$STOCK" < tests/load/reset.sql > "$prefix.reset-before-warmup.txt"
  psql_load -v "stock=$STOCK" -v "products=$PRODUCTS" -v successes=0 < tests/load/verify.sql > "$prefix.stock-before.txt"
  psql_load -c ANALYZE > "$prefix.analyze.txt"
  PHASE=warmup RESULT_PREFIX="$prefix.warmup" "$K6_BIN" run --out "json=$prefix.warmup.json" tests/load/average.js > "$prefix.warmup.log" 2>&1
  python3 tests/load/report.py "$prefix.warmup" > "$prefix.warmup-report.txt"
  psql_load -v "stock=$STOCK" < tests/load/reset.sql > "$prefix.reset-before-measurement.txt"
  psql_load --csv -P footer=off < monitoring/statements.sql > "$prefix.statements-before.csv"
  compose stats --format json > "$prefix.container-stats.json" &
  stats_pid=$!
  test_status=0
  PHASE=average RESULT_PREFIX="$prefix" "$K6_BIN" run --out "json=$prefix.json" tests/load/average.js > "$prefix.k6.log" 2>&1 &
  k6_pid=$!
  (while kill -0 "$k6_pid" 2>/dev/null; do
    date -u +%Y-%m-%dT%H:%M:%SZ
    ps -o pid=,%cpu=,rss= -p "$k6_pid" || true
    sleep 1
  done) > "$prefix.k6-resources.txt" &
  resources_pid=$!
  trap 'kill "$stats_pid" "$resources_pid" "$k6_pid" 2>/dev/null || true' EXIT
  wait "$k6_pid" || test_status=$?
  kill "$resources_pid" 2>/dev/null || true
  wait "$resources_pid" 2>/dev/null || true
  kill "$stats_pid" 2>/dev/null || true
  wait "$stats_pid" 2>/dev/null || true
  trap - EXIT
  psql_load --csv -P footer=off < monitoring/statements.sql > "$prefix.statements-after.csv"
  report_status=0
  python3 tests/load/report.py "$prefix" > "$prefix.report.txt" || report_status=$?
  successes=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["successful_decrements_all_phases"])' "$prefix.report.json")
  psql_load -v "stock=$STOCK" -v "products=$PRODUCTS" -v "successes=$successes" < tests/load/verify.sql > "$prefix.stock.txt"
  curl -fsS "$BASE_URL/health/ready" > "$prefix.ready.json"
  if (( test_status != 0 || report_status != 0 )); then echo "Trial $trial failed; inspect $OUTPUT_DIR" >&2; exit 1; fi
  psql_load -v "stock=$STOCK" < tests/load/reset.sql > "$prefix.reset-after.txt"
  echo "Trial $trial passed: $prefix.report.json"
done
