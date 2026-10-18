package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
	"strings"
	"testing"
)

func TestRecoveryValuesSelectCanonicalBranches(t *testing.T) {
	for _, tc := range []struct {
		name, body          string
		ordinary, inherited callflow.Outcomes
	}{
		{"disabled OR rhs fault", `if recover()!=nil||values[index]>0{return};panic("empty")`, callflow.Returns | callflow.Panics, callflow.RecoversInherited},
		{"disabled RHS fault after nonempty recovery", `if recover()==nil&&values[index]>0{panic("rhs")}`, callflow.Returns | callflow.Panics, callflow.RecoversInherited},
		{"disabled RHS fault after ordinary recovery", `if recover()!=nil&&values[index]>0{panic("rhs")}`, callflow.Returns, callflow.RecoversInherited | callflow.Panics},
		{"short circuit and", `if recover()!=nil&&recover()!=nil{panic("wrong")}`, callflow.Returns, callflow.RecoversInherited},
		{"short circuit or", `if recover()!=nil||recover()!=nil{panic("got")}`, callflow.Returns, callflow.Panics},
		{"branch snapshots", `value:=recover();if choose{value=nil};if value==nil{panic("empty")}`, callflow.Panics, callflow.RecoversInherited | callflow.Panics},
		{"nonempty guard", `if recover()!=nil{panic("got")}`, callflow.Returns, callflow.Panics},
		{"empty guard", `if recover()==nil{panic("empty")}`, callflow.Panics, callflow.RecoversInherited},
		{"initializer", `if value:=recover();value!=nil{panic("got")}`, callflow.Returns, callflow.Panics},
		{"binding copy", `value:=recover();alias:=value;if alias==nil{panic("empty")}`, callflow.Panics, callflow.RecoversInherited},
		{"overwrite", `value:=recover();value=nil;if value==nil{panic("empty")}`, callflow.Panics, callflow.Panics},
		{"second recovery is empty", `first:=recover();second:=recover();if first==nil||second!=nil{panic("wrong")}`, callflow.Panics, callflow.RecoversInherited},
		{"boolean copy", `found:=recover()!=nil;if !found{panic("empty")}`, callflow.Panics, callflow.RecoversInherited},
		{"parallel swap", `first:=recover();var second any;first,second=second,first;if first!=nil||second==nil{panic("wrong")}`, callflow.Panics, callflow.RecoversInherited},
		{"repeated destination", `value:=recover();value,value=value,nil;if value!=nil{panic("wrong")}`, callflow.Returns, callflow.RecoversInherited},
		{"typed nil boxing", `value:=recover();var pointer *int;value=pointer;if value==nil{panic("wrong")}`, callflow.Returns, callflow.RecoversInherited},
		{"unknown overwrite", `value:=recover();value=unknown;if value==nil{panic("unknown")}`, callflow.Returns | callflow.Panics, callflow.RecoversInherited | callflow.Panics},
		{"unknown parameter", `recover();if unknown!=nil{panic("unknown")}`, callflow.Returns | callflow.Panics, callflow.RecoversInherited | callflow.Panics},
		{"identity conversion", `value:=any(recover());if value==nil{panic("unknown")}`, callflow.Panics, callflow.RecoversInherited},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := recoveryValueGraph(t, `package p;var values []int;var index int;func f(unknown any,choose bool){`+tc.body+`}`, pdg.NewBuilder())
			name := graph.Tables.Text(graph.Function.ID)
			summaries, err := callflow.ExitEffects(map[string]*pdg.Graph{name: &graph})
			if err != nil {
				t.Fatal(err)
			}
			got := summaries[name]
			if got.Ordinary != tc.ordinary || got.DeferredPanic != tc.inherited {
				t.Fatalf("ordinary=%v inherited=%v want %v/%v", got.Ordinary, got.DeferredPanic, tc.ordinary, tc.inherited)
			}
		})
	}
}

func nativeRecoveryNilness(flags *[]bool) {
	first := recover()
	second := recover()
	*flags = []bool{first == nil, second == nil}
}
func TestNativeRecoveredNilnessAndBoxedNil(t *testing.T) {
	t.Setenv("GODEBUG", "panicnil=0")
	var ordinary []bool
	nativeRecoveryNilness(&ordinary)
	if len(ordinary) != 2 || !ordinary[0] || !ordinary[1] {
		t.Fatalf("ordinary flags=%v", ordinary)
	}
	for _, payload := range []any{"panic", (*int)(nil), nil} {
		var flags []bool
		func() { defer nativeRecoveryNilness(&flags); panic(payload) }()
		if len(flags) != 2 || flags[0] || !flags[1] {
			t.Fatalf("payload %T flags=%v", payload, flags)
		}
	}
	t.Setenv("GODEBUG", "panicnil=1")
	var legacy []bool
	func() { defer nativeRecoveryNilness(&legacy); panic(nil) }()
	if len(legacy) != 2 || !legacy[0] || !legacy[1] {
		t.Fatalf("legacy nil panic flags=%v", legacy)
	}
	var pointer *int
	var boxed any = pointer
	if boxed == nil {
		t.Fatal("typed nil pointer becomes an empty interface")
	}
}

