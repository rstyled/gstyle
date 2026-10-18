package gopatterns

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeUnwindValueEvidence(t *testing.T) { verifyNativeValueCases(t, unwindValueCases()) }

func verifyNativeValueCases(t *testing.T, cases []unwindValueCase) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(path, []byte(nativeValueProgram(t, cases)), 0644); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("go", "run", path).CombinedOutput()
	if err != nil {
		t.Fatalf("native execution: %v\n%s", err, output)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != len(cases) {
		t.Fatalf("native traces=%d want %d", len(lines), len(cases))
	}
	for i, line := range lines {
		var index int
		var outcome uint16
		if _, err := fmt.Sscanf(line, "%d %d", &index, &outcome); err != nil || index != i {
			t.Fatalf("trace %q: %v", line, err)
		}
		if uint16(cases[i].want)&outcome == 0 {
			t.Fatalf("native %s produces %d outside predicted %d", cases[i].name, outcome, cases[i].want)
		}
	}
}

func nativeValueProgram(t *testing.T, cases []unwindValueCase) string {
	t.Helper()
	var source bytes.Buffer
	source.WriteString(`package main
import("fmt";"runtime")
var choose bool
var _ = runtime.Goexit
func nativeRun(fn func())uint16{
 result:=make(chan uint16,1)
 go func(){completed:=false;defer func(){mode:=uint16(16);if recover()!=nil{mode=4}else if completed{mode=1};result<-mode}();fn();completed=true}()
 return <-result
}
`)
	for i, tc := range cases {
		file, err := parser.ParseFile(token.NewFileSet(), "case.go", fmt.Sprintf("package main;func nativeCase%d(){%s};%s", i, tc.body, tc.helper), 0)
		if err != nil {
			t.Fatal(err)
		}
		renamed := map[string]string{}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				t.Fatal("native fixture contains non-function declaration")
			}
			if !strings.HasPrefix(fn.Name.Name, "nativeCase") {
				renamed[fn.Name.Name] = fmt.Sprintf("native%d_%s", i, fn.Name.Name)
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			id, ok := node.(*ast.Ident)
			if ok && renamed[id.Name] != "" {
				id.Name = renamed[id.Name]
			}
			return true
		})
		for _, decl := range file.Decls {
			if err := format.Node(&source, token.NewFileSet(), decl); err != nil {
				t.Fatal(err)
			}
			source.WriteByte('\n')
		}
	}
	source.WriteString("func main(){\n")
	for i := range cases {
		fmt.Fprintf(&source, "fmt.Printf(\"%%d %%d\\n\",%d,nativeRun(nativeCase%d))\n", i, i)
	}
	source.WriteString("}\n")
	return source.String()
}
