package gopatterns

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestTypeSwitchVariadicBehavior(t *testing.T) {
	for _, test := range []struct {
		name, element, parameters, first, second string
		safe                                     bool
	}{
		{"spread any", "any", "args ...any", "args...", "args...", true},
		{"spread strings", "string", "args ...string", "args...", "args...", true},
		{"fixed and spread", "any", "prefix string, args ...any", `"prefix", args...`, `"prefix", args...`, true},
		{"ordinary slice", "any", "args ...any", "args", "args", true},
		{"mixed spread first", "any", "args ...any", "args...", "args", false},
		{"mixed spread second", "any", "args ...any", "args", "args...", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`package main
import "fmt"
type Dog struct{}
type Cat struct{}
func (Dog) Speak(%s) { fmt.Printf("dog:%%#v\n", args) }
func (Cat) Speak(%s) { fmt.Printf("cat:%%#v\n", args) }
func speak(x any, args []%s) {
 switch v := x.(type) {
 case Dog: v.Speak(%s)
 case Cat: v.Speak(%s)
 }
}
func main() {
 for _, x := range []any{Dog{}, Cat{}, nil, "other"} {
  speak(x, []%s{"hello", "world"})
  speak(x, nil)
  speak(x, []%s{})
 }
}
`, test.parameters, test.parameters, test.element, test.first, test.second, test.element, test.element))
			pkg := loadTestPackage(t, string(source))
			fn := findFunc(pkg.Syntax[0], "speak")
			hits := findTypeSwitches(fn, pkg.Fset, pkg.TypesInfo)
			if test.safe && (len(hits) != 1 || hits[0].Args != test.first) {
				t.Errorf("lost argument form: %+v", hits)
			}
			if !test.safe && len(hits) != 0 {
				t.Errorf("offered mixed argument forms: %+v", hits)
			}
			spec := &patterns.FixSpec{Kind: patterns.TypeSwitch, File: "main.go", Line: pkg.Fset.Position(fn.Body.List[0].Pos()).Line,
				Params: map[string]string{"bound": "v", "expr": "x", "method": "Speak", "types": "Dog,Cat", "args": test.first}}
			fixed, err := applyProjectedFix(t, spec, source)
			if !test.safe {
				if err == nil {
					t.Error("accepted mixed argument forms")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			before := runTypeSwitchProgram(t, source)
			if after := runTypeSwitchProgram(t, fixed); after != before {
				t.Errorf("behavior changed: before=%q after=%q\n%s", before, after, fixed)
			}
			if !strings.HasSuffix(test.first, "...") {
				return
			}
			spec.Params["args"] = strings.TrimSuffix(test.first, "...")
			if _, err := applyProjectedFix(t, spec, source); err == nil {
				t.Error("accepted stale spec without spread")
			}
		})
	}
}
