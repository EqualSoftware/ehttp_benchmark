# ehttp_benchmark

Two things:

- **B. Comparison:** ehttp vs net/http, Gin, Echo and Fiber, on 4 identical routes.
- **A. Stress tests:** stress, breakpoint, spike and soak (k6) against any of the 5 apps (default `ehttp`).

> **Limit of this setup (Mac).** Docker on a Mac runs the apps in a Linux VM.
> `cpuset` pins vCPUs of that VM. It does not pin the performance or efficiency
> cores of the host, and the host scheduler moves them. Port forwarding adds a
> constant cost to all apps. The results are valid for a **relative comparison
> on the same machine only**. They are **not absolute production numbers**.

## Layout

| Path | Content |
| :-- | :-- |
| `apps/{ehttp,nethttp,gin,echo,fiber}/main.go` | The 5 apps. One module (`go.mod`). |
| `internal/common` | Shared bodies, headers, port and server timeouts. |
| `docker/Dockerfile`, `compose.yaml` | Multi-stage Alpine build; one service per app, same limits. |
| `k6/compare/compare.js` | Fixed-rate comparison (open model). |
| `k6/stress/{stress,breakpoint,spike,soak}.js`, `k6/lib` | Stress tests and helpers. |
| `scripts/` | `verify`, `compare`, `compare-all`, `stress`, `env`. |
| `cmd/report` | Median and spread of N runs, as a Markdown table. |
| `results/` | Output (see `results/README.md`). |

## Quick start

```sh
make build            # 5 images
make verify           # the 5 apps answer the same status, headers and body
make smoke            # 10 s per app, 0 errors expected
make compare-all      # the comparison (10 rounds by default), writes results/<ts>/report.md
make up APP=ehttp && make stress-spike APP=ehttp
make down
```

Requires Docker, `k6` (tested v2.3.0), Go 1.27+ and `jq`.

### The ehttp library

`go.mod` has `replace github.com/EqualSoftware/ehttp => ../ehttp`. Clone ehttp
next to this repo. The Docker build gets it through a **named build context**:
`additional_contexts: ehttp: ../ehttp` in `compose.yaml`, copied to `/ehttp` in
the image, so the same `replace` works in the build. The commit of ehttp (and
whether the tree is dirty) goes into `env.json` of every result.

## The 4 routes (same in all apps)

| Route | Response |
| :-- | :-- |
| `GET /plaintext` | `Hello, World!`, `text/plain; charset=utf-8` |
| `GET /json` | `{"message":"Hello, World!"}`, `application/json`, serialized on each request |
| `GET /users/:id` | `{"id":"<id>"}`, `application/json` (router parameter) |
| `GET /mw/json` | Like `/json`, behind recover + request id + security headers |

`make verify` compares status, header set (names and values) and body of every
route across the 5 apps. Only `Date` and the value of `X-Request-Id` may differ
(they are masked).

### Fairness rules

