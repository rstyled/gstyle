package goquality

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestCurrentSourceInitializerCallsStayInTheirBody(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"domain/patterns/normalize.go", "domain/patterns/report.go"})
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, function := range facts {
		if !strings.Contains(function.Name, ".<literal@") {
			continue
		}
		checked++
		last := function.Line + strings.Count(function.Source, "\n")
		for _, call := range function.Calls {
			if call.Line < function.Line || call.Line > last {
				t.Errorf("%s:%d owns lines %d..%d, but receives %s at line %d", function.Path, function.Line, function.Line, last, call.Callee, call.Line)
			}
		}
		if function.Path == "domain/patterns/normalize.go" && len(function.Calls) != 0 {
			t.Errorf("setter has no calls: %+v", function.Calls)
		}
	}
	if checked < 2 {
		t.Fatalf("initializer corpus missing: checked %d", checked)
	}
}

func TestInitializerAttachmentKeepsPolicyAtItsOwnCall(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "fmt"
var first = func(){fmt.Println("io")}
var second = func(){fail()}
func fail(){panic("retained")}
`))
	if err := os.WriteFile(filepath.Join(root, "other.go"), []byte("package repro\n\nvar third = func(){}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	config := quality.DefaultConfig()
	config.MinStatements = 1
	config.Weights.Panic = 2
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.fail", Kind: quality.Network})
	report, err := quality.Evaluate(facts, config)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	replayed, err := quality.Evaluate(restored, config)
	if err != nil || !reflect.DeepEqual(report, replayed) {
		t.Fatal("initializer policy changes after replay", err)
	}
	checked := 0
	for _, function := range report.Functions {
		if !strings.Contains(function.Name, ".<literal@") {
			continue
		}
		checked++
		if function.Path == "other.go" {
			if len(function.Effects) != 0 {
				t.Errorf("empty literal at the same line and column receives effects: %+v", function.Effects)
			}
			continue
		}
		want := quality.IO
		if function.Line == 4 {
			want = quality.Network
		}
		if len(function.Effects) != 1 || function.Effects[0].Kind != want || function.Effects[0].Line != function.Line {
			t.Errorf("initializer at %d loses its own policy effect %s: %+v", function.Line, want, function.Effects)
		}
	}
	if checked != 3 {
		t.Fatalf("initializer count: %d, want 3", checked)
	}
	defaults, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, function := range defaults.Functions {
		if function.Path != "input.go" || function.Line != 4 {
			continue
		}
		if len(function.Effects) != 1 || function.Effects[0].Kind != quality.Panic || function.Effects[0].Line != 4 {
			t.Fatalf("initializer loses its inherited panic: %+v", function.Effects)
		}
	}
}
