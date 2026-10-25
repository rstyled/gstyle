package gorisk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/changerisk"
	"github.com/rstyled/gstyle/domain/pdg"
)

func sourceModule(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files["go.mod"] = "module example.com/projection\n\ngo 1.26.0\n"
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestProjectionIgnoresOpaqueIDsAndSourceLines(t *testing.T) {
	root := sourceModule(t, map[string]string{"input.go": "package sample\nfunc Plus(value int) int { return value+1 }\n"})
	ext, err := gopatterns.Extract(context.Background(), root, []string{"input.go"})
	if err != nil || len(ext.Funcs) != 1 {
		t.Fatalf("extraction: %v", err)
	}
	graph := ext.Funcs[0].Pdg
	before, err := changerisk.Project(&graph)
	if err != nil {
		t.Fatal(err)
	}
	for i := range graph.Nodes {
		graph.Tables.Strings = append(graph.Tables.Strings, fmt.Sprintf("renamed-node-%d", i))
		graph.Nodes[i].ID = pdg.Text(len(graph.Tables.Strings))
		graph.Nodes[i].Line += 200
	}
	after, err := changerisk.Project(&graph)
	if err != nil || before.Shape != after.Shape {
		t.Fatalf("metadata changed fingerprint: %v", err)
	}
	slices.Reverse(graph.Nodes)
	for i := range graph.Edges {
		graph.Edges[i].Source = pdg.Ref(len(graph.Nodes)) + 1 - graph.Edges[i].Source
		graph.Edges[i].Target = pdg.Ref(len(graph.Nodes)) + 1 - graph.Edges[i].Target
	}
	for i := range graph.Definitions {
		graph.Definitions[i].Node = pdg.Ref(len(graph.Nodes)) + 1 - graph.Definitions[i].Node
	}
	graph.NormalFlow, graph.ValueFlow = nil, nil
	renumbered, err := changerisk.Project(&graph)
	if err != nil || before.Shape != renumbered.Shape {
		t.Fatalf("graph-local references changed fingerprint: %v", err)
	}
}

func TestMultipleInitDeclarationsKeepDistinctIdentities(t *testing.T) {
	root := sourceModule(t, map[string]string{"init.go": "package sample\nimport \"os\"\nfunc init() {}\nfunc init() { _ = os.WriteFile(\"file\",nil,0600) }\n"})
	functions, err := Extract(context.Background(), root, []string{"init.go"})
	if err != nil || len(functions) != 2 {
		t.Fatalf("init extraction: %v", err)
	}
	if functions[0].ID == functions[1].ID {
		t.Fatal("init declarations overwrite each other")
	}
	report := changerisk.Evaluate(changerisk.Snapshot{}, changerisk.Snapshot{Functions: functions}, changerisk.Config{Calls: []changerisk.CallRule{{Callee: "os.WriteFile", Level: changerisk.High}}})
	if report.ChangedFunctions != 2 || report.Risk != changerisk.High {
		t.Fatalf("an init call disappeared: %+v", report)
	}
}

func TestClosuresAreMeasuredWithoutBorrowingOtherFiles(t *testing.T) {
	root := sourceModule(t, map[string]string{
		"a.go": "package sample\nfunc Plain(value int) int { value++; value++; value++; value++; value++; return value }\n",
		"b.go": "package sample\nimport \"os\"\nfunc Other() error { return os.WriteFile(\"file\",nil,0600) }\n",
	})
	plain, err := Extract(context.Background(), root, []string{"a.go"})
	if err != nil || len(plain) != 1 {
		t.Fatalf("plain extraction: %v", err)
	}
	for _, node := range plain[0].Graph.Nodes {
		if node.Callee == "os.WriteFile" {
			t.Fatal("borrowed a call from another source file")
		}
	}
	closure := "package sample\nimport \"os\"\nfunc Parent() func() error { return func() error { return os.WriteFile(\"file\",nil,0600) } }\n"
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte(closure), 0600); err != nil {
		t.Fatal(err)
	}
	functions, err := Extract(context.Background(), root, []string{"a.go"})
	if err != nil || len(functions) != 1 {
		t.Fatalf("closure extraction: %v", err)
	}
	found := false
	for _, node := range functions[0].Graph.Nodes {
		found = found || node.Callee == "os.WriteFile"
	}
	if !found || !functions[0].Graph.Ambiguous {
		t.Fatal("closure call disappeared or acquired a fabricated cross-closure diagram")
	}
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("\n\n\n"+closure), 0600); err != nil {
		t.Fatal(err)
	}
	moved, err := Extract(context.Background(), root, []string{"a.go"})
	if err != nil || moved[0].Graph.Shape != functions[0].Graph.Shape || moved[0].Body != functions[0].Body {
		t.Fatalf("closure offsets changed the comparison: %v", err)
	}
}

func TestLiteralClosureTextIsNotTreatedAsAnIdentity(t *testing.T) {
	root := sourceModule(t, map[string]string{"a.go": "package sample\nfunc Value() string { return \".closure@123\" }\n"})
	before, err := Extract(context.Background(), root, []string{"a.go"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package sample\nfunc Value() string { return \".closure@456\" }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := Extract(context.Background(), root, []string{"a.go"})
	if err != nil || before[0].Graph.Shape == after[0].Graph.Shape {
		t.Fatalf("literal change was normalized away: %v", err)
	}
}

func TestIndirectCallTargetsRemainUnknown(t *testing.T) {
	root := sourceModule(t, map[string]string{"indirect.go": "package sample\nimport \"os\"\nfunc Save() error { write := os.WriteFile; return write(\"file\",nil,0600) }\n"})
	functions, err := Extract(context.Background(), root, []string{"indirect.go"})
	if err != nil || len(functions) != 1 {
		t.Fatalf("indirect extraction: %v", err)
	}
	report := changerisk.Evaluate(changerisk.Snapshot{}, changerisk.Snapshot{Functions: functions}, changerisk.Config{PDGChange: changerisk.Medium, Calls: []changerisk.CallRule{{Callee: "os.WriteFile", Level: changerisk.High}}})
	if report.PDGRisk != changerisk.Unknown {
		t.Fatalf("unresolved call received a conclusive verdict: %+v", report)
	}
	found := false
	for _, reason := range report.Reasons {
		if reason.Rule == "added_call" {
			t.Fatalf("indirect call acquired an invented target: %+v", reason)
		}
		found = found || reason.Rule == "unresolved_calls"
	}
	if !found {
		t.Fatalf("unresolved target evidence missing: %+v", report)
	}
}
