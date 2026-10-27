package changerisk

import (
	"strings"
	"testing"
)

func TestAPISourceLinksPinBothRevisionsAndEscapePaths(t *testing.T) {
	base, head := strings.Repeat("a", 40), strings.Repeat("b", 40)
	report := Report{RepositoryURL: "https://github.com/tree/repo", Base: base, Head: head,
		GoAPI: APIReport{Status: APIBreaking, Changes: []APIChange{{Package: "example.com/lib", Message: "Save: changed from func(int) to func(string)",
			Before: &APIDeclaration{Path: "old name#é).go", Line: 2, Declaration: "func Save(int)"},
			After:  &APIDeclaration{Path: "new.go", Line: 7, Declaration: "func Save(string)"}}}}}
	markdown := Markdown(report)
	for _, want := range []string{"https://github.com/tree/repo/blob/" + head + "/new.go#L7", "https://github.com/tree/repo/blob/" + base + "/old%20name%23%C3%A9%29.go#L2", "changed from func(int) to func(string)", "func Save(string)", "func Save(int)"} {
		if !strings.Contains(markdown, want) {
			t.Errorf("missing %q in %s", want, markdown)
		}
	}
	for _, invalid := range []struct{ repository, revision, path string }{
		{"https://github.com/owner/repo", "HEAD", "api.go"},
		{"https://github.com/owner/repo", head, "../api.go"},
		{"javascript:alert(1)", head, "api.go"},
		{"https://github.com/owner/repo?token=secret", head, "api.go"},
		{"https://github.com/owner/repo", strings.Repeat("z", 40), "api.go"},
	} {
		if link := sourceURL(invalid.repository, &APIDeclaration{Path: invalid.path, Line: 1}, invalid.revision); link != "" {
			t.Errorf("unsafe or unpinned link: %s", link)
		}
	}
}

func TestRemovedAPIAndNewPackagesUseCorrectSourceSide(t *testing.T) {
	base, head := strings.Repeat("a", 40), strings.Repeat("b", 40)
	report := Report{RepositoryURL: "https://github.com/owner/repo", Base: base, Head: head}
	removed := APIChange{Package: "example.com/lib", Message: "Old: removed", Before: &APIDeclaration{Path: "old.go", Line: 3, Declaration: "func Old()"}}
	if line := apiChangeLine(report, removed); !strings.Contains(line, "/blob/"+base+"/old.go#L3") || strings.Contains(line, head) {
		t.Fatalf("removed declaration linked to head: %s", line)
	}
	added := APIChange{Package: "example.com/lib/new", Message: "package added", Compatible: true, After: &APIDeclaration{Path: "new", Declaration: "package new", Exports: 4}}
	if line := apiChangeLine(report, added); !strings.Contains(line, "/tree/"+head+"/new") || !strings.Contains(line, "4 exported declarations") {
		t.Fatalf("new package linked as a file: %s", line)
	}
}

func TestDeclarationCodeSpansPreserveQuotedValuesAndEmbeddedBackticks(t *testing.T) {
	if got := codeSpan(`const Name = "value"`); got != "`const Name = \"value\"`" {
		t.Fatalf("quotes became HTML entities in a code span: %s", got)
	}
	if got := codeSpan("const Name = `value`"); got != "`` const Name = `value` ``" {
		t.Fatalf("backticks escaped the code span: %s", got)
	}
}

func TestReasonLocationsLinkTheRevisionThatHoldsThem(t *testing.T) {
	base, head := strings.Repeat("a", 40), strings.Repeat("b", 40)
	report := Report{RepositoryURL: "https://github.com/owner/repo", Base: base, Head: head}
	for _, check := range []struct {
		reason Reason
		want   string
	}{
		{Reason{Path: "stdlib.go"}, "[`stdlib.go`](https://github.com/owner/repo/blob/" + head + "/stdlib.go)"},
		{Reason{Path: "gone_test.go", Removed: true}, "[`gone_test.go`](https://github.com/owner/repo/blob/" + base + "/gone_test.go)"},
		{Reason{Path: "run.go", Line: 12}, "[`run.go:12`](https://github.com/owner/repo/blob/" + head + "/run.go#L12)"},
	} {
		if got := reasonLocation(report, check.reason); got != check.want {
			t.Errorf("location = %s, want %s", got, check.want)
		}
	}
	report.RepositoryURL = ""
	if got := reasonLocation(report, Reason{Path: "stdlib.go"}); got != "`stdlib.go`" {
		t.Errorf("unlinked whole-file location = %s", got)
	}
}

func TestRemovedSourcesLocateInBase(t *testing.T) {
	base := Snapshot{
		Functions:             []Function{{Name: "old", Path: "old.go", Line: 4, Body: "a"}},
		Declarations:          map[string]string{"old.go": "decl"},
		PreservedDeclarations: map[string]map[string]string{"old_test.go": {"": "test"}},
	}
	head := Snapshot{Declarations: map[string]string{}, PreservedDeclarations: map[string]map[string]string{}}
	_, reasons := functionChanges(base, head, Config{})
	reasons = append(reasons, declarationReasons(base, head)...)
	reasons = append(reasons, auxiliaryReasons(base, head)...)
	for _, reason := range reasons {
		if reason.Path != "" && !reason.Removed {
			t.Errorf("%s at %s must locate in base", reason.Rule, reason.Path)
		}
	}
	if len(reasons) < 3 {
		t.Fatalf("missing removal reasons: %+v", reasons)
	}
}
