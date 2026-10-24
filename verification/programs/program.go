// Command programs collects production branch/value results for Lean certificates.
package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type expression struct {
	Input bool  `json:"input"`
	Value int64 `json:"value"`
}

type condition struct {
	Operator string `json:"operator"`
	Bound    int64  `json:"bound"`
}

type program struct {
	Value   expression `json:"value"`
	Test    *condition `json:"test,omitempty"`
	Yes, No *program
}

type suite struct {
	Name   string       `json:"name"`
	Leaves []expression `json:"leaves"`
	Tests  []condition  `json:"tests"`
	Depth  int          `json:"depth"`
}

func suites() []suite {
	leaves := []expression{{Input: true}, {Value: -1}, {}, {Value: 1}}
	var tests []condition
	for _, bound := range []int64{math.MinInt64, -1, 0, 1, math.MaxInt64} {
		for _, operator := range []string{"<", "<=", ">", ">=", "==", "!="} {
			tests = append(tests, condition{operator, bound})
		}
	}
	return []suite{
		{"boundaries", leaves, tests, 1},
		{"nested", []expression{{Input: true}, {}}, []condition{{"<", 0}, {"==", 0}, {">", 0}}, 2},
	}
}

func enumerate(s suite, depth int) []program {
	result := make([]program, 0, len(s.Leaves))
	for _, value := range s.Leaves {
		result = append(result, program{Value: value})
	}
	if depth == 0 {
		return result
	}
	children := enumerate(s, depth-1)
	for _, test := range s.Tests {
		for _, yes := range children {
			for _, no := range children {
				result = append(result, program{Test: &test, Yes: &yes, No: &no})
			}
		}
	}
	return result
}

func (e expression) source() string {
	if e.Input {
		return "x"
	}
	return strconv.FormatInt(e.Value, 10)
}

func (p program) body() string {
	if p.Test == nil {
		return "result = " + p.Value.source() + "\n"
	}
	return "if x " + p.Test.Operator + " " + strconv.FormatInt(p.Test.Bound, 10) + " {\n" +
		p.Yes.body() + "} else {\n" + p.No.body() + "}\n"
}

func (p program) source(name string) string {
	return "func " + name + "(x int64) int64 {\nvar result int64\n_ = *new(int)\n" + p.body() +
		"observe(result)\nreturn result\n}\n"
}

func (p program) eval(x int64) int64 {
	if p.Test != nil {
		return p.chosen(x).eval(x)
	}
	if p.Value.Input {
		return x
	}
	return p.Value.Value
}

func (p program) chosen(x int64) *program {
	if compare(p.Test.Operator, x, p.Test.Bound) {
		return p.Yes
	}
	return p.No
}

func compare(operator string, x, y int64) bool {
	switch operator {
	case "<":
		return x < y
	case "<=":
		return x <= y
	case ">":
		return x > y
	case ">=":
		return x >= y
	case "==":
		return x == y
	default:
		return x != y
	}
}

func (e expression) lean() string {
	if e.Input {
		return ".input"
	}
	return fmt.Sprintf("(.literal (%d))", e.Value)
}

func (c condition) lean() string {
	operators := map[string]string{"<": "lt", "<=": "le", ">": "gt", ">=": "ge", "==": "eq", "!=": "ne"}
	return fmt.Sprintf("⟨.%s, (%d)⟩", operators[c.Operator], c.Bound)
}

func (p program) lean() string {
	if p.Test == nil {
		return "(.result " + p.Value.lean() + ")"
	}
	return "(.branch " + p.Test.lean() + " " + p.Yes.lean() + " " + p.No.lean() + ")"
}

func (s suite) leanPrograms() string {
	leaves := make([]string, 0, len(s.Leaves))
	for _, leaf := range s.Leaves {
		leaves = append(leaves, leaf.lean())
	}
	tests := make([]string, 0, len(s.Tests))
	for _, test := range s.Tests {
		tests = append(tests, test.lean())
	}
	return fmt.Sprintf("enumerate [%s] [%s] %d", strings.Join(leaves, ", "), strings.Join(tests, ", "), s.Depth)
}
