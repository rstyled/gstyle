package gopatterns

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestTypeSwitchPointerBehavior(t *testing.T) {
	for _, test := range []struct {
		name, first, second, dogReceiver, catReceiver string
		safe                                          bool
	}{
		{"value cases", "Dog", "Cat", "Dog", "Cat", true},
		{"pointer cases value methods", "*Dog", "*Cat", "Dog", "Cat", true},
		{"pointer cases pointer methods", "*Dog", "*Cat", "*Dog", "*Cat", true},
		{"mixed cases", "*Dog", "Cat", "*Dog", "Cat", true},
		{"same base type", "Dog", "*Dog", "Dog", "Cat", true},
		{"value cases pointer methods", "Dog", "Cat", "*Dog", "*Cat", false},
		{"mixed implicit address", "*Dog", "Cat", "Dog", "*Cat", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`package main
type Dog struct{}
type Cat struct{}
func (%s) Speak(){println("dog")}
func (%s) Speak(){println("cat")}
func speak(x any){
 switch v:=x.(type){
 case %s:v.Speak()
 case %s:v.Speak()
 }
}
func probe(x any){defer func(){if recover()!=nil{println("panic")}}();speak(x)}
func main(){probe(Dog{});probe(&Dog{});probe(Cat{});probe(&Cat{});probe((*Dog)(nil));probe((*Cat)(nil));probe(nil);probe("other")}
`, test.dogReceiver, test.catReceiver, test.first, test.second))
			pkg := loadTestPackage(t, string(source))
			fn := findFunc(pkg.Syntax[0], "speak")
			hits := findTypeSwitches(fn, pkg.Fset, pkg.TypesInfo)
			if test.safe && (len(hits) != 1 || strings.Join(hits[0].Types, ",") != test.first+","+test.second) {
				t.Errorf("lost case types: %+v", hits)
			}
			if !test.safe && len(hits) != 0 {
				t.Errorf("unsafe method set offered: %+v", hits)
			}
			spec := &patterns.FixSpec{Kind: patterns.TypeSwitch, File: "main.go", Line: pkg.Fset.Position(fn.Body.List[0].Pos()).Line,
				Params: map[string]string{"bound": "v", "expr": "x", "method": "Speak", "types": test.first + "," + test.second, "args": ""}}
			before := runTypeSwitchProgram(t, source)
			fixed, err := applyProjectedFix(t, spec, source)
			if !test.safe {
				if err == nil {
					t.Errorf("accepted implicit address-taking; before=%q after=%q", before, runTypeSwitchProgram(t, fixed))
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

func runTypeSwitchProgram(t *testing.T, source []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("go", "run", path).CombinedOutput()
	if err != nil {
		t.Fatalf("run type switch: %v\n%s", err, output)
	}
	return string(output)
}

func TestTypeSwitchRejectsCaseSetThatLostPointers(t *testing.T) {
	source := []byte(`package p
type Dog struct{}
type Cat struct{}
func(Dog)Speak(){}
func(Cat)Speak(){}
func f(x any){switch v:=x.(type){case *Dog:v.Speak();case *Cat:v.Speak()}}`)
	spec := &patterns.FixSpec{Kind: patterns.TypeSwitch, File: "main.go", Line: 6,
		Params: map[string]string{"bound": "v", "expr": "x", "method": "Speak", "types": "Dog,Cat", "args": ""}}
	if _, err := applyProjectedFix(t, spec, source); err == nil {
		t.Fatal("accepted a stale case set that omits pointers")
	}
}
