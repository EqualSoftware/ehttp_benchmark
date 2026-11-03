#!/usr/bin/env bash
# Prints the test environment as JSON. compare.sh saves it next to the results.
set -euo pipefail
cd "$(dirname "$0")/.."

# ehttp comes from GitHub at the version in go.mod. A pseudo-version ends with the commit.
ehttp_version="$(go list -m -f '{{.Version}}' github.com/EqualSoftware/ehttp)"

jq -n \
  --arg date "$(date -u +%FT%TZ)" \
  --arg machine "$(sysctl -n machdep.cpu.brand_string 2>/dev/null || uname -m)" \
  --arg host_cores "$(sysctl -n hw.ncpu 2>/dev/null || nproc)" \
  --arg host_mem_gb "$(( $(sysctl -n hw.memsize 2>/dev/null || echo 0) / 1073741824 ))" \
  --arg os "$(uname -srm)" \
  --arg docker "$(docker version --format '{{.Server.Version}}' 2>/dev/null)" \
  --arg vm_cpus "$(docker info --format '{{.NCPU}}' 2>/dev/null)" \
  --arg vm_mem_bytes "$(docker info --format '{{.MemTotal}}' 2>/dev/null)" \
  --arg cpuset "${BENCH_CPUSET:-0-3}" \
  --arg cpus "${BENCH_CPUS:-4}" \
  --arg mem "${BENCH_MEM:-1g}" \
  --arg k6 "$(k6 version | head -1)" \
  --arg go "$(grep -m1 '^FROM golang' docker/Dockerfile | cut -d' ' -f2)" \
  --arg ehttp_version "$ehttp_version" \
  --arg ehttp_commit "${ehttp_version##*-}" \
  --arg bench_commit "$(git rev-parse --verify -q HEAD || echo uncommitted)" \
  --arg gin "$(go list -m -f '{{.Version}}' github.com/gin-gonic/gin)" \
  --arg echo "$(go list -m -f '{{.Version}}' github.com/labstack/echo/v5)" \
  --arg fiber "$(go list -m -f '{{.Version}}' github.com/gofiber/fiber/v3)" \
  '{date:$date, host:{machine:$machine, cores:$host_cores, mem_gb:$host_mem_gb, os:$os},
    docker:{version:$docker, vm_cpus:$vm_cpus, vm_mem_bytes:$vm_mem_bytes},
    app_limits:{cpuset:$cpuset, cpus:$cpus, mem:$mem},
    versions:{k6:$k6, go_image:$go, gin:$gin, echo:$echo, fiber:$fiber, nethttp:"stdlib of the Go image"},
    ehttp:{version:$ehttp_version, commit:$ehttp_commit}, bench_commit:$bench_commit}'
