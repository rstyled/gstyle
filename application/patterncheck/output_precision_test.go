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
	for _, name := range []string{"OutputDirect", "OutputLocal", "OutputBranch", "OutputBoxedNil"} {
		if counts[name+"Bad"] != 1 {
			t.Errorf("%sBad findings=%d,want one", name, counts[name+"Bad"])
		}
		if counts[name+"Safe"] != 0 {
			t.Errorf("%sSafe findings=%d,want zero", name, counts[name+"Safe"])
		}
	}
	for _, name := range []string{"OutputPanicSafe", "OutputExistingFailureSafe", "OutputFalseSafe", "OutputContradictionSafe", "OutputClearedInterfaceSafe", "OutputZeroInterfaceSafe", "OutputNilGuardWriterSafe", "OutputLiteralNilWriterSafe", "OutputDeclaredNilWriterSafe", "OutputAssignedNilWriterSafe", "OutputReturnPanicSafe", "OutputReturnIndexPanicSafe"} {
		if counts[name] != 0 {
			t.Errorf("%s findings=%d,want zero", name, counts[name])
		}
	}
}
