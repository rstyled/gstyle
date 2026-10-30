package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTUIConfigLoadsQualityAndPatternPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gstyle.toml")
	source := "[fn_params]\nmax = 7\n[patterns]\nsimilarity_threshold = 85\n[patterns.measurements]\nmin_function_calls = 2\n"
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	for _, cpg := range []bool{false, true} {
		config, err := loadRunConfig(runOptions{config: path, cpg: cpg})
		if err != nil {
			t.Fatal(err)
		}
		if config.quality.FnParams.Max != 7 || config.patterns.SimilarityThreshold != 85 || config.patterns.Measurements.MinFunctionCalls != 2 {
			t.Fatal("TUI does not load both policies from the same file")
		}
	}
}
