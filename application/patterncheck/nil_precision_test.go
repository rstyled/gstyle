package patterncheck

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestExecutableNilPrecisionCorpus(t *testing.T) {
	t.Chdir("testdata/nil_precision")
	if out, err := exec.Command("go", "test", "-p", "1", "-count=1", "./...").CombinedOutput(); err != nil {
		t.Fatalf("runtime oracle: %v\n%s", err, out)
	}
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, c := range report.Candidates {
		if c.Kind == patterns.NilGuard {
			for _, site := range c.Sites {
				counts[site.Name[strings.LastIndex(site.Name, ".")+1:]]++
			}
		}
	}
	for _, name := range []string{"NilAssigned", "NilGuard", "NilAlias", "NilBranch"} {
		if counts[name+"Bad"] != 1 {
			t.Errorf("%sBad findings=%d,want one", name, counts[name+"Bad"])
		}
		if counts[name+"Safe"] != 0 {
			t.Errorf("%sSafe findings=%d,want zero", name, counts[name+"Safe"])
		}
	}
}
