package patternfix

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrossFileParameterizationRunsThroughPhasedCLI(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/parallel\n\ngo 1.24\n",
		"first.go": `package parallel
// First has a stable public entry point.
func First(x int)(int,error){
 value,err:=left(x)
 if err!=nil{return 0,err}
 return len(value),nil
}`,
		"second.go": `package parallel
// Second has a stable public entry point.
func Second(x int)(int,error){
 value,err:=right(x)
 if err!=nil{return 0,err}
 return len(value),nil
}`,
		"calls.go": `package parallel
func left(int)(string,error){return "a",nil}
func right(int)(string,error){return "bb",nil}`,
		"behavior_test.go": `package parallel
import "testing"
func TestValues(t *testing.T){a,e:=First(1);b,f:=Second(1);if a!=1 || b!=2 || e!=nil || f!=nil{t.Fatal(a,b,e,f)}}`,
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	defer chdir(t, root)()
	var preview, applied, repeat bytes.Buffer
	if err := Run(context.Background(), []string{"--kind", "all", "."}, &preview); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.String(), "(parameterize)") {
		t.Fatalf("proof not connected to CLI: %s", &preview)
	}
	for name, source := range files {
		before, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(before) != source {
			t.Fatalf("preview changed %s", name)
		}
	}
	if err := Run(context.Background(), []string{"--kind", "all", "--apply", "."}, &applied); err != nil {
		t.Fatal(err)
	}
	for name, source := range files {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		path := filepath.Join(root, name)
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := unifiedDiff(path, []byte(source), after)
		if err != nil {
			t.Fatal(err)
		}
		if expected != "" && !strings.Contains(preview.String(), expected) {
			t.Fatalf("preview differs from apply for %s", name)
		}
	}
	command := exec.Command("go", "test", "-count=1", "./...")
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("behavior changed: %v\n%s", err, out)
	}
	if err := Run(context.Background(), []string{"--kind", "all", "."}, &repeat); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(repeat.String(), "0 fixable candidate(s)") {
		t.Fatalf("second run still changes source: %s", &repeat)
	}
}
