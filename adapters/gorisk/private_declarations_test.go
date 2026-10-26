package gorisk

import (
	"context"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func TestPrivateHelpersRequireCompiledBodiesAndStableExistingSignatures(t *testing.T) {
	const source = "package sample\nfunc Count(n int) int { return helper(n) }\nfunc helper(n int) int { return n+1 }\n"
	for _, test := range []struct {
		name, after string
		want        changerisk.Level
	}{
		{"added helper", source + "func extra(n int) int { return n+2 }\n", changerisk.Low},
		{"body change", strings.Replace(source, "n+1", "n+2", 1), changerisk.Low},
		{"changed signature", strings.ReplaceAll(source, "helper(n)", "helper(n, 1)") + "", changerisk.Unknown},
		{"removed helper", "package sample\nfunc Count(n int) int { return n+1 }\n", changerisk.Unknown},
		{"added init", source + "func init() {}\n", changerisk.Unknown},
		{"added variable", source + "var value = helper(1)\n", changerisk.Unknown},
		{"directive", source + "//go:noinline\nfunc extra(n int) int { return n+2 }\n", changerisk.Unknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			after := test.after
			if test.name == "changed signature" {
				after = strings.Replace(after, "func helper(n int)", "func helper(n, extra int)", 1)
			}
			root := sourceModule(t, map[string]string{"source.go": source})
			base, err := ExtractSource(context.Background(), root, []string{"source.go"})
			if err != nil {
				t.Fatal(err)
			}
			coverageWrite(t, root, "source.go", after)
			head, err := ExtractSource(context.Background(), root, []string{"source.go"})
			if err != nil {
				t.Fatal(err)
			}
			report := changerisk.Evaluate(base, head, changerisk.Config{PDGChange: changerisk.Medium, PDGLowRiskMaxFunctions: 3}, changerisk.APIReport{})
			if report.Risk != test.want {
				t.Fatalf("risk=%s want=%s: %+v", report.Risk, test.want, report.Reasons)
			}
		})
	}
}
