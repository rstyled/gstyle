package changerisk

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func TestAPIOnlyBreakUsesBasePolicyWithoutAnyFunctionPDG(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "[risk]\napi_breaking='high'\n")
	write(t, root, "api.go", "package sample\ntype Options struct { Value int }\n")
	base := commit(t, root)
	git(t, root, "remote", "add", "origin", "git@github.com:owner/repo.git")
	write(t, root, "gstyle.toml", "")
	write(t, root, "api.go", "package sample\ntype Options struct { Value string }\n")
	head := commit(t, root)
	t.Chdir(root)
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--base", base, "--head", head, "--format", "json"}, &output); err != nil {
		t.Fatal(err)
	}
	var report changerisk.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Risk != changerisk.High || report.GoAPI.Status != changerisk.APIBreaking || report.ChangedFunctions != 0 || report.DependenciesChanged {
		t.Fatalf("type-only break escaped the base policy: %s", output.String())
	}
	if len(report.GoAPI.Changes) == 0 || !strings.Contains(report.GoAPI.Changes[0].Message, "Value") {
		t.Fatalf("API break lacks concrete evidence: %+v", report.GoAPI)
	}
	change := report.GoAPI.Changes[0]
	if report.RepositoryURL != "https://github.com/owner/repo" || change.Before == nil || change.After == nil {
		t.Fatalf("committed report lost API sources or repository: %+v", report)
	}
	for _, source := range []*changerisk.APIDeclaration{change.Before, change.After} {
		if source.Path != "api.go" || source.Line != 2 {
			t.Fatalf("API source is not repository-relative: %+v", source)
		}
	}
	if markdown := changerisk.Markdown(report); !strings.Contains(markdown, "/blob/"+head+"/api.go#L2") || !strings.Contains(markdown, "/blob/"+base+"/api.go#L2") {
		t.Fatalf("API links are not pinned to the compared revisions: %s", markdown)
	}
	for _, field := range report.Unmeasured {
		if field == "compatibility" || field == "go_api" {
			t.Fatalf("collected Go API evidence marked unmeasured: %s", output.String())
		}
	}
}
