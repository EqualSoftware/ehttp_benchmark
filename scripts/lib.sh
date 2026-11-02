# Shared shell helpers. Source this file.
APPS="${APPS:-ehttp nethttp gin echo fiber}"
BENCH_PORT="${BENCH_PORT:-8090}"
export BENCH_PORT
BASE_URL="${BASE_URL:-http://127.0.0.1:${BENCH_PORT}}"

profiles() { for a in $APPS; do printf -- '--profile %s ' "$a"; done; }

# app_up APP: start one app and wait until it answers.
app_up() {
  docker compose --profile "$1" up -d >/dev/null 2>&1
  for _ in $(seq 1 100); do
    curl -fs -o /dev/null "$BASE_URL/plaintext" && return 0
    sleep 0.2
  done
  echo "app $1 did not start" >&2
  return 1
}

# app_down: stop and remove all benchmark containers (images stay).
app_down() { docker compose $(profiles) down --remove-orphans >/dev/null 2>&1; }
