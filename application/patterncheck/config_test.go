package patterncheck

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func patternPolicy(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gstyle.toml")
	if err := os.WriteFile(path, []byte("[patterns]\nsimilarity_threshold = 85\nmin_coverage = 80\n[patterns.measurements]\nmin_source_lines = 3\nmin_function_calls = 2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPatternCLIThresholdOverridesPolicyIncludingDefaultValue(t *testing.T) {
	path := patternPolicy(t)
	for _, test := range []struct {
		flags []string
		want  uint32
	}{
		{nil, 85},
		{[]string{"--threshold", "60"}, 60},
		{[]string{"--threshold", "100"}, 100},
		{[]string{"--threshold", "0"}, 0},
	} {
		args := append([]string{"--config", path}, test.flags...)
		opts, err := parseOptions(args)
		if err != nil {
			t.Fatal(err)
		}
		if opts.settings.SimilarityThreshold != test.want || opts.settings.MinCoverage != 80 {
			t.Fatalf("CLI precedence is wrong: %+v", opts.settings)
		}
	}
}

func TestCPPCLIUsesMeasurementPolicy(t *testing.T) {
	var output strings.Builder
	fixture := "../../adapters/cpp/testdata/features.graphson.json"
	if err := Run(context.Background(), []string{"--cpg", "--config", patternPolicy(t), "--format", "json", fixture}, &output); err != nil {
		t.Fatal(err)
	}
	var report patterns.PatternsReport
	if err := json.Unmarshal([]byte(output.String()), &report); err != nil {
		t.Fatal(err)
	}
	if report.Configuration == nil || report.Configuration.SimilarityThreshold != 85 || len(report.AnalysisCoverage) != 27 {
		t.Fatal("CPP CLI drops effective settings or enabled analyses")
	}
	for _, measurement := range report.Measurements {
		if measurement.Calls < 2 || len(measurement.SourceLines) < 3 {
			t.Fatalf("CPP CLI ignores codebase thresholds: %+v", measurement)
		}
	}
}

func TestGoCLIAndTUIConfigPathUsesSharedMiningSettings(t *testing.T) {
	t.Chdir("testdata/single")
	config := patterns.DefaultConfig()
	config.SimilarityThreshold, config.MinCoverage, config.MaxHoles = 97, 90, 1
	report, err := GetReportWithConfig(context.Background(), []string{"--threshold", "60", "."}, &config)
	if err != nil {
		t.Fatal(err)
	}
	if report.Configuration == nil || report.Configuration.SimilarityThreshold != 60 || report.Configuration.MinCoverage != 90 || report.Configuration.MaxHoles != 1 {
		t.Fatal("Go mining does not receive the merged pattern policy")
	}
}
