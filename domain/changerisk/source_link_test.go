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
