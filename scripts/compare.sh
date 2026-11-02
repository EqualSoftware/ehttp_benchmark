#!/usr/bin/env bash
# Runs the comparison for one app: N runs of each route, k6 on the host.
#
#   scripts/compare.sh APP            env: RUNS (10) RUN_FROM (1) ROUTES (plaintext json users mw)
#                                          RATE (4000) DURATION (30s) WARMUP (10s)
#                                          RESULTS (results/<timestamp>) COOLDOWN (5)
#
# Each run writes run-N.json (k6 summary) and run-N.stats.csv (docker stats of
# the app during the measured part, about 1 sample per 1.5 s, no warm-up).
set -euo pipefail
cd "$(dirname "$0")/.."
. scripts/lib.sh

app="${1:?usage: compare.sh APP}"
RUNS="${RUNS:-10}"
RUN_FROM="${RUN_FROM:-1}"
ROUTES="${ROUTES:-plaintext json users mw}"
RATE="${RATE:-4000}"
DURATION="${DURATION:-30s}"
WARMUP="${WARMUP:-10s}"
COOLDOWN="${COOLDOWN:-5}"
RESULTS="${RESULTS:-results/$(date -u +%Y%m%dT%H%M%SZ)}"

secs() { case "$1" in *m) echo $(( ${1%m} * 60 ));; *s) echo "${1%s}";; *) echo "$1";; esac; }
warm_s="$(secs "$WARMUP")"
dur_s="$(secs "$DURATION")"

mkdir -p "$RESULTS"
[ -f "$RESULTS/env.json" ] || scripts/env.sh >"$RESULTS/env.json"

# sample CONTAINER FILE SECONDS: one docker stats line per sample, for SECONDS.
sample() {
  echo "cpu_percent,mem_bytes" >"$2"
  end=$((SECONDS + $3 - 2)) # a docker stats call takes about 2 s
  while [ "$SECONDS" -lt "$end" ]; do
    docker stats --no-stream --format '{{.CPUPerc}} {{.MemUsage}}' "$1" 2>/dev/null |
      awk '{ gsub("%", "", $1); v = $2; u = v; gsub(/[0-9.]/, "", u); sub(u, "", v);
             m = (u == "GiB") ? 1073741824 : (u == "MiB") ? 1048576 : (u == "KiB") ? 1024 : 1;
             printf "%s,%.0f\n", $1, v * m }' >>"$2" || true
  done
}

app_down
app_up "$app"
for route in $ROUTES; do
  dir="$RESULTS/$app/$route"
  mkdir -p "$dir"
  for run in $(seq "$RUN_FROM" $((RUN_FROM + RUNS - 1))); do
    echo "== $app $route run $run"
    (sleep "$warm_s"; sample "bench-$app" "$dir/run-$run.stats.csv" "$dur_s") &
    sampler=$!
    k6 run --quiet --no-usage-report \
      -e APP="$app" -e ROUTE="$route" -e RATE="$RATE" -e DURATION="$DURATION" -e WARMUP="$WARMUP" \
      -e BASE_URL="$BASE_URL" -e OUT="$dir/run-$run.json" \
      k6/compare/compare.js | grep '^RESULT' || true
    wait "$sampler"
    sleep "$COOLDOWN"
  done
done
app_down
echo "results: $RESULTS"
