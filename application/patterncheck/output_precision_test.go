package patterncheck

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestExecutableOutputPrecisionCorpus(t *testing.T) {
	t.Chdir("testdata/output_precision")
	if out, err := exec.Command("go", "test", "-p", "1", "-count=1", "./...").CombinedOutput(); err != nil {
		t.Fatalf("runtime oracle: %v\n%s", err, out)
	}
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, c := range report.Candidates {
		if c.Kind == patterns.UncheckedOutput {
			for _, site := range c.Sites {
				counts[site.Name[strings.LastIndex(site.Name, ".")+1:]]++
			}
		}
	}
	if len(counts) != 0 {
		t.Fatalf("runtime write failure does not establish required output: %v", counts)
	}
}
