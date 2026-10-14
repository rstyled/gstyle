package patterncheck

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestPatternsNilGuardRegression(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example\n\ngo 1.26\n",
		"sample.go": `package example
type Item struct { Value int }
func Unsafe(p, q *Item) int { if q == nil { return 0 }; return p.Value }
func Allocated(q *int) int { if q == nil { return 0 }; p := new(int); return *p }
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	var found []patterns.Candidate
	for _, candidate := range report.Candidates {
		if candidate.Kind == patterns.NilGuard {
			found = append(found, candidate)
		}
	}
	if len(found) != 1 || found[0].Sites[0].Name != "example.Unsafe" {
		t.Fatalf("want only the unsafe field access, got %+v", found)
	}
}
