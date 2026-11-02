// Command report reads a results folder of scripts/compare.sh and writes
// report.md with the median and the spread (min-max) of N runs for each
// app and route.
//
//	go run ./cmd/report results/<run>
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type run struct {
	RPS     float64 `json:"rps"`
	P50     float64 `json:"p50_ms"`
	P95     float64 `json:"p95_ms"`
	P99     float64 `json:"p99_ms"`
	Err     float64 `json:"error_rate"`
	Dropped float64 `json:"dropped"`

	cpu, mem float64 // mean CPU percent and peak memory in MiB of the run
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: report <results folder>")
	}

	dir := os.Args[1]

	var out strings.Builder

	if env, err := os.ReadFile(filepath.Join(dir, "env.json")); err == nil {
		fmt.Fprintf(&out, "# Report\n\nEnvironment (env.json):\n\n```json\n%s```\n", env)
	}

	routes := []string{"plaintext", "json", "users", "mw"}
	apps, _ := filepath.Glob(filepath.Join(dir, "*", "*", "run-*.json"))
	seen := map[string]bool{}

	for _, f := range apps {
		seen[filepath.Base(filepath.Dir(filepath.Dir(f)))] = true
	}

	names := make([]string, 0, len(seen))
	for a := range seen {
		names = append(names, a)
	}

	slices.Sort(names)

	for _, route := range routes {
		table(&out, dir, route, names)
	}

	if err := os.WriteFile(filepath.Join(dir, "report.md"), []byte(out.String()), 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Print(out.String())
}

func table(out *strings.Builder, dir, route string, apps []string) {
	var rows []string

	for _, app := range apps {
		runs := load(filepath.Join(dir, app, route))
		if len(runs) == 0 {
			continue
		}

		rows = append(rows, fmt.Sprintf("| %s | %d | %s | %s | %s | %s | %s | %s | %s | %s |",
			app, len(runs),
			spread(runs, func(r run) float64 { return r.RPS }, "%.0f"),
			spread(runs, func(r run) float64 { return r.P50 }, "%.3f"),
			spread(runs, func(r run) float64 { return r.P95 }, "%.3f"),
			spread(runs, func(r run) float64 { return r.P99 }, "%.3f"),
			spread(runs, func(r run) float64 { return r.Err * 100 }, "%.2f"),
			spread(runs, func(r run) float64 { return r.Dropped }, "%.0f"),
			spread(runs, func(r run) float64 { return r.cpu }, "%.0f"),
			spread(runs, func(r run) float64 { return r.mem }, "%.1f")))
	}

	if len(rows) == 0 {
		return
	}

	fmt.Fprintf(out, "\n## %s\n\nMedian (min-max) of N runs.\n\n", route)
	out.WriteString("| app | runs | RPS | p50 ms | p95 ms | p99 ms | errors % | dropped | CPU % (of 1 CPU) | mem MiB |\n")
	out.WriteString("| :-- | --: | --: | --: | --: | --: | --: | --: | --: | --: |\n")
	out.WriteString(strings.Join(rows, "\n") + "\n")
}

func load(dir string) []run {
	files, _ := filepath.Glob(filepath.Join(dir, "run-*.json"))
	runs := make([]run, 0, len(files))

	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		var r run
		if json.Unmarshal(b, &r) != nil {
			continue
		}

		r.cpu, r.mem = stats(strings.TrimSuffix(f, ".json") + ".stats.csv")
		runs = append(runs, r)
	}

	return runs
}

// stats returns the mean CPU percent and the peak memory (MiB) of a docker stats file.
func stats(path string) (cpu, mem float64) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	rows, _ := csv.NewReader(f).ReadAll()
	n := 0

	for _, row := range rows[min(1, len(rows)):] {
		if len(row) < 2 {
			continue
		}

		c, err1 := strconv.ParseFloat(row[0], 64)
		m, err2 := strconv.ParseFloat(row[1], 64)

		if err1 != nil || err2 != nil {
			continue
		}

		cpu += c
		mem = max(mem, m/1048576)
		n++
	}

	if n > 0 {
		cpu /= float64(n)
	}

	return cpu, mem
}

func spread(runs []run, get func(run) float64, format string) string {
	v := make([]float64, len(runs))
	for i, r := range runs {
		v[i] = get(r)
	}

	slices.Sort(v)

	return fmt.Sprintf(format+" ("+format+"-"+format+")", median(v), v[0], v[len(v)-1])
}

func median(sorted []float64) float64 {
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}

	return (sorted[n/2-1] + sorted[n/2]) / 2
}
