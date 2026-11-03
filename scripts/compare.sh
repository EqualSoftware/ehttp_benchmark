#!/usr/bin/env bash
# Runs the comparison for one app: N runs of each route, k6 on the host.
#
#   scripts/compare.sh APP            env: RUNS (10) RUN_FROM (1) ROUTES (plaintext json users mw)
#                                          RATE (4000) DURATION (30s) WARMUP (10s)
#                                          RESULTS (results/<timestamp>) COOLDOWN (5)
#                                          RETRIES (0) LOG_LINES (2000)
#
# Each run writes run-N.json (k6 summary), run-N.stats.csv (docker stats of
# the app during the measured part, about 1 sample per 1.5 s, no warm-up),
# run-N.log (k6 output, first LOG_LINES lines) and run-N.container.json (state
# of the app container after the run). At the end, container-N.log keeps the
# app log of the container that served the runs from N on.
#
# A run is dirty when it has errors or dropped iterations, or the container is
# not running after it. With RETRIES > 0, a dirty run is moved to retried/ and
# run again, up to RETRIES times. The report counts the retried runs.
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
RETRIES="${RETRIES:-0}"
LOG_LINES="${LOG_LINES:-2000}"
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

# measure DIR RUN: one k6 run with docker stats sampling.
measure() {
  local dir="$1" run="$2" sampler
  (sleep "$warm_s"; sample "bench-$app" "$dir/run-$run.stats.csv" "$dur_s") &
  sampler=$!
  k6 run --quiet --no-usage-report \
    -e APP="$app" -e ROUTE="$route" -e RATE="$RATE" -e DURATION="$DURATION" -e WARMUP="$WARMUP" \
    -e BASE_URL="$BASE_URL" -e OUT="$dir/run-$run.json" \
    k6/compare/compare.js 2>&1 |
    awk -v out="$dir/run-$run.log" -v max="$LOG_LINES" \
      '{ if (NR <= max) print > out; else extra++ } /^RESULT/ { print }
       END { if (extra) printf "... %d more lines\n", extra > out }' || true
  wait "$sampler"
  docker inspect "bench-$app" 2>/dev/null |
    jq '.[0] | { status: .State.Status, oom_killed: .State.OOMKilled, exit_code: .State.ExitCode,
                 restart_count: .RestartCount, started_at: .State.StartedAt }' \
      >"$dir/run-$run.container.json" ||
    echo '{"status": "missing"}' >"$dir/run-$run.container.json"
}

# clean DIR RUN: true if the run has no errors, no dropped iterations and the container still runs.
clean() {
  jq -e '.error_rate == 0 and .dropped == 0' "$1/run-$2.json" >/dev/null 2>&1 &&
    jq -e '.status == "running"' "$1/run-$2.container.json" >/dev/null 2>&1
}

app_down
app_up "$app"
for route in $ROUTES; do
  dir="$RESULTS/$app/$route"
  mkdir -p "$dir"
  for run in $(seq "$RUN_FROM" $((RUN_FROM + RUNS - 1))); do
    attempt=0
    while :; do
      label=""
      if [ "$attempt" -gt 0 ]; then label=" (retry $attempt)"; fi
      echo "== $app $route run $run$label"
      measure "$dir" "$run"
      if clean "$dir" "$run"; then break; fi
      echo "   dirty run: $dir/run-$run"
      [ "$attempt" -lt "$RETRIES" ] || break
      attempt=$((attempt + 1))
      mkdir -p "$dir/retried"
      for f in "$dir/run-$run".*; do mv "$f" "$dir/retried/$(basename "$f" | sed "s/^run-$run\./run-$run.try-$attempt./")"; done
      sleep "$COOLDOWN"
    done
    sleep "$COOLDOWN"
  done
done
docker logs "bench-$app" >"$RESULTS/$app/container-$RUN_FROM.log" 2>&1 || true
app_down
echo "results: $RESULTS"
