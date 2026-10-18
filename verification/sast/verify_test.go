package sast

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "refresh the reviewed SAST baseline")

type corpusCase struct {
	Function string   `json:"function"`
	Label    string   `json:"label"`
	Expected []string `json:"expected"`
	Gap      string   `json:"gap,omitempty"`
	Observed []string `json:"observed"`
}

type coverage struct {
	Analysis     string `json:"analysis"`
	Complete     int    `json:"complete"`
	Partial      int    `json:"partial"`
	NotEvaluated int    `json:"not_evaluated"`
	Unavailable  int    `json:"unavailable"`
}

type candidate struct {
	Security *struct {
		RuleID string `json:"rule_id"`
		CWE    string `json:"cwe"`
	} `json:"security"`
	Sites    []struct{ Name string }
	Evidence struct {
		Nodes []string `json:"node_ids"`
	} `json:"analysis_evidence"`
}

type baseline struct {
	Version  int           `json:"version"`
	Summary  summaryCounts `json:"summary"`
	Coverage coverage      `json:"analysis_coverage"`
	Cases    []corpusCase  `json:"cases"`
}

func readJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, dir, name string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stderr strings.Builder
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s: %v\n%s", name, err, stderr.String())
	}
	return data
}

func scan(t *testing.T, binary, dir, format string, value any) {
	t.Helper()
	data := run(t, dir, binary, "patterns", "--format", format, ".")
	if err := json.Unmarshal(data, value); err != nil {
		t.Fatal(err)
	}
}

func prepareCLI(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join("testdata", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, entry.Name()), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	binary := filepath.Join(dir, "gstyle")
	run(t, "../..", "go", "build", "-buildvcs=false", "-o", binary, ".")
	return binary, dir
}

func observedFindings(t *testing.T, cases []corpusCase, candidates []candidate) map[string][]string {
	t.Helper()
	observed := map[string][]string{}
	for _, row := range cases {
		if _, exists := observed[row.Function]; exists {
			t.Fatalf("duplicate function: %s", row.Function)
		}
		observed[row.Function] = []string{}
	}
	for _, item := range candidates {
		recordFinding(t, observed, item)
	}
	return observed
}

func recordFinding(t *testing.T, observed map[string][]string, item candidate) {
	t.Helper()
	if item.Security == nil {
		return
	}
	if len(item.Sites) == 0 || len(item.Evidence.Nodes) == 0 || !strings.HasPrefix(item.Security.CWE, "CWE-") {
		t.Fatalf("missing security evidence: %+v", item)
	}
	name := item.Sites[0].Name
	name = name[strings.LastIndex(name, ".")+1:]
	if _, exists := observed[name]; !exists {
		t.Fatalf("unlabeled security finding: %s", name)
	}
	observed[name] = append(observed[name], item.Security.RuleID)
}

type summaryCounts struct {
	Unsafe         int `json:"unsafe"`
	DetectedUnsafe int `json:"detected_unsafe"`
	Safe           int `json:"safe"`
	FlaggedSafe    int `json:"flagged_safe"`
	KnownGaps      int `json:"known_gaps"`
}

func evaluateCases(t *testing.T, cases []corpusCase, observed map[string][]string) summaryCounts {
	t.Helper()
	summary := map[string]int{"unsafe": 0, "detected_unsafe": 0, "safe": 0, "flagged_safe": 0, "known_gaps": 0}
	for i := range cases {
		row := &cases[i]
		row.Observed = observed[row.Function]
		slices.Sort(row.Observed)
		expected := slices.Clone(row.Expected)
		slices.Sort(expected)
		if !slices.Equal(row.Observed, expected) {
			t.Fatalf("%s: got %v, want %v", row.Function, row.Observed, expected)
		}
		wrong := (len(row.Observed) > 0) != (row.Label == "unsafe")
		if wrong != (row.Gap != "") {
			t.Fatalf("stale gap label: %s", row.Function)
		}
		countCase(summary, *row)
	}
	return summaryCounts{summary["unsafe"], summary["detected_unsafe"], summary["safe"], summary["flagged_safe"], summary["known_gaps"]}
}

func countCase(summary map[string]int, row corpusCase) {
	if row.Label == "unsafe" {
		summary["unsafe"]++
		if len(row.Observed) > 0 {
			summary["detected_unsafe"]++
		}
	}
	if row.Label == "safe" {
		summary["safe"]++
		if len(row.Observed) > 0 {
			summary["flagged_safe"]++
		}
	}
	if row.Gap != "" {
		summary["known_gaps"]++
	}
}

func checkSARIF(t *testing.T, binary, dir string, observed map[string][]string) {
	t.Helper()
	var report struct {
		Runs []struct {
			Results []struct {
				RuleID string `json:"ruleId"`
			} `json:"results"`
		} `json:"runs"`
	}
	scan(t, binary, dir, "sarif", &report)
	if len(report.Runs) != 1 {
		t.Fatalf("want one SARIF run, got %d", len(report.Runs))
	}
	actual, expected := map[string]int{}, map[string]int{}
	for _, result := range report.Runs[0].Results {
		if strings.HasPrefix(result.RuleID, "go/") {
			actual[result.RuleID]++
		}
	}
	for _, rules := range observed {
		for _, rule := range rules {
			expected[rule]++
		}
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("JSON/SARIF disagreement: %v != %v", actual, expected)
	}
}

func taintCoverage(t *testing.T, rows []coverage, count int) coverage {
	t.Helper()
	for _, row := range rows {
		if row.Analysis != "taint_flow" {
			continue
		}
		if row.Complete+row.Partial+row.NotEvaluated+row.Unavailable != count {
			t.Fatalf("incomplete corpus coverage: %+v", row)
		}
		return row
	}
	t.Fatal("missing taint_flow coverage")
	return coverage{}
}

func TestSourceBaseline(t *testing.T) {
	var manifest struct {
		Cases []corpusCase `json:"cases"`
	}
	readJSON(t, "cases.json", &manifest)
	binary, dir := prepareCLI(t)
	var report struct {
		Candidates []candidate `json:"candidates"`
		Coverage   []coverage  `json:"analysis_coverage"`
	}
	scan(t, binary, dir, "json", &report)
	observed := observedFindings(t, manifest.Cases, report.Candidates)
	result := baseline{Version: 1, Summary: evaluateCases(t, manifest.Cases, observed), Coverage: taintCoverage(t, report.Coverage, len(manifest.Cases)), Cases: manifest.Cases}
	checkSARIF(t, binary, dir, observed)
	if *update {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("baseline.json", append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var expected baseline
	readJSON(t, "baseline.json", &expected)
	if !reflect.DeepEqual(result, expected) {
		t.Fatal("baseline changed: inspect and run make sast-baseline")
	}
	t.Logf("coverage: %v", result.Summary)
}
