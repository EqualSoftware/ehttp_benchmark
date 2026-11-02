# Results

Each run writes a folder `results/<UTC timestamp>/` (git ignores it):

```
env.json                      machine, Docker VM, limits, versions, ehttp commit
<app>/<route>/run-N.json      k6 summary of the measured phase (rps, p50, p95, p99, errors, dropped)
<app>/<route>/run-N.stats.csv docker stats of the app: cpu_percent (of 1 CPU), mem_bytes
report.md                     median (min-max) of N runs, from `make report RESULTS=<folder>`
stress-<timestamp>/           stress runs: <type>-<app>.log (k6 output) and .stats.csv
```
