package gopatterns

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardRewritePreservesDeclarationScopes(t *testing.T) {
	for _, declaration := range []string{"x:=2", "var x=2", "const x=2", "type x=int;var local x;_=local"} {
		t.Run(declaration, func(t *testing.T) {
			use := "_=x"
			if strings.HasPrefix(declaration, "type ") {
				use = "var another x;_=another"
			}
			source := "package scope\nfunc outer(c bool)int{x:=7;if c{" + declaration + ";" + use + "}else{return 0};return x}\nfunc later(c bool)int{if c{" + declaration + ";" + use + "}else{return 0};x:=7;return x}\n"
			output := fixSource(t, source)
			if strings.Contains(output, "} else {") {
				t.Fatalf("guard not rewritten: %s", output)
			}
			root := t.TempDir()
			files := map[string]string{"go.mod": "module scope\n\ngo 1.24\n", "scope.go": source, "scope_test.go": `package scope
import "testing"
func TestResults(t *testing.T){for _,c:=range []bool{false,true}{want:=0;if c{want=7};if outer(c)!=want||later(c)!=want{t.Fatal("declaration scope changes")}}}
`}
			for name, text := range files {
				if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			run := func() {
				t.Helper()
				command := exec.Command("go", "test", "-count=1", "./...")
				command.Dir = root
				if out, err := command.CombinedOutput(); err != nil {
					t.Fatalf("native scope check: %v\n%s", err, out)
				}
			}
			run()
			if err := os.WriteFile(filepath.Join(root, "scope.go"), []byte(output), 0600); err != nil {
				t.Fatal(err)
			}
			run()
			if second := fixSource(t, output); second != output {
				t.Fatal("scope-preserving rewrite is not idempotent")
			}
		})
	}
}
