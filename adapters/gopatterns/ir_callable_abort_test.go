package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAbortCallbackSkipsOlderDefers(t *testing.T) {
	checkCallableSourceOracle(t, `package example;import "os";func apply(cb func(int)){cb(0)};func f(){defer func(){panic("must not run")}();apply(os.Exit)}`, map[string]callflow.Outcomes{"example.f": callflow.Aborts})
}
func TestNativeAbortCallbackEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.go")
	source := `package main;import("fmt";"os");func apply(cb func(int)){cb(0)};func main(){fmt.Println("before");defer fmt.Println("deferred");apply(os.Exit);fmt.Println("after")}`
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("go", "run", path).CombinedOutput()
	if err != nil {
		t.Fatalf("native abort callback: %v\n%s", err, output)
	}
	if string(output) != "before\n" {
		t.Fatalf("abort callback output=%q", output)
	}
}
