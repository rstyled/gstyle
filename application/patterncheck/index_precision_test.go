package patterncheck

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestExecutableIndexPrecisionCorpus(t *testing.T) {
	t.Chdir("testdata/index_precision")
	if out, err := exec.Command("go", "test", "-p", "1", "-count=1", "./...").CombinedOutput(); err != nil {
		t.Fatalf("runtime oracle: %v\n%s", err, out)
	}
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, c := range report.Candidates {
		if c.Kind == patterns.NegativeIndex || c.Kind == patterns.LiteralIndex {
			for _, site := range c.Sites {
				counts[site.Name[strings.LastIndex(site.Name, ".")+1:]]++
			}
		}
	}
	for _, name := range []string{"IndexNegative", "IndexHigh", "IndexBranch", "IndexSlice", "IndexConstant", "IndexEmpty", "IndexGeneric"} {
		if counts[name+"Bad"] != 1 {
			t.Errorf("%sBad findings=%d,want exactly one", name, counts[name+"Bad"])
		}
		if counts[name+"Safe"] != 0 {
			t.Errorf("%sSafe findings=%d,want zero", name, counts[name+"Safe"])
		}
	}
	for _, name := range []string{"IndexEmptyNegatedSafe", "IndexEmptyConjunctionSafe", "IndexEmptyDisjunctionSafe"} {
		if counts[name] != 0 {
			t.Errorf("%s findings=%d,want zero", name, counts[name])
		}
	}

}