func TestRecoveryValueNeedsCanonicalRepresentation(t *testing.T) {
	graph := recoveryValueGraph(t, `package p;func f(){value:=recover();var pointer *int;value=pointer;if value==nil{panic("bad")}}`, pdg.NewBuilder())
	name := graph.Tables.Text(graph.Function.ID)
	summarize := func() callflow.ExitSummary {
		values, err := callflow.ExitEffects(map[string]*pdg.Graph{name: &graph})
		if err != nil {
			t.Fatal(err)
		}
		return values[name]
	}
	if got := summarize(); got.Ordinary != callflow.Returns || got.DeferredPanic != callflow.RecoversInherited {
		t.Fatalf("baseline=%v", got)
	}
	changed := false
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if graph.Tables.Text(attrs.WriteForm) != "=" {
			continue
		}
		set := graph.Attributes[node.Attributes-1]
		graph.AttributeTables.Extensions[set.Extensions-1] = slices.DeleteFunc(slices.Clone(attrs.Extensions), func(e pdg.Extension) bool {
			if graph.Tables.Text(e.Name) == "gstyle.go:input-representation" {
				changed = true
				return true
			}
			return false
		})
	}
	if !changed {
		t.Fatal("missing representation to remove")
	}
	if got := summarize(); got.Ordinary != callflow.Returns|callflow.Panics || got.DeferredPanic != callflow.RecoversInherited|callflow.Panics {
		t.Fatalf("missing boxing fact replaced by source type: %v", got)
	}
}

func TestRecoveredNilnessNeedsLanguageVersion(t *testing.T) {
	for _, version := range []string{"go1.20", "unknown", "missing"} {
		pool := pdg.NewBuilder()
		graph := parseIR(t, `package p;func f(){if recover()==nil{panic("empty")}}`, pool)
		if version == "missing" {
			graph.Producer.LanguageVersion = pool.Missing(pdg.Unknown, "version absent")
		} else {
			graph.Producer.LanguageVersion = pool.Known(version)
		}
		graph.Tables = pool.Tables
		name := graph.Tables.Text(graph.Function.ID)
		summaries, err := callflow.ExitEffects(map[string]*pdg.Graph{name: &graph})
		if err != nil {
			t.Fatal(err)
		}
		if got := summaries[name].DeferredPanic; got != callflow.RecoversInherited|callflow.Panics {
			t.Fatalf("version=%s exits=%v", version, got)
		}
	}
}

func recoveryValueGraph(t *testing.T, source string, pool *pdg.Builder) pdg.Graph {
	t.Helper()
	pkg := loadTestPackage(t, source)
	graph, err := buildGraph(pkg, findFunc(pkg.Syntax[0], "f"), "p.go", pool)
	if err != nil {
		t.Fatal(err)
	}
	return graph
}

func TestUnknownWriteInvalidatesRecoveredBindingValue(t *testing.T) {
	pkg := loadTestPackage(t, `package p;func f(){value:=recover();noop();if value==nil{panic("empty")}};func noop(){}`)
	pool := pdg.NewBuilder()
	graph, err := buildGraph(pkg, findFunc(pkg.Syntax[0], "f"), "p.go", pool)
	if err != nil {
		t.Fatal(err)
	}
	noop, err := buildGraph(pkg, findFunc(pkg.Syntax[0], "noop"), "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	name := graph.Tables.Text(graph.Function.ID)
	summarize := func() callflow.ExitSummary {
		values, err := callflow.ExitEffects(map[string]*pdg.Graph{name: &graph, noop.Tables.Text(noop.Function.ID): &noop})
		if err != nil {
			t.Fatal(err)
		}
		return values[name]
	}
	if got := summarize(); got.Ordinary != callflow.Panics || got.DeferredPanic != callflow.RecoversInherited {
		t.Fatalf("baseline=%v", got)
	}
	reason := pool.Text("unknown binding write")
	graph.Tables = pool.Tables
	changed := false
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if node.Category != pdg.Call || !strings.HasSuffix(graph.Tables.Text(attrs.Callee.Value), ".noop") {
			continue
		}
		writes := append(slices.Clone(attrs.Writes), pdg.Access{UnresolvedReason: reason})
		graph.AttributeTables.Accesses = append(graph.AttributeTables.Accesses, writes)
		graph.Attributes[node.Attributes-1].Writes = pdg.Ref(len(graph.AttributeTables.Accesses))
		changed = true
	}
	if !changed {
		t.Fatal("missing call to mutate")
	}
	if got := summarize(); got.Ordinary != callflow.Returns|callflow.Panics || got.DeferredPanic != callflow.RecoversInherited|callflow.Panics {
		t.Fatalf("unknown write preserves exact value: %v", got)
	}
}
