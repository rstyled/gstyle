package changerisk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func TestSmallHelperWithAddedTestsAndTighterBaselinePassesRiskCheck(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "[risk]\npdg_change='medium'\npdg_low_risk_max_functions=3\ndependency_change='high'\napi_breaking='high'\n")
	write(t, root, "source.go", "package sample\nfunc Count(n int) int { return n+1 }\n")
	write(t, root, "source_test.go", "package sample\nimport \"testing\"\nfunc TestExisting(t *testing.T) { if Count(0)!=1 {t.Fatal(\"count\")} }\n")
	baseline := `{"version":1,"functions":100,"tolerance":0.05,"rules":{"density":{"limit":500,"excess":10,"per_1000_functions":100}}}`
	write(t, root, "quality-baseline.json", baseline)
	base := commit(t, root)
	write(t, root, "source.go", "package sample\nfunc Count(n int) int { return helper(n) }\nfunc helper(n int) int { return n+1 }\n")
	write(t, root, "extra_test.go", "package sample\nimport \"testing\"\nfunc TestExtra(t *testing.T) { if Count(1)!=2 {t.Fatal(\"count\")} }\n")
	write(t, root, "quality-baseline.json", strings.ReplaceAll(baseline, `"excess":10,"per_1000_functions":100`, `"excess":9,"per_1000_functions":90`))
	head := commit(t, root)
	t.Chdir(root)
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--base", base, "--head", head, "--check", "--format", "json"}, &output); err != nil {
		t.Fatalf("covered change rejected: %v\n%s", err, output.String())
	}
	var report changerisk.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Risk != changerisk.Low || report.ChangedFunctions != 2 || report.DependenciesChanged {
		t.Fatalf("wrong covered report: %s", output.String())
	}
	write(t, root, "quality-baseline.json", strings.ReplaceAll(baseline, `"excess":10,"per_1000_functions":100`, `"excess":11,"per_1000_functions":110`))
	head = commit(t, root)
	output.Reset()
	err := Run(context.Background(), []string{"--base", base, "--head", head, "--check", "--format", "json"}, &output)
	if !errors.Is(err, ErrReviewRequired) || !strings.Contains(output.String(), "permitted excess rates") {
		t.Fatalf("weaker baseline accepted: %v\n%s", err, output.String())
	}
}

func TestAuxiliaryFilesRemainCoveredByEmbedsAcrossBuilds(t *testing.T) {
	for _, name := range []string{"quality-baseline.json", "value_test.go"} {
		t.Run(name, func(t *testing.T) {
			root := repository(t)
			write(t, root, name, "content")
			write(t, root, "asset_windows.go", "//go:build windows\n\npackage sample\nimport _ \"embed\"\n//go:embed "+name+"\nvar Content string\n")
			commit(t, root)
			message, err := embeddedDocumentationCoverage(context.Background(), root, []string{name})
			if err != nil || !strings.Contains(message, name) {
				t.Fatalf("embedded auxiliary resource disappeared: %q %v", message, err)
			}
		})
	}
}
