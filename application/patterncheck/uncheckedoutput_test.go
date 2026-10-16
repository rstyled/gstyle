package patterncheck

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestPatternsUncheckedOutput(t *testing.T) {
	root := t.TempDir()
	source := `package example
import("io";f "fmt";"bytes";"strings")
func Failure(w io.Writer, err error) error {if err != nil {f.Fprintln(w,err);return err};return nil}
func NestedFailure(w io.Writer, err error) (any,error) {if err != nil {if w != nil {f.Fprintln(w,err)};return nil,err};return nil,nil}
func Dropped(w io.Writer){f.Fprintln(w,"lost")}
func Blank(w io.Writer){_,_=f.Fprintf(w,"lost")}
func CountOnly(w io.Writer)int{n,_:=f.Fprint(w,"lost");return n}
func Generic[W io.Writer](w W){f.Fprintln(w,"generic")}
func Checked(w io.Writer)error{_,err:=f.Fprintln(w,"ok");return err}
func Buffer(w *bytes.Buffer){f.Fprintln(w,"ok")}
func Builder(w *strings.Builder){f.Fprintln(w,"ok")}
type Memory = strings.Builder
func AliasedBuffer(w *Memory){f.Fprintln(w,"ok")}
// gstyle-allow(unchecked_output): best effort diagnostic
func Allowed(w io.Writer){f.Fprintln(w,"ok")}
//lint:ignore unchecked_output best effort diagnostic
func LintAllowed(w io.Writer){f.Fprintln(w,"ok")}
type Printer struct{}
func(Printer) Fprintln(w io.Writer,s string)(int,error){return w.Write([]byte(s))}
func Shadow(w io.Writer){f:=Printer{};f.Fprintln(w,"local")}
func Closure(w io.Writer){_ = func(){f.Fprintln(w,"not attributed to outer")}}
`
	for name, text := range map[string]string{"go.mod": "module example\n\ngo 1.26\n", "p.go": source} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	for range 2 {
		report, err := GetReport(context.Background(), []string{"."})
		if err != nil {
			t.Fatal(err)
		}
		found := map[string]bool{}
		for _, c := range report.Candidates {
			if c.Kind != patterns.UncheckedOutput {
				continue
			}
			if c.FixSpec != nil || len(c.CounterEvidence) == 0 {
				t.Fatalf("advisory contract missing: %+v", c)
			}
			found[c.Sites[0].Name] = true
		}
		if len(found) != 4 || !found["example.Dropped"] || !found["example.Blank"] || !found["example.CountOnly"] || !found["example.Generic"] {
			t.Fatalf("unexpected findings: %v", found)
		}
	}
}
