package patterncheck

import (
	"context"
	"github.com/rstyled/gstyle/domain/patterns"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatternsTaintUsesCanonicalCallableCatalog(t *testing.T) {
	root := t.TempDir()
	source := `package example
import("database/sql";"net/http")
func capture(q string)func()string{return func()string{return q}}
func id(q string)string{return q}
func clean(q string)string{return "SELECT 1"}
func apply(cb func(string)string,q string)string{return cb(q)}
func UnsafeCapture(r *http.Request,db *sql.DB){db.Exec(capture(r.FormValue("q"))())}
func SafeCapture(r *http.Request,db *sql.DB){db.Exec(capture("SELECT 1")())}
func UnsafeCallback(r *http.Request,db *sql.DB){db.Exec(apply(id,r.FormValue("q")))}
func SafeCallback(r *http.Request,db *sql.DB){db.Exec(apply(clean,r.FormValue("q")))}
func InClosure(r *http.Request,db *sql.DB){func(){db.Exec(r.FormValue("q"))}()}
func Parameterized(r *http.Request,db *sql.DB){db.Exec("SELECT ?",capture(r.FormValue("q"))())}
`
	for name, body := range map[string]string{"go.mod": "module example\n\ngo 1.26.0\n", "case.go": source} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	hits := map[string]int{}
	for _, candidate := range report.Candidates {
		if candidate.Kind != patterns.TaintFlowKind {
			continue
		}
		if len(candidate.Sites) != 2 {
			t.Fatalf("source/sink evidence missing: %+v", candidate)
		}
		for _, site := range candidate.Sites {
			if site.Path != "case.go" || site.Line == 0 {
				t.Fatalf("callable source location lost: %+v", site)
			}
		}
		name := candidate.Sites[1].Name
		if strings.HasPrefix(name, "example.InClosure") {
			name = "example.InClosure"
		}
		hits[name]++
	}
	for _, name := range []string{"UnsafeCapture", "UnsafeCallback", "InClosure"} {
		if hits["example."+name] != 1 {
			t.Errorf("%s findings=%d want 1", name, hits["example."+name])
		}
		delete(hits, "example."+name)
	}
	if len(hits) != 0 {
		t.Fatalf("safe case or extra callable warning: %v", hits)
	}
}
