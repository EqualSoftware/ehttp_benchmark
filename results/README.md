# Results

Each run writes a folder `results/<UTC timestamp>/` (git ignores it):

```
env.json                      machine, Docker VM, limits, versions, ehttp commit
<app>/<route>/run-N.json      k6 summary of the measured phase (rps, p50, p95, p99, errors, dropped,
                              failures by kind: timeout, refused, reset, http_5xx, ...)
<app>/<route>/run-N.stats.csv docker stats of the app: cpu_percent (of 1 CPU), mem_bytes
<app>/<route>/run-N.log       k6 output of the run (first LOG_LINES lines): the failed requests
<app>/<route>/run-N.container.json  app container state after the run: status, exit code, restarts, OOM
<app>/<route>/retried/        dirty runs replaced by a new run (RETRIES > 0), as run-N.try-K.*
<app>/container-N.log         app log of the container that served the runs from N on
report.md                     median (min-max) of N runs and the dirty runs, from `make report RESULTS=<folder>`
stress-<timestamp>/           stress runs: <type>-<app>.log (k6 output) and .stats.csv
```
