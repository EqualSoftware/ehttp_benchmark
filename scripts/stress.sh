#!/usr/bin/env bash
# Runs one stress test against one benchmark app, with docker stats sampling.
#
#   scripts/stress.sh TYPE            TYPE: stress | breakpoint | spike | soak
#   env: APP (ehttp), RESULTS (results/stress-<timestamp>), BASE_URL, and the
#        env of the k6 script (see the header of k6/stress/TYPE.js)
#
# Starts the app (same cpuset, CPU and memory limits as the comparison), runs
# k6 on the host, and samples `docker stats` until k6 ends. It writes
# TYPE-APP.stats.csv and TYPE-APP.log in RESULTS. Then it stops the app.
set -euo pipefail
cd "$(dirname "$0")/.."
. scripts/lib.sh

type="${1:?usage: stress.sh stress|breakpoint|spike|soak}"
APP="${APP:-ehttp}"
script="k6/stress/$type.js"
[ -f "$script" ] || { echo "no such test: $type" >&2; exit 2; }

RESULTS="${RESULTS:-results/stress-$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$RESULTS"
[ -f "$RESULTS/env.json" ] || scripts/env.sh >"$RESULTS/env.json"
stats="$RESULTS/$type-$APP.stats.csv"

trap 'app_down' EXIT
app_down
app_up "$APP"

k6 run --no-usage-report -e BASE_URL="$BASE_URL" "$script" 2>&1 | tee "$RESULTS/$type-$APP.log" &
k6pid=$!

echo "cpu_percent,mem_bytes" >"$stats"
while kill -0 "$k6pid" 2>/dev/null; do
  docker stats --no-stream --format '{{.CPUPerc}} {{.MemUsage}}' "bench-$APP" 2>/dev/null |
    awk '{ gsub("%", "", $1); v = $2; u = v; gsub(/[0-9.]/, "", u); sub(u, "", v);
           m = (u == "GiB") ? 1073741824 : (u == "MiB") ? 1048576 : (u == "KiB") ? 1024 : 1;
           printf "%s,%.0f\n", $1, v * m }' >>"$stats" || true
done
wait "$k6pid" || status=$?

echo "app CPU % (of 1 CPU): mean / max, memory MiB: max"
awk -F, 'NR > 1 { c += $1; n++; if ($1 > cm) cm = $1; if ($2 > mm) mm = $2 }
         END { if (n) printf "%.0f / %.0f, %.1f (%d samples)\n", c / n, cm, mm / 1048576, n }' "$stats"
echo "results: $RESULTS"
exit "${status:-0}"
