package gopatterns

import (
	"fmt"
	"go/ast"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestTypeSwitchArgumentBehavior(t *testing.T) {
	for _, test := range []struct {
		name, argument string
		safe           bool
	}{
		{"address", "&v", false},
		{"field", "v.Name", false},
		{"nested call", "identity(&v)", false},
		{"closure capture", "func() any { return &v }()", false},
		{"method value", "v.Speak", false},
		{"composite literal", "[]any{&v}", false},
		{"direct value", "v", false},
		{"shadowed name conservatively skipped", `func(v string) string { return v }("same")`, false},
		{"literal containing name", `"v"`, true},
		{"unrelated variable", "value", true},
		{"unrelated closure", "func() string { return value }()", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`package main
import "fmt"
type Dog struct{ Name string }
type Cat struct{ Name string }
func (Dog) Speak(args ...any) { fmt.Printf("dog:%%T\n", args[0]) }
func (Cat) Speak(args ...any) { fmt.Printf("cat:%%T\n", args[0]) }
func identity(x any) any { return x }
func speak(x any, value string) {
 switch v := x.(type) {
 case Dog: v.Speak(%s)
 case Cat: v.Speak(%s)
 }
}
func main() {
 for _, x := range []any{Dog{"dog"}, Cat{"cat"}, nil, "other"} {
  speak(x, "external")
 }
}
`, test.argument, test.argument))
			pkg := loadTestPackage(t, string(source))
			fn := findFunc(pkg.Syntax[0], "speak")
			hits := findTypeSwitches(fn, pkg.Fset, pkg.TypesInfo)
			if test.safe && len(hits) != 1 {
				t.Errorf("safe call not offered: %+v", hits)
			}
			if !test.safe && len(hits) != 0 {
				t.Errorf("case-bound argument offered: %+v", hits)
			}
			sw := fn.Body.List[0].(*ast.TypeSwitchStmt)
			call, _ := singleCallExpr(sw.Body.List[0].(*ast.CaseClause).Body)
			spec := &patterns.FixSpec{Kind: patterns.TypeSwitch, File: "main.go", Line: pkg.Fset.Position(sw.Pos()).Line,
				Params: map[string]string{"bound": "v", "expr": "x", "method": "Speak", "types": "Dog,Cat", "args": argsText(call)}}
			before := runTypeSwitchProgram(t, source)
			fixed, err := applyProjectedFix(t, spec, source)
			if !test.safe {
				if err == nil {
					t.Errorf("accepted case-bound argument; original output=%q\n%s", before, fixed)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if after := runTypeSwitchProgram(t, fixed); after != before {
				t.Errorf("behavior changed: before=%q after=%q\n%s", before, after, fixed)
			}
		})
	}
}
