package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
	"testing"
)

func TestErrorBeliefsUseCanonicalCallTargets(t *testing.T) {
	fn, fset, info := parseNilErrFunc(t, `package p
func f(a A,b B,d interface{Run()error},dynamic func()error){
 _=a.Run();_=b.Run();_=d.Run();_=dynamic()
}

type A struct{};func(A)Run()error{return nil}
type B struct{};func(B)Run()error{return nil}`)
	sites := findErrorCheckSites(fn, fset, info)
	if len(sites) != 4 {
		t.Fatalf("sites=%+v", sites)
	}
	for i, want := range []string{"test.A.Run", "test.B.Run"} {
		if sites[i].Callee != want || sites[i].CheckUnknown {
			t.Fatalf("site=%+v want %s", sites[i], want)
		}
	}
	for _, site := range sites[2:] {
		if !site.CheckUnknown {
			t.Fatalf("dynamic target supplies belief: %+v", site)
		}
	}
}

func TestErrorTargetFollowsCanonicalFact(t *testing.T) {
	file, fs, info, pkg := typeCheck(t, `package p;func f(){_=produce()};func produce()error{return nil}`)
	fn := findFunc(file, "f")
	graph, err := buildGraph(&packages.Package{Fset: fs, TypesInfo: info, Types: pkg, PkgPath: pkg.Path()}, fn, "test.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	for i, node := range graph.Nodes {
		if node.Category != pdg.Call {
			continue
		}
		attrs := &graph.Attributes[node.Attributes-1]
		graph.Tables.Strings = append(graph.Tables.Strings, "other.Target")
		attrs.Callee = pdg.Fact{Status: pdg.Known, Value: pdg.Text(len(graph.Tables.Strings))}
		if err := pdg.Validate(&graph); err != nil {
			t.Fatal(err)
		}
		sites, _, err := reachableErrorCheckFacts(fn, fs, info, &graph)
		if err != nil {
			t.Fatal(err)
		}
		if len(sites) != 1 || sites[0].Callee != "other.Target" {
			t.Fatalf("source replaces canonical target at %d: %+v", i, sites)
		}
		attrs.CallForm = pdg.IndirectCall
		sites, _, err = reachableErrorCheckFacts(fn, fs, info, &graph)
		if err != nil {
			t.Fatal(err)
		}
		if len(sites) != 1 || !sites[0].CheckUnknown {
			t.Fatalf("indirect target supplies belief: %+v", sites)
		}
		return
	}
	t.Fatal("missing call")
}

func TestErrorTargetIgnoresImportAliases(t *testing.T) {
	for _, alias := range []string{"fmt", "out", "."} {
		name := alias + ".Errorf"
		if alias == "." {
			name = "Errorf"
		}
		file, fs, info, _ := typeCheck(t, `package p;import `+alias+` "fmt";func f(){_=`+name+`("failed")}`)
		sites := findErrorCheckSites(findFunc(file, "f"), fs, info)
		if len(sites) != 1 || sites[0].Callee != "fmt.Errorf" || sites[0].CheckUnknown {
			t.Fatalf("alias %s: %+v", alias, sites)
		}
	}
}