Based on the [TechEmpower rules](https://github.com/TechEmpower/FrameworkBenchmarks/wiki/Project-Information-Framework-Tests-Overview):

- Every response has `Server: bench`, `Date` and `Content-Length`.
- No response cache. No access log (Gin and Echo start without the Logger middleware; ehttp does not use `middleware.Log` and has a no-op logger).
- Same bytes: Echo's default JSON serializer adds a trailing newline, so the app sets a serializer that writes `json.Marshal` bytes. Content types are set explicitly where a framework default differs (ehttp: `UTF-8` upper case; Fiber: adds a charset to JSON).
- No `GOMAXPROCS` override. Go 1.25+ reads the cgroup CPU limit ([release notes](https://go.dev/doc/go1.25)).
- Same limits for every container: `cpuset 0-3`, `cpus 4`, `mem_limit 1g` (override with `BENCH_CPUSET`, `BENCH_CPUS`, `BENCH_MEM`).
- Server timeouts: `ReadHeaderTimeout 10s`, all others 0 (the ehttp defaults), in every `net/http` based app. **Fiber (fasthttp) has no ReadHeaderTimeout**: its read, write and idle timeouts are 0. Prefork is off.
- Each app has one `Server` header step. ehttp, Gin, Echo and net/http use a small global middleware; Fiber uses its native `ServerHeader` option (no per-request cost in the handler chain).

### Middleware route: what each app uses

Headers on `/mw/json`: `X-Request-Id`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: SAMEORIGIN`, `X-XSS-Protection: 0` (the ehttp `DefaultSecureConfig`).

| App | Recover | Request id | Security headers |
| :-- | :-- | :-- | :-- |
| ehttp | `middleware.Recover()` | `middleware.RequestId()` (uuid, validates an incoming id) | `middleware.Secure(DefaultSecureConfig)` |
| net/http | own `defer recover` handler | own (google/uuid; keeps an incoming id) | own handler, 3 headers |
| Gin | `gin.Recovery()` | own (google/uuid) | own (3 headers). `gin-contrib/secure` is not used to avoid a dependency. |
| Echo | `middleware.Recover()` | `middleware.RequestID()` (random 32 chars, not a UUID) | `middleware.SecureWithConfig` (XSS `0`; default is `1; mode=block`) |
| Fiber | `recover.New()` | `requestid.New()` (UUID v4) | own (3 headers). `helmet` adds 8 more headers that it cannot disable, so it is not used. |

Differences to keep in mind: ehttp validates the incoming id; the id generator
differs (UUID string vs random string); Recover implementations differ in what
they build on a panic (nothing panics in the benchmark).

## Comparison method (B)

- **Open model** (`constant-arrival-rate`): the rate does not slow down when the app does, so latency under load is honest ([k6 docs](https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/open-vs-closed/)). `discardResponseBodies: true`; 1 request in 100 reads and checks the body.
- A **warm-up** phase (`WARMUP`, default 10 s) runs first. It is not counted: thresholds and the summary read metrics tagged `scenario:measured`.
- **N runs** per app and route (`RUNS`, default 10; [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat): at least 10, ideally 20). `compare-all` interleaves the apps and rotates their order each round, so drift (heat, other load) does not favor one app.
- The report shows **median (min-max)** of RPS, p50, p95, p99, error rate, `dropped_iterations`, app CPU (mean of `docker stats`, % of one CPU) and peak memory.
- **`dropped_iterations` must be 0.** If not, the generator could not hold the rate: lower `RATE` or raise `MAX_VUS`. The result is then not valid.
- A run with errors, dropped iterations or an app container that is not running after it is **dirty**. The report counts the dirty runs per app and route and lists each one with its failures by kind (k6 `error_code`: timeout, refused, reset, http_5xx, ...) and the container state. The k6 output of each run is in `run-N.log`, the app log in `container-N.log`.
- `RETRIES` (default 0) runs a dirty run again, up to that many times; the dirty one moves to `retried/` and the report counts it. A retry hides a dirty run from the tables, so keep `RETRIES=0` when the question is whether an app fails, and read the retried column.
- At a fixed rate, RPS is equal by design; compare latency and CPU. To compare capacity, raise `RATE` until `dropped` or errors appear, or use the breakpoint test on each app.
- Each run records `env.json`: machine, Docker VM CPUs and memory, limits, versions, ehttp commit.

```sh
make compare APP=gin RUNS=10 RATE=4000 DURATION=30s WARMUP=10s ROUTES="plaintext json users mw"
make compare-all RUNS=10
make compare-all RUNS=2 RUN_FROM=4 RESULTS=results/<ts>   # redo rounds 4-5 in an existing folder
make report RESULTS=results/<ts>
```

### Check the load generator

k6 needs at least 20% idle CPU ([guide](https://grafana.com/docs/k6/latest/testing-guides/running-large-tests/)). The apps run in the VM, k6 on the host:

1. Docker Desktop > Settings > Resources: set the VM CPUs (this machine: 8). The app uses `cpuset 0-3`. Keep the VM smaller than the host cores if you can, so k6 has free cores.
2. During a run, watch the host: `top -l 2 | grep -E 'CPU usage|k6'`. Idle must stay above 20%.
3. Check `dropped` in the report (0) and that the app CPU is below its limit (`cpus` x 100%) at the chosen `RATE`. If the app is at the limit, you measure saturation, not latency.
4. Close other apps. Plug in the power. Do not run two tests at once.

## Stress tests (A)

They target one benchmark app container (default `ehttp`, same limits as the comparison) on the 4 routes. `BASE_URL` overrides the target (default `http://127.0.0.1:8090`). `scripts/stress.sh` starts the app, runs k6, samples `docker stats` until k6 ends, prints the CPU/memory summary and stops the app. The output is the k6 summary and the stats file; there is no Grafana or Prometheus.

```sh
make stress-stress     APP=ehttp
make stress-breakpoint APP=gin
make stress-spike      APP=ehttp
make stress-soak       APP=ehttp
```

One iteration is 4 requests (one per route). Every size is an environment variable with a safe default (see the header of each script):

| Test | Shape ([k6 test types](https://grafana.com/docs/k6/latest/testing-guides/test-types/)) | Main env (default) |
| :-- | :-- | :-- |
| `stress` | ramp to average, hold, ramp above average, plateau, ramp down (closed model, VUs with think time). Default 9 min; real runs 5-60 min. | `AVG_VUS` 20, `STRESS_VUS` 100, `RAMP` 1m, `AVG_HOLD` 1m, `HOLD` 3m |
| `breakpoint` | `ramping-arrival-rate` until a threshold aborts (`abortOnFail`, `delayAbortEval` 30 s): p95 and error rate | `START_RATE` 50, `MAX_RATE` 8000, `DURATION` 10m, `P95_MS` 1000, `ERROR_RATE` 0.05 |
| `spike` | base rate, sudden peak, back to base, watch recovery | `BASE_RATE` 200, `SPIKE_RATE` 3000, `WARM` 1m, `SPIKE` 1m, `RECOVER` 3m |
| `soak` | constant moderate rate for long (default 30 min) | `RATE` 500, `DURATION` 30m |

Example, short: `make stress-spike WARM=5s SPIKE=10s RECOVER=10s`.

For soak, read memory in the stats file: a steady rise points to a leak.

## Not verified / known limits

- Numbers from `make smoke` are a check, not a benchmark.
- The app CPU from `docker stats` comes from the VM and has a sampling step of about 1.5 s. The sampler itself uses a little CPU.
- Results from a Mac are not comparable with Linux bare metal.
- The port forwarding of Docker Desktop can reset connections while the app is far from its CPU limit. Such runs show `failures: reset` in the report with a low app CPU. It is a cost of the setup, not of the app: redo the run.
- Noise on the host (background jobs, heat) can slow down every app for minutes. In the report, it shows as dirty runs of several apps in a row. Redo those rounds in the same folder: `make compare-all RUNS=<n> RUN_FROM=<first round> RESULTS=results/<ts>`.
