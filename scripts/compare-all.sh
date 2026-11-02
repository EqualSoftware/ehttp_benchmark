#!/usr/bin/env bash
# Runs the comparison for all apps, interleaved: round 1 runs every app once,
# then round 2, and so on. The order of the apps rotates each round. A slow
# drift of the machine (heat, other load) then does not favor one app.
#
#   scripts/compare-all.sh            env: RUNS (10 rounds) and the env of compare.sh
set -euo pipefail
cd "$(dirname "$0")/.."
. scripts/lib.sh

RUNS="${RUNS:-10}"
export RESULTS="${RESULTS:-results/$(date -u +%Y%m%dT%H%M%SZ)}"
read -r -a apps <<<"$APPS"
n=${#apps[@]}

for round in $(seq 0 $((RUNS - 1))); do
  for i in $(seq 0 $((n - 1))); do
    app="${apps[$(((i + round) % n))]}"
    RUNS=1 RUN_FROM=$((round + 1)) scripts/compare.sh "$app"
  done
done
go run ./cmd/report "$RESULTS"
