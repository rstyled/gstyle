package patterncheck

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestNilErrRemainsEnabledWithoutOutputContractInference(t *testing.T) {
	// Moby #53934: a failed privilege fetch returned empty privileges and nil.
	root := t.TempDir()
	source := `package example
import("fmt";"io")
type Privileges struct{}
func LostFetchError(fetch func() error)(Privileges,error){
 if err:=fetch();err!=nil{return Privileges{},nil}
 return Privileges{},nil
}
func PropagatedFetchError(fetch func() error)(Privileges,error){
 if err:=fetch();err!=nil{return Privileges{},err}
 return Privileges{},nil
}
func Status(w io.Writer)error{fmt.Fprintln(w,"done");return nil}
`
	for name, text := range map[string]string{"go.mod": "module example\n\ngo 1.26\n", "p.go": source} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	hits := 0
	for _, candidate := range report.Candidates {
		if candidate.Kind == patterns.UncheckedOutput {
			t.Fatalf("unsupported output inference: %+v", candidate)
		}
		if candidate.Kind != patterns.NilErr {
			continue
		}
		for _, site := range candidate.Sites {
			if site.Name != "example.LostFetchError" {
				t.Fatalf("unexpected nilerr: %+v", site)
			}
			hits++
		}
	}
	if hits != 1 {
		t.Fatalf("Moby-shaped failure findings=%d,want 1", hits)
	}
}
