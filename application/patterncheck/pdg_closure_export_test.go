package patterncheck

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPDGExportContainsEveryLexicalClosureBody(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example\n\ngo 1.26\n", "p.go": `package example;func f(){fn:=func(){func(){}()};fn()}`} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	var output strings.Builder
	if err := Run(context.Background(), []string{"--format", "pdg-json", "."}, &output); err != nil {
		t.Fatal(err)
	}
	var documents []struct {
		Function struct{ ID string }
		Nodes    []struct {
			Attributes struct{ Extensions map[string]json.RawMessage }
		}
	}
	if err := json.Unmarshal([]byte(output.String()), &documents); err != nil {
		t.Fatal(err)
	}
	if len(documents) != 3 {
		t.Fatalf("exported graphs=%d want 3 canonical callables", len(documents))
	}
	ids := map[string]bool{}
	for _, document := range documents {
		if ids[document.Function.ID] {
			t.Fatal("duplicate callable graph")
		}
		ids[document.Function.ID] = true
	}
	references := 0
	for _, document := range documents {
		for _, node := range document.Nodes {
			data, found := node.Attributes.Extensions["gstyle.closure:body-id"]
			if !found {
				continue
			}
			var body string
			if err := json.Unmarshal(data, &body); err != nil {
				t.Fatal(err)
			}
			if !ids[body] {
				t.Fatalf("closure body %s is absent from export", body)
			}
			references++
		}
	}
	if references != 2 {
		t.Fatalf("closure body references=%d want 2", references)
	}
}
