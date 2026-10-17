package gopatterns

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
)

func TestCanonicalFactoryContracts(t *testing.T) {
	source := `package p
import("os";"io/fs")
var choose bool
var other *os.File
func direct(name string)(*os.File,error){return os.Open(name)}
func local(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};return f,nil}
func alias(name string)(*os.File,error){f,err:=os.Open(name);alias:=f;f=nil;_=f;return alias,err}
func loopCopy(name string)(*os.File,error){f,err:=os.Open(name);alias:=f;for i:=0;i<2;i++{alias=alias};return alias,err}
func returnOrClose(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};if choose{return f,nil};return nil,f.Close()}
func closed(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};f.Close();return f,nil}
func replaced(name string)(*os.File,error){f,err:=os.Open(name);f=other;return f,err}
func unreachable(name string)(*os.File,error){return os.Open(name);return other,nil}
func recursive(name string,b bool)(*os.File,error){if b{return os.Open(name)};return recursive(name,!b)}
func cycle(name string)(*os.File,error){return cycle(name)}
func nilOnly(name string)(*os.File,error){return nil,nil}
func mixed(name string,b bool)(*os.File,error){if b{return nilOnly(name)};return os.Open(name)}
func bad(name string,cause error)(*os.File,error){f,_:=os.Open(name);return f,cause}
type problem struct{}
func (*problem) Error()string{return "problem"}
func boxedError(name string)(*os.File,error){f,_:=os.Open(name);var cause *problem;return f,cause}
func boxedCopyError(name string)(*os.File,error){f,_:=os.Open(name);var cause *problem;var err error=cause;return f,err}
func badForward(name string,cause error)(*os.File,error){return bad(name,cause)}
func single(name string)*os.File{f,err:=os.Open(name);if err!=nil{return nil};return f}
func fsSource(files fs.FS,name string)(fs.File,error){return files.Open(name)}
func fsExpression(files fs.FS,name string)(fs.File,error){return fs.FS.Open(files,name)}
type unknownFS interface{Open(string)(fs.File,error)}
func unknown(files unknownFS,name string)(fs.File,error){return files.Open(name)}
func generic[T ~string](name T)(*os.File,error){return os.Open(string(name))}
func instantiated(name string)(*os.File,error){return generic[string](name)}
func printer(name string)(*os.File,error){f,err:=os.Open(name);defer println("exit");return f,err}
func deferredClose(name string)(*os.File,error){f,err:=os.Open(name);defer f.Close();return f,err}
func methodExpressionClose(name string)(*os.File,error){f,err:=os.Open(name);defer (*os.File).Close(f);return f,err}
func named(name string)(f *os.File,err error){if name==""{return};f,err=os.Open(name);return}
func genericZero[T any](name string,b bool)(any,error){if b{return os.Open(name)};var zero T;return zero,nil}
func closedForward(name string)(*os.File,error){return closed(name)}
func mixedErrors(name string)(*os.File,error){f,_:=os.Open(name);_,err:=os.Open(name);return f,err}
func osOtherClose(name string,other *os.File)(*os.File,error){f,err:=os.Open(name);other.Close();return f,err}
func wrappedOtherClose(name string,other *os.File)(*os.File,error){f,err:=direct(name);other.Close();return f,err}
func fsOtherClose(files fs.FS,name string,other fs.File)(fs.File,error){f,err:=files.Open(name);other.Close();return f,err}
`
	pkg := loadTestPackage(t, source)
	facts, err := packageResourceSummaries(pkg)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]resourceflow.FactorySummary{
		"direct": {Fresh: true, FailureEmpty: true}, "local": {Fresh: true, FailureEmpty: true}, "alias": {Fresh: true, FailureEmpty: true}, "loopCopy": {Fresh: true, FailureEmpty: true}, "returnOrClose": {Fresh: true, FailureEmpty: true},
		"closed": {}, "replaced": {}, "unreachable": {Fresh: true, FailureEmpty: true}, "recursive": {Fresh: true, FailureEmpty: true}, "cycle": {}, "nilOnly": {}, "mixed": {Fresh: true, FailureEmpty: true},
		"boxedError": {Fresh: true}, "boxedCopyError": {Fresh: true}, "bad": {Fresh: true}, "badForward": {Fresh: true}, "single": {Fresh: true}, "fsSource": {Fresh: true, FailureEmpty: true}, "fsExpression": {Fresh: true, FailureEmpty: true}, "unknown": {}, "generic": {Fresh: true, FailureEmpty: true}, "instantiated": {Fresh: true, FailureEmpty: true}, "printer": {Fresh: true, FailureEmpty: true}, "deferredClose": {}, "methodExpressionClose": {}, "named": {Fresh: true, FailureEmpty: true}, "genericZero": {}, "closedForward": {}, "mixedErrors": {Fresh: true}, "osOtherClose": {Fresh: true, FailureEmpty: true}, "wrappedOtherClose": {Fresh: true, FailureEmpty: true}, "fsOtherClose": {},
	} {
		fn := findFunc(pkg.Syntax[0], name)
		object := pkg.TypesInfo.Defs[fn.Name]
		got := resourceflow.FactorySummary{Fresh: facts.sources[object], FailureEmpty: facts.failures[object]}
		if got != want {
			t.Errorf("%s=%+v want %+v", name, got, want)
		}
	}
}
func TestFactoryAndCleanupChainsHaveNoPassLimit(t *testing.T) {
	var source strings.Builder
	source.WriteString("package p;import \"os\"\n")
	for i := 0; i < 24; i++ {
		fmt.Fprintf(&source, "func source%d(name string)(*os.File,error){return source%d(name)}\n", i, i+1)
		fmt.Fprintf(&source, "func cleanup%d(f *os.File)error{return cleanup%d(f)}\n", i, i+1)
	}
	source.WriteString("func source24(name string)(*os.File,error){return os.Open(name)}\nfunc cleanup24(f *os.File)error{return f.Close()}\n")
	pkg := loadTestPackage(t, source.String())
	facts, err := packageResourceSummaries(pkg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= 24; i++ {
		fn := findFunc(pkg.Syntax[0], fmt.Sprintf("source%d", i))
		object := pkg.TypesInfo.Defs[fn.Name]
		if !facts.sources[object] || !facts.failures[object] {
			t.Errorf("source chain stops at %d", i)
		}
		fn = findFunc(pkg.Syntax[0], fmt.Sprintf("cleanup%d", i))
		if !facts.helpers[pkg.TypesInfo.Defs[fn.Name]] {
			t.Errorf("cleanup chain stops at %d", i)
		}
	}
}
func TestAcquisitionBoundariesUseFactoryFailureContract(t *testing.T) {
	pkg := loadTestPackage(t, `package p;import "os"
func good(name string)(*os.File,error){return os.Open(name)}
func bad(name string,cause error)(*os.File,error){f,_:=os.Open(name);return f,cause}
func single(name string)*os.File{f,_:=os.Open(name);return f}
func goodCaller(name string)error{var f,err=good(name);if err!=nil{return err};defer (*os.File).Close(f);return nil}
func badCaller(name string,cause error)error{f,err:=bad(name,cause);if err!=nil{return err};defer f.Close();return nil}
func singleLeak(name string)error{var f=single(name);_=f;return nil}
func reassigned(name string)error{var f *os.File;var err error;f,err=good(name);if err!=nil{return err};defer f.Close();return nil}
`)
	summaries, err := packageResourceSummaries(pkg)
	if err != nil {
		t.Fatal(err)
	}
	ctx := extractCtx{fileSources: summaries.sources, fileFailures: summaries.failures, helpers: summaries.helpers}
	for name, want := range map[string]int{"goodCaller": 0, "badCaller": 1, "singleLeak": 1, "reassigned": 0} {
		fn := findFunc(pkg.Syntax[0], name)
		graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
		if err != nil {
			t.Fatal(err)
		}
		_, sites, err := fileObligationFacts(pkg, fn, ctx, &graph)
		if err != nil {
			t.Fatal(err)
		}
		if len(sites) != want {
			t.Errorf("%s sites=%v want %d", name, sites, want)
		}
	}
}

func TestFactoryEvidenceFollowsCanonicalGraph(t *testing.T) {
	for _, change := range []string{"reaching source", "returned producer", "call target", "failure input", "missing representation", "boxed representation"} {
		t.Run(change, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import "os";func factory(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};return f,err}`)
			fn := findFunc(pkg.Syntax[0], "factory")
			graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			summary := func() resourceflow.FactorySummary {
				query, err := resourceflow.New(&graph)
				if err != nil {
					t.Fatal(err)
				}
				return resourceflow.SummarizeFactories(map[string]*resourceflow.Query{"p.factory": query}, nil)["p.factory"]
			}
			if got := summary(); !got.Fresh || !got.FailureEmpty {
				t.Fatalf("baseline=%+v", got)
			}
			switch change {
			case "missing representation", "boxed representation":
				for _, node := range graph.Nodes {
					if node.Category != pdg.Return {
						continue
					}
					attrs := &graph.Attributes[node.Attributes-1]
					if change == "missing representation" {
						graph.AttributeTables.Extensions[attrs.Extensions-1] = slices.DeleteFunc(graph.AttributeTables.Extensions[attrs.Extensions-1], func(ext pdg.Extension) bool { return graph.Tables.Text(ext.Name) == "gstyle.go:input-representation" })
						continue
					}
					for i, ext := range graph.AttributeTables.Extensions[attrs.Extensions-1] {
						if graph.Tables.Text(ext.Name) != "gstyle.go:input-representation" {
							continue
						}
						graph.Tables.Strings = append(graph.Tables.Strings, `[{"position":1,"kind":"identity"},{"position":2,"kind":"boxed"}]`)
						graph.AttributeTables.Extensions[attrs.Extensions-1][i].Value = pdg.Text(len(graph.Tables.Strings))
					}
				}
			case "reaching source":
				graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool {
					return edge.Symbol != 0 && graph.Tables.Text(graph.Symbols[edge.Symbol-1].Name.Value) == "f" && graph.Tables.Text(edge.Subkind) == "reaching-definition"
				})
			case "returned producer":
				var nilRef pdg.Ref
				for i, node := range graph.Nodes {
					for _, ext := range graph.NodeAttributes(node).Extensions {
						if graph.Tables.Text(ext.Name) == "gstyle.runtime:nil" && graph.Tables.Text(ext.Value) == "true" {
							nilRef = pdg.Ref(i + 1)
						}
					}
				}
				if nilRef == 0 {
					t.Fatal("nil producer missing")
				}
				for i, edge := range graph.Edges {
					if canonicalResourceValueEdge(&graph, edge) && graph.Nodes[edge.Target-1].Category == pdg.Return && edge.Position == 1 {
						graph.Edges[i].Source = nilRef
					}
				}
			case "call target":
				for _, node := range graph.Nodes {
					if node.Category == pdg.Call {
						attrs := &graph.Attributes[node.Attributes-1]
						graph.Tables.Strings = append(graph.Tables.Strings, "p.unknown")
						attrs.Callee = pdg.Fact{Status: pdg.Known, Value: pdg.Text(len(graph.Tables.Strings))}
					}
				}
			case "failure input":
				graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool {
					return canonicalResourceValueEdge(&graph, edge) && graph.Nodes[edge.Target-1].Category == pdg.Return && edge.Position == 2
				})
			}
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal(err)
			}
			got := summary()
			if change == "failure input" || change == "missing representation" || change == "boxed representation" {
				if !got.Fresh || got.FailureEmpty {
					t.Fatalf("missing error input=%+v", got)
				}
			} else if got.Fresh {
				t.Fatalf("source syntax replaces changed graph: %+v", got)
			}
		})
	}
}

func TestMissingFactoryEffectInputStaysUnknown(t *testing.T) {
	pkg := loadTestPackage(t, `package p;import "os";func factory(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};f.Close();return f,nil}`)
	fn := findFunc(pkg.Syntax[0], "factory")
	graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool {
		attrs := graph.NodeAttributes(graph.Nodes[edge.Target-1])
		return canonicalResourceValueEdge(&graph, edge) && graph.Nodes[edge.Target-1].Category == pdg.Call && graph.Tables.Text(attrs.Callee.Value) == "os.File.Close"
	})
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	query, err := resourceflow.New(&graph)
	if err != nil {
		t.Fatal(err)
	}
	if summary := resourceflow.SummarizeFactories(map[string]*resourceflow.Query{"p.factory": query}, nil)["p.factory"]; summary.Fresh {
		t.Fatalf("missing effect evidence establishes freshness: %+v", summary)
	}
}
