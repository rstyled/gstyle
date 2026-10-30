package qualitycheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestCPPCheckAndSavedFactsUseSharedEvaluator(t *testing.T) {
	fixture := "../../adapters/cpp/testdata/features.graphson.json"
	var facts bytes.Buffer
	if err := Run(context.Background(), []string{"--cpg", "--format", "facts", fixture}, &facts); err != nil {
		t.Fatal(err)
	}
	saved := writeTemp(t, "cpp-facts.json", facts.String())
	var reports []quality.Report
	for _, args := range [][]string{
		{"--cpg", "--format", "json", fixture},
		{"--facts-in", saved, "--format", "json"},
	} {
		var output bytes.Buffer
		if err := Run(context.Background(), args, &output); !errors.Is(err, ErrFindings) {
			t.Fatalf("C++ check does not report findings: %v", err)
		}
		var report quality.Report
		if err := json.Unmarshal(output.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		reports = append(reports, report)
	}
	if !reflect.DeepEqual(reports[0], reports[1]) || reports[0].Summary.Functions == 0 {
		t.Fatal("saved C++ facts lose evaluation evidence")
	}
}

func TestCPPCheckUsesProjectPolicy(t *testing.T) {
	config := writeTemp(t, "cpp.toml", "[fn_params]\nmax = 0\n")
	var output bytes.Buffer
	err := Run(context.Background(), []string{"--cpg", "--config", config, "--format", "json", "../../adapters/cpp/testdata/features.graphson.json"}, &output)
	if !errors.Is(err, ErrFindings) {
		t.Fatal(err)
	}
	var report quality.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.RuleID == "fn_params" && diagnostic.Limit == 0 {
			return
		}
	}
	t.Fatal("C++ check ignores the project policy")
}
