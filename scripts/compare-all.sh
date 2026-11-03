#!/usr/bin/env bash
# Runs the comparison for all apps, interleaved: round 1 runs every app once,
# then round 2, and so on. The order of the apps rotates each round. A slow
# drift of the machine (heat, other load) then does not favor one app.
#
#   scripts/compare-all.sh            env: RUNS (10 rounds), RUN_FROM (1) and the env of compare.sh
#
# RUN_FROM with RESULTS redoes rounds in an existing folder: RUN_FROM=4 RUNS=2
# runs rounds 4 and 5 again, with the same app order as the first time.
set -euo pipefail
cd "$(dirname "$0")/.."
. scripts/lib.sh

RUNS="${RUNS:-10}"
RUN_FROM="${RUN_FROM:-1}"
export RESULTS="${RESULTS:-results/$(date -u +%Y%m%dT%H%M%SZ)}"
read -r -a apps <<<"$APPS"
n=${#apps[@]}

for round in $(seq $((RUN_FROM - 1)) $((RUN_FROM + RUNS - 2))); do
  for i in $(seq 0 $((n - 1))); do
    app="${apps[$(((i + round) % n))]}"
    RUNS=1 RUN_FROM=$((round + 1)) scripts/compare.sh "$app"
  done
done
go run ./cmd/report "$RESULTS"
