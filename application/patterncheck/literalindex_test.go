package patterncheck

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestPatternsLiteralIndexRegression(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example\n\ngo 1.26\n",
		"sample.go": `package example
const slot = 1
func Constant() string { return [...]string{"a", "b"}[slot] }
func Unsafe(i int) string { i = 2; return [...]string{"a", "b"}[i] }
func Unknown(i int) string { return [...]string{"a", "b"}[i] }
func Complete(i int) string { if i < 0 || i >= 2 { return "" }; return [...]string{"a", "b"}[i] }
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
		if candidate.Kind == patterns.LiteralIndex {
			found = append(found, candidate)
		}
	}
	if len(found) != 1 || found[0].Sites[0].Name != "example.Unsafe" {
		t.Fatalf("want only the incomplete bounds check, got %+v", found)
	}
}
