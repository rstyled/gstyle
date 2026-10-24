package main

import (
	"context"
	"github.com/rstyled/gstyle/domain/runtimeguard"
	"strings"
	"testing"
)

func TestCollectProductionRanges(t *testing.T) {
	selected := []suite{{Name: "small", Leaves: []expression{{Input: true}, {Value: 1}}, Tests: []condition{{"<", 0}}, Depth: 1}}
	cases, source := corpus(selected)
	got, err := collect(context.Background(), cases, source, representativeInputs(selected))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		for _, x := range []int64{-9223372036854775808, -1, 0, 1, 9223372036854775807} {
			result := c.Program.eval(x)
			if c.Range.HasLower && result < c.Range.Lower || c.Range.HasUpper && result > c.Range.Upper {
				t.Fatalf("%s input %d result %d range %+v", c.Name, x, result, c.Range)
			}
		}
	}
}

func TestRuntimeRejectsWrongSourceMeaning(t *testing.T) {
	selected := []suite{{Name: "small", Leaves: []expression{{Input: true}}}}
	cases, source := corpus(selected)
	source = strings.Replace(source, "result = x", "result = 0", 1)
	_, err := collect(context.Background(), cases, source, representativeInputs(selected))
	if err == nil || !strings.Contains(err.Error(), "disagrees with source semantics") {
		t.Fatalf("wrong source meaning accepted: %v", err)
	}
}

func TestMissingProgramReportIsRejected(t *testing.T) {
	cases, _ := corpus([]suite{{Name: "small", Leaves: []expression{{Input: true}}}})
	if _, err := attachReports(cases, map[string]runtimeguard.Value{}); err == nil {
		t.Fatal("missing report accepted")
	}
}
