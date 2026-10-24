package main

import (
	"context"
	"math"
	"strings"
	"testing"
)

const suppliedExample = `package generated
func observe(value int64) {}
func Choice(x int64) int64 {
var result int64
_ = *new(int)
if x < -42 {
result = -9223372036854775808
} else {
if x >= 117 {
result = x
} else {
result = 23
}
}
observe(result)
return result
}
`

func TestSuppliedSourceMeaningAndProduction(t *testing.T) {
	cases, err := parseSource(suppliedExample)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].Name != "Choice" {
		t.Fatalf("source inventory: %+v", cases)
	}
	inputs := []int64{math.MinInt64, -43, -42, 116, 117, math.MaxInt64}
	expected := []int64{math.MinInt64, math.MinInt64, 23, 23, 117, math.MaxInt64}
	for i, x := range inputs {
		if got := cases[0].Program.eval(x); got != expected[i] {
			t.Fatalf("input %d: got %d, want %d", x, got, expected[i])
		}
	}
	if _, err := collect(context.Background(), cases, suppliedExample, inputs); err != nil {
		t.Fatal(err)
	}
}

func TestSuppliedSourceHasNoEnumerationDepthLimit(t *testing.T) {
	p := program{Value: expression{Input: true}}
	for depth := range 32 {
		yes := p
		no := program{Value: expression{Value: int64(depth)}}
		p = program{Test: &condition{Operator: ">=", Bound: int64(depth)}, Yes: &yes, No: &no}
	}
	source := sourceHeader + p.source("Deep")
	cases, err := parseSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(cases[0].Program.conditions()); got != 32 {
		t.Fatalf("decoded %d comparisons, want 32", got)
	}
}

func TestSuppliedSourceRejectsUnmodeledBehavior(t *testing.T) {
	changes := map[string][2]string{
		"extra write":      {"observe(result)", "result = 99\nobserve(result)"},
		"input write":      {"result = 23", "x = 23"},
		"shadow input":     {"if x < -42", "if x := int64(1); x < -42"},
		"call result":      {"result = 23", "result = unknown()"},
		"overflow":         {"result = 23", "result = 9223372036854775808"},
		"wrong wrapper":    {"return result", "return x"},
		"changed observer": {"func observe(value int64) {}", "func observe(value int64) {panic(1)}"},
		"missing else":     {"} else {\nresult = 23\n}", "}"},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			source := strings.Replace(suppliedExample, change[0], change[1], 1)
			if source == suppliedExample {
				t.Fatal("test mutation did not change source")
			}
			if _, err := parseSource(source); err == nil {
				t.Fatal("unmodeled source accepted")
			}
		})
	}
}
