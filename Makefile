# ehttp_benchmark. Run `make help`.
APPS  ?= ehttp nethttp gin echo fiber
APP   ?= ehttp
export APPS

.PHONY: help build up down verify smoke compare compare-all report stress-stress stress-breakpoint stress-spike stress-soak vet

help:
	@echo "make build                 build the 5 images"
	@echo "make up APP=gin            start one app on 127.0.0.1:8090 (stops the others)"
	@echo "make down                  stop and remove the benchmark containers"
	@echo "make verify                check that the 5 apps answer the same status, headers and body"
	@echo "make smoke                 10 s k6 run per app, 0 errors expected"
	@echo "make compare APP=gin       RUNS (10) runs of each route for one app"
	@echo "make compare-all           all apps, interleaved (RUNS rounds), then report"
	@echo "make report RESULTS=dir    write report.md for a results folder"
	@echo "make stress-{stress,breakpoint,spike,soak} APP=ehttp"
	@echo "env: RUNS RATE DURATION WARMUP ROUTES BENCH_CPUSET BENCH_CPUS BENCH_MEM BENCH_PORT"

vet:
	go vet ./... && go build ./...

build:
	docker compose $(foreach a,$(APPS),--profile $(a)) build

up:
	@. scripts/lib.sh && app_down && app_up $(APP) && echo "$(APP) is up on $$BASE_URL"

down:
	@. scripts/lib.sh && app_down

verify:
	scripts/verify.sh

smoke:
	@. scripts/lib.sh; trap app_down EXIT; \
	for a in $(APPS); do \
	  app_down; app_up $$a; \
	  k6 run --quiet --no-usage-report -e APP=$$a -e ROUTE=json -e RATE=$${RATE:-2000} -e DURATION=10s -e WARMUP=0s \
	    -e BASE_URL=$$BASE_URL k6/compare/compare.js | grep '^RESULT' || exit 1; \
	done

compare:
	scripts/compare.sh $(APP)
	@. scripts/lib.sh; go run ./cmd/report $$(ls -dt results/*/ | head -1)

compare-all:
	scripts/compare-all.sh

report:
	go run ./cmd/report $(RESULTS)

stress-stress stress-breakpoint stress-spike stress-soak:
	APP=$(APP) scripts/stress.sh $(patsubst stress-%,%,$@)
