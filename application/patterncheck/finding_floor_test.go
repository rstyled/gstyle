package patterncheck

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/patterns"
)

func TestPartialHelperMatchDoesNotBecomeAFinding(t *testing.T) {
	root := t.TempDir()
	source := `package example
// Keep error handling reachable so the fixture tests partial helper matching.
var fixtureErr error
func First(x int)(int,error){value,err:=left(x);if err!=nil{return 0,err};return len(value),nil}
func Second(x int)(int,error){value,err:=right(x);if err!=nil{return 0,err};return len(value)+1,nil}
func left(int)(string,error){return "a",fixtureErr}
func right(int)(string,error){return "bb",fixtureErr}
`
	for name, text := range map[string]string{"go.mod": "module example\n\ngo 1.26\n", "p.go": source} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	extraction, err := gopatterns.Extract(context.Background(), "", []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	raw := patterns.Run(toFacts(extraction.Funcs), patterns.Options{})
	matches := 0
	for _, c := range raw.Candidates {
		if c.Kind == patterns.Parameterize {
			matches++
		}
	}
	if matches == 0 {
		t.Fatal("fixture must produce a partial graph match")
	}
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range report.Candidates {
		if c.Kind == patterns.Parameterize {
			t.Fatalf("unproved helper proposal: %+v", c)
		}
	}
	if len(report.SignatureGroups) != 0 || len(report.Suppressed) != 0 {
		t.Fatal("internal matching groups leaked into the public report")
	}
}
