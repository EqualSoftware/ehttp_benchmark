// Command report reads a results folder of scripts/compare.sh and writes
// report.md with the median and the spread (min-max) of N runs for each
// app and route, and lists the dirty runs (errors, dropped iterations or a
// container that is not running) with what failed.
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
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// runFile matches a k6 summary (run-N.json, or run-N.try-K.json in retried/),
// not the other files of a run such as run-N.container.json.
var runFile = regexp.MustCompile(`^run-\d+(\.try-\d+)?\.json$`)

// runFiles returns the k6 summaries in dir.
func runFiles(dir string) []string {
	files, _ := filepath.Glob(filepath.Join(dir, "run-*.json"))

	return slices.DeleteFunc(files, func(f string) bool { return !runFile.MatchString(filepath.Base(f)) })
}

type run struct {
	RPS     float64 `json:"rps"`
	P50     float64 `json:"p50_ms"`
	P95     float64 `json:"p95_ms"`
	P99     float64 `json:"p99_ms"`
	Err     float64 `json:"error_rate"`
	Dropped float64 `json:"dropped"`

	Failures map[string]float64 `json:"failures"` // failed requests by kind

	name      string    // app/route/run-N
	container container // state of the app container after the run
	cpu, mem  float64   // mean CPU percent and peak memory in MiB of the run
}

// container is run-N.container.json. Status is empty when the file is missing
// (runs made before it existed).
type container struct {
	Status       string `json:"status"`
	OOMKilled    bool   `json:"oom_killed"`
	ExitCode     int    `json:"exit_code"`
	RestartCount int    `json:"restart_count"`
}

func (r run) dirty() bool {
	c := r.container
	return r.Err > 0 || r.Dropped > 0 || c.OOMKilled || c.RestartCount > 0 || (c.Status != "" && c.Status != "running")
}

// detail describes a dirty run in one line.
func (r run) detail() string {
	parts := []string{fmt.Sprintf("errors %.2f%%, dropped %.0f, p99 %.3f ms", r.Err*100, r.Dropped, r.P99)}

	if len(r.Failures) > 0 {
		kinds := make([]string, 0, len(r.Failures))
		for k, n := range r.Failures {
			kinds = append(kinds, fmt.Sprintf("%s %.0f", k, n))
		}

		slices.Sort(kinds)
		parts = append(parts, "failures: "+strings.Join(kinds, ", "))
	}

	if c := r.container; c.Status != "" {
		parts = append(parts, fmt.Sprintf("container: %s, exit %d, restarts %d, oom %t", c.Status, c.ExitCode, c.RestartCount, c.OOMKilled))
	}

	return fmt.Sprintf("- `%s`: %s", r.name, strings.Join(parts, "; "))
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

	var dirty []string

	for _, route := range routes {
		dirty = append(dirty, table(&out, dir, route, names)...)
	}

	if len(dirty) > 0 {
		fmt.Fprintf(&out, "\n## Dirty runs\n\nRuns with errors, dropped iterations or a container that is not running. "+
			"They count in the tables above. The k6 output is in run-N.log, the app log in container-N.log.\n\n%s\n",
			strings.Join(dirty, "\n"))
	}

	if err := os.WriteFile(filepath.Join(dir, "report.md"), []byte(out.String()), 0o644); err != nil {
		log.Fatal(err)
	}

	fmt.Print(out.String())
}

// table writes the table of one route and returns the details of its dirty runs.
func table(out *strings.Builder, dir, route string, apps []string) []string {
	var rows, dirty []string

	for _, app := range apps {
		runs := load(filepath.Join(dir, app, route), app+"/"+route)
		if len(runs) == 0 {
			continue
		}

		n := 0

		for _, r := range runs {
			if r.dirty() {
				n++

				dirty = append(dirty, r.detail())
			}
		}

		retried := runFiles(filepath.Join(dir, app, route, "retried"))

		rows = append(rows, fmt.Sprintf("| %s | %d | %d | %d | %s | %s | %s | %s | %s | %s | %s | %s |",
			app, len(runs), n, len(retried),
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
		return nil
	}

	fmt.Fprintf(out, "\n## %s\n\nMedian (min-max) of N runs. dirty: runs with errors, dropped iterations "+
		"or a container that is not running (see Dirty runs). retried: dirty runs replaced by a new run (RETRIES).\n\n", route)
	out.WriteString("| app | runs | dirty | retried | RPS | p50 ms | p95 ms | p99 ms | errors % | dropped | CPU % (of 1 CPU) | mem MiB |\n")
	out.WriteString("| :-- | --: | --: | --: | --: | --: | --: | --: | --: | --: | --: | --: |\n")
	out.WriteString(strings.Join(rows, "\n") + "\n")

	return dirty
}

func load(dir, prefix string) []run {
	files := runFiles(dir)
	out := make([]run, 0, len(files))

	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		var r run
		if json.Unmarshal(b, &r) != nil {
			continue
		}

		base := strings.TrimSuffix(f, ".json")
		r.name = prefix + "/" + filepath.Base(base)
		r.cpu, r.mem = stats(base + ".stats.csv")

		if b, err := os.ReadFile(base + ".container.json"); err == nil {
			_ = json.Unmarshal(b, &r.container)
		}

		out = append(out, r)
	}

	return out
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
