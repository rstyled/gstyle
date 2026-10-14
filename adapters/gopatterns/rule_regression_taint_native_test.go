package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/patterns"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditNativeSQLArguments(t *testing.T) {
	for _, tc := range []struct {
		name, call string
		want       int
	}{
		{"unsafe query", `db.Exec(input)`, 1},
		{"safe parameter", `db.Exec("SELECT ?",input)`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			os.WriteFile(filepath.Join(root, "go.mod"), []byte("module audit\n\ngo 1.26\n"), 0600)
			source := `package audit;import("os";"database/sql");func f(db *sql.DB){input:=os.Getenv("QUERY");_,_=` + tc.call + `}`
			os.WriteFile(filepath.Join(root, "p.go"), []byte(source), 0600)
			ext, err := (&FixExtractor{}).Extract(context.Background(), root, []string{"."}, SourceOverlay{})
			if err != nil {
				t.Fatal(err)
			}
			var facts []*patterns.MiningFacts
			for i := range ext.Funcs {
				f := &ext.Funcs[i]
				facts = append(facts, &patterns.MiningFacts{ID: f.Name, Name: f.Name, Canonical: &f.Pdg})
			}
			flows := patterns.FindTaintFlows(facts)
			if len(flows) != tc.want {
				t.Fatalf("got %+v,want %d SQL flows", flows, tc.want)
			}
		})
	}
}
