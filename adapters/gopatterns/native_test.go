package gopatterns

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/adapters/govalidation"
	"github.com/rstyled/gstyle/domain/ccgraph"
	"github.com/rstyled/gstyle/domain/pdg"
)

func nativeFixture(t *testing.T) *NativeMatching {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/native\n\ngo 1.26\n",
		"example.go": `package native
type Count = int
func First(x Count) Count { return x }
func Second(y Count) Count { return y }
func Arithmetic(x int) int { return x+1 }
func Call(x int) int { return First(x) }
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	result, err := ExtractMatching(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestNativeMatchingAdmitsVerifiedSourceAndReportsExclusions(t *testing.T) {
	input := nativeFixture(t)
	if len(input.Functions) != 2 || len(input.Ineligible) != 2 {
		t.Fatalf("eligible=%d exclusions=%v", len(input.Functions), input.Ineligible)
	}
	if input.Build.GOARCH == "" || input.Build.GOOS == "" || input.Build.GOVERSION == "" {
		t.Fatal("unknown build target")
	}
	for _, graph := range input.Graphs {
		if graph.Conformance.Status != pdg.Conformant {
			t.Fatal("validated producer stays nonconformant")
		}
		if err := pdg.Validate(&graph); err != nil {
			t.Fatal(err)
		}
	}
	for _, evidence := range input.Evidence {
		want := [9]uint64{0, 1, 3, 0, 0, 0, 0, 4, 1}
		if evidence.Projection.Features != want {
			t.Fatalf("features=%v; want %v", evidence.Projection.Features, want)
		}
		if evidence.ContentIdentity == "" {
			t.Fatal("source identity omitted")
		}
	}
	accepted := 0
	run, err := ccgraph.Evaluate(context.Background(), input.Functions, ccgraph.DefaultConfig(), func(pair ccgraph.Pair) error {
		if pair.Accepted {
			accepted++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !run.Complete || run.Pairs != 1 || accepted != 1 {
		t.Fatalf("matching run=%+v accepted=%d", run, accepted)
	}
}

func TestNativeContextPreservesSourceAndPairedGraphs(t *testing.T) {
	input := nativeFixture(t)
	var buffer bytes.Buffer
	if err := input.WriteContext(&buffer); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&buffer)
	counts := map[string]int{}
	for {
		var row struct {
			Type     string         `json:"type"`
			Document map[string]any `json:"document"`
		}
		err := decoder.Decode(&row)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		counts[row.Type]++
		if row.Type == "cfg" && row.Document["schema_version"] != "0.2.0" {
			t.Fatal("wrong CFG schema")
		}
		if row.Type == "pdg" && row.Document["schema_version"] != "0.1.0" {
			t.Fatal("wrong PDG schema")
		}
	}
	if counts["source"] != 1 || counts["pdg"] != 2 || counts["cfg"] != 2 {
		t.Fatal(counts)
	}
	if err := os.WriteFile(filepath.Join(input.root, "example.go"), []byte("package native"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := input.WriteContext(io.Discard); err == nil {
		t.Fatal("stale source context is accepted")
	}
}

func TestNativeControlRequiresSemanticDependencies(t *testing.T) {
	seed := nativeFixture(t)
	source := `package native
func Branch(x int,a bool)int{if a{x=1}else{x=2};return x}
func Early(x int,a bool)int{if a{return 1};return x}
func Nested(x int,a,b bool)int{if a{if b{return x}};return 0}
`
	if err := os.WriteFile(filepath.Join(seed.root, "example.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	input, err := ExtractMatching(context.Background(), seed.root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if len(input.Functions) != 3 || len(input.Ineligible) != 0 {
		t.Fatalf("functions=%d excluded=%v", len(input.Functions), input.Ineligible)
	}
	for _, graph := range input.Graphs {
		mutant := graph
		mutant.Edges = nil
		omitted := false
		for _, edge := range graph.Edges {
			if !omitted && edge.Kind == pdg.ControlEdge {
				omitted = true
				continue
			}
			mutant.Edges = append(mutant.Edges, edge)
		}
		if !omitted {
			t.Fatal("fixture has no semantic control edge")
		}
		if err := govalidation.Ordering(&mutant); err == nil {
			t.Fatal("missing semantic dependency passed independent validation")
		}
	}
}
