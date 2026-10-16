package patterncheck

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

// Compare the source-local boolean subset with actual Go execution, rather than
// computing expected paths using the analyzer's own branch transfer functions.
func TestNilErrBooleanRuntimeOracle(t *testing.T) {
	root := t.TempDir()
	source, main := nilErrBooleanPrograms()
	files := map[string]string{"go.mod": "module nilerroracle\n\ngo 1.26\n", "cases.go": source, "cmd/oracle/main.go": main}
	for name, contents := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	out, err := exec.Command("go", "run", "-p", "1", "./cmd/oracle").CombinedOutput()
	if err != nil {
		t.Fatalf("runtime oracle: %v\n%s", err, out)
	}
	expected := map[string]bool{}
	if err := json.Unmarshal(out, &expected); err != nil {
		t.Fatalf("runtime result: %v\n%s", err, out)
	}
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, candidate := range report.Candidates {
		if candidate.Kind != patterns.NilErr {
			continue
		}
		for _, site := range candidate.Sites {
			counts[site.Name[strings.LastIndex(site.Name, ".")+1:]]++
		}
	}
	faults := 0
	for name, lost := range expected {
		want := 0
		if lost {
			want = 1
			faults++
		}
		if counts[name] != want {
			t.Errorf("%s: findings=%d,runtime loss=%v", name, counts[name], lost)
		}
		delete(counts, name)
	}
	if len(expected) != 54 || faults != 18 {
		t.Fatalf("oracle coverage changed: cases=%d,faults=%d", len(expected), faults)
	}
	if len(counts) != 0 {
		t.Fatalf("unexpected findings: %v", counts)
	}
}

func nilErrBooleanPrograms() (string, string) {
	bodies := []string{
		`return nil`, `return err`, `if b{return nil};return err`, `if b{return err};return nil`,
		`if b{return err};if !b{return err};return nil`,
		`if b{c=b;if !c{return nil};return err};return err`,
		`b=c;if b{return err};if !c{return err};return nil`,
		`c=!b;if b{return err};if c{return err};return nil`,
		`a:=b;b=false;if a{return err};if !a{return err};return nil`,
		`a:=b;b=false;if a{return nil};return err`,
		`b,c=c,b;if b{return err};if c{return nil};return err`,
		`if b{if c{return err};return nil};return err`,
		`if b{if c{return nil};return err};if c{return err};return nil`,
		`if b{b=false};if b{return nil};return err`,
		`if b{b=false};if !b{return err};return nil`,
		`if c{b=c}else{b=!c};if !b{return nil};return err`,
		`if false{return nil};return err`, `if true{return err};return nil`,
	}
	source := "package oracle\n"
	var entries []string
	for _, prefix := range []string{"", "c=b;", "b=!c;"} {
		for _, body := range bodies {
			name := fmt.Sprintf("F%d", len(entries))
			source += fmt.Sprintf("func %s(err error,b,c bool)error{%sif err!=nil{%s};return nil}\n", name, prefix, body)
			entries = append(entries, fmt.Sprintf("%q:oracle.%s", name, name))
		}
	}
	main := `package main
import("encoding/json";"errors";"os";oracle "nilerroracle")
func main(){
 cause:=errors.New("failure");out:=map[string]bool{}
 for name,call:=range map[string]func(error,bool,bool)error{` + strings.Join(entries, ",") + `}{
  lost:=false
  for _,b:=range []bool{false,true}{for _,c:=range []bool{false,true}{lost=lost||call(cause,b,c)==nil}}
  out[name]=lost
 }
 if err:=json.NewEncoder(os.Stdout).Encode(out);err!=nil{panic(err)}
}`
	return source, main
}
