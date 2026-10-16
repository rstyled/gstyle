package patterncheck

import (
	"context"
	"github.com/rstyled/gstyle/domain/patterns"
	"os/exec"
	"strings"
	"testing"
)

func TestExecutableNilErrPrecisionCorpus(t *testing.T) {
	t.Chdir("testdata/nilerr_precision")
	if out, err := exec.Command("go", "test", "-p", "1", "-count=1", "./...").CombinedOutput(); err != nil {
		t.Fatalf("runtime oracle: %v\n%s", err, out)
	}
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, candidate := range report.Candidates {
		if candidate.Kind != patterns.NilErr {
			continue
		}
		for _, site := range candidate.Sites {
			counts[site.Name[strings.LastIndex(site.Name, ".")+1:]]++
		}
	}
	for _, name := range []string{"MobyBad", "LoggedBad", "FieldBad", "SiblingBad", "ZeroAggregateBad", "NamedNilBad", "LocalNilBad", "RedundantCheckBad", "InverseBad"} {
		if counts[name] != 1 {
			t.Errorf("%s findings=%d,want one", name, counts[name])
		}
		delete(counts, name)
	}
	if len(counts) != 0 {
		t.Fatalf("valid or unknown handling reported: %v", counts)
	}
}
