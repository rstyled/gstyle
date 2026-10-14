package patterncheck

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestPatternsNegativeIndexRegression(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example\n\ngo 1.26\n",
		"sample.go": `package example
func Unsafe(s []int, i int) int { if i >= len(s) { return -1 }; return s[i] }
func Complete(s []int, i int) int { if i < 0 || i >= len(s) { return -1 }; return s[i] }
func Unsigned(s [2]int, i uint) int { if i < uint(len(s)) { return s[i] }; return -1 }
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
		if candidate.Kind == patterns.NegativeIndex {
			found = append(found, candidate)
		}
	}
	if len(found) != 1 || found[0].Sites[0].Name != "example.Unsafe" {
		t.Fatalf("want only the signed unchecked access, got %+v", found)
	}
}
