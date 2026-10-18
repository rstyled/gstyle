package gopatterns

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
)

func TestCorpusFalsePositives(t *testing.T) {
	pkg := loadTestPackage(t, corpusRegressionSource)
	root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
	functions, err := packagePdgs(pkg, root, pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]map[patterns.CandidateKind]int{
		"LoopClose": {}, "InfiniteClose": {patterns.Obligation: 1},
		"SafeClose": {}, "ReadAfterOpen": {}, "PayloadIsNotPath": {}, "DiscardedEnvironment": {},
		"BlockedClose": {patterns.Obligation: 1}, "MissingClose": {patterns.Obligation: 1},
		"RealCommandFlow": {patterns.TaintFlowKind: 1},
		"HTTPBody":        {}, "EnvironmentGate": {}, "FlagsAreNotExecutable": {}, "RealPathFlow": {patterns.TaintFlowKind: 1}, "OtherBody": {patterns.Obligation: 1}, "ConditionalClose": {patterns.Obligation: 1}, "UnknownDeferred": {patterns.Obligation: 1},
	}
	for _, function := range functions {
		name := function.Name[strings.LastIndex(function.Name, ".")+1:]
		expected, ok := want[name]
		if !ok {
			t.Fatalf("unexpected function %s", name)
		}
		fact := &patterns.FuncFacts{ID: function.Name, Name: function.Name, Pdg: &function.Pdg, ReleasedFileLines: function.ReleasedFileLines, DeferredReleaseLines: function.DeferredReleaseLines}
		got := map[patterns.CandidateKind]int{}
		for _, candidate := range patterns.Run([]*patterns.FuncFacts{fact}, patterns.Options{}).Candidates {
			if candidate.Kind == patterns.Obligation || candidate.Kind == patterns.TaintFlowKind {
				got[candidate.Kind]++
			}
		}
		for _, kind := range []patterns.CandidateKind{patterns.Obligation, patterns.TaintFlowKind} {
			if got[kind] != expected[kind] {
				t.Errorf("%s %s: got %d want %d", name, kind, got[kind], expected[kind])
			}
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("missing functions: %v", want)
	}
}

func TestDeferredPresenceWithMissingOrderingEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, cleanup string
		want          bool
	}{
		{"saved receiver", "defer f.Close()", true},
		{"later abort", "defer f.Close();defer os.Exit(0)", false},
		{"different receiver", "defer other.Close()", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := parseIR(t, `package p;import "os"
func caller(path string,other *os.File,xs []int,index int)error{
 f,err:=os.Open(path);if err!=nil{return err};`+tc.cleanup+`
 consume(f,xs[index]);return nil
};func consume(*os.File,int){}`, pdg.NewBuilder())
			var call pdg.Ref
			for i, node := range graph.Nodes {
				attrs := graph.NodeAttributes(node)
				if graph.Tables.Text(attrs.Callee.Value) == "os.Open" {
					call = pdg.Ref(i + 1)
				}
				if graph.Tables.Text(attrs.Operator.Value) != "index" {
					continue
				}
				set := graph.Attributes[node.Attributes-1]
				graph.AttributeTables.Extensions[set.Extensions-1] = slices.DeleteFunc(slices.Clone(attrs.Extensions), func(e pdg.Extension) bool { return graph.Tables.Text(e.Name) == "gstyle.go:evaluation-region" })
			}
			if call == 0 {
				t.Fatal("missing acquisition")
			}
			query, err := resourceflow.New(&graph)
			if err != nil {
				t.Fatal(err)
			}
			got := query.RecognizesDeferredCleanup(call, resourceflow.CleanupRule{Method: "Close"}, nil)
			if got != tc.want {
				t.Fatalf("recognized=%v want %v", got, tc.want)
			}
		})
	}
}

const corpusRegressionSource = `package corpusrepro

import (
 "bufio"
 "encoding/json"
 "io"
 "net/http"
 "os"
 "os/exec"
 "path/filepath"
)

func SafeClose(path string) error {
 f, err := os.Open(path)
 if err != nil { return err }
 defer f.Close()
 var value any
 return json.NewDecoder(f).Decode(&value)
}
func BlockedClose(path string) error {
 f, err := os.Open(path)
 if err != nil { return err }
 defer f.Close()
 defer os.Exit(0)
 return nil
}
func MissingClose(path string) error {
 f, err := os.Open(path)
 if err != nil { return err }
 var value any
 return json.NewDecoder(f).Decode(&value)
}
func DiscardedEnvironment() error {
 os.Getenv("GITHUB_SHA")
 return exec.Command("git", "status").Run()
}
func PayloadIsNotPath(input io.Reader, dir string) error {
 data, err := io.ReadAll(input)
 if err != nil { return err }
 path := filepath.Join(dir, "fixed-data")
 return os.WriteFile(path, data, 0600)
}
func ReadAfterOpen(path string) ([]byte, error) {
 f, err := os.Open(path)
 if err != nil { return nil, err }
 defer f.Close()
 return io.ReadAll(f)
}
func RealCommandFlow() error {
 return exec.Command(os.Getenv("EXECUTABLE")).Run()
}

func HTTPBody(url string) ([]byte,error) {
 resp,err:=http.Get(url); if err!=nil{return nil,err};defer resp.Body.Close()
 return io.ReadAll(resp.Body)
}
func OtherBody(url string, other *http.Response) ([]byte,error) {
 resp,err:=http.Get(url); if err!=nil{return nil,err};defer other.Body.Close()
 return io.ReadAll(resp.Body)
}
func ConditionalClose(path string, choose bool) error {
 f,err:=os.Open(path);if err!=nil{return err};if choose {defer f.Close()};return nil
}
func UnknownDeferred(path string, stop func()) error {
 f,err:=os.Open(path);if err!=nil{return err};defer f.Close();defer stop();return nil
}

func EnvironmentGate() error {
 if os.Getenv("GITHUB_REPOSITORY") != "" {return nil}
 return exec.Command("git","status").Run()
}
func FlagsAreNotExecutable(runtimePath string) error {
 options:=struct{NoPivotRoot bool}{NoPivotRoot:os.Getenv("DOCKER_RAMDISK")!=""}
 _=json.NewEncoder(io.Discard).Encode(options)
 return exec.Command(runtimePath,"features").Run()
}
func RealPathFlow(input io.Reader) error {
 data,err:=io.ReadAll(input);if err!=nil{return err}
 return os.WriteFile(string(data),nil,0600)
}

func LoopClose(paths []string) error {
 for _, path:=range paths {
  f,err:=os.Open(path);if err!=nil{return err};defer f.Close()
  scanner:=bufio.NewScanner(f)
  for scanner.Scan() { line:=scanner.Text(); if len(line)>1 { _=line[len(line)-1:] } }
  if err:=scanner.Err();err!=nil{return err}
 }
 return nil
}
func InfiniteClose(path string) error {
 f,err:=os.Open(path);if err!=nil{return err};defer f.Close();for{}
}
`
