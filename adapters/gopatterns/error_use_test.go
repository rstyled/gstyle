package gopatterns

import (
	"bytes"
	"encoding/json"
	"github.com/rstyled/gstyle/domain/errorflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestErrorUsesFollowCanonicalDefinitions(t *testing.T) {
	cases := []struct {
		name, body       string
		checked, unknown bool
	}{
		{"overwritten", `err:=produce();err=nil;if err!=nil{println(err)}`, false, false},
		{"copied and checked", `err:=produce();alias:=err;if alias!=nil{println(alias)}`, true, false},
		{"copied then overwritten", `err:=produce();alias:=err;alias=nil;if alias!=nil{println(alias)}`, false, false},
		{"copy survives original overwrite", `err:=produce();alias:=err;err=nil;_=err;if alias!=nil{println(alias)}`, true, false},
		{"discarded comparison", `err:=produce();_=err!=nil`, false, false},
		{"checked boolean copy", `err:=produce();bad:=err!=nil;if bad{println("bad")}`, true, false},
		{"blank before copied value", `err:=produce();_,alias:=1,err;if alias!=nil{println(alias)}`, true, false},
		{"declaration error slot", `var n,err=pair();_=n;if err!=nil{println(err)}`, true, false},
		{"declaration offset", `var n,err=1,produce();_=n;if err!=nil{println(err)}`, true, false},
		{"partly checked error tuple", `left,right:=both();if left!=nil{println(left)};_=right`, false, false},
		{"fully checked error tuple", `left,right:=both();if left!=nil{println(left)};if right!=nil{println(right)}`, true, false},
		{"opaque closure reader", `err:=produce();_=func(){_=err}`, false, true},
		{"unmodeled address reader", `err:=produce();inspect(&err)`, false, true},
		{"unmodeled aggregate carry", `err:=produce();_= []error{err}`, false, true},
		{"only integer slot used", `n,err:=pair();_=err;println(n)`, false, false},
		{"nested call consumption", `consume(produce())`, true, false},
		{"previous iteration", `var err error;for c{if err!=nil{println(err)};err=produce()}`, true, false},
		{"deferred body remains unknown", `defer noop();err:=produce();if err!=nil{println(err)}`, false, true},
		{"unreachable call", `return;err:=produce();if err!=nil{println(err)}`, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn, fset, info := parseNilErrFunc(t, `package p;func f(c bool){`+tc.body+`};func produce()error{return nil};func pair()(int,error){return 0,nil};func consume(error){};func noop(){};func inspect(*error){};func both()(error,error){return nil,nil}`)
			sites := findErrorCheckSites(fn, fset, info)
			if len(sites) != 1 || sites[0].Checked != tc.checked || sites[0].CheckUnknown != tc.unknown {
				t.Fatalf("sites=%+v want checked=%v unknown=%v", sites, tc.checked, tc.unknown)
			}
		})
	}
}

func TestAggregateOperandsDoNotReadStructFieldNames(t *testing.T) {
	source := `package p;func f(err error)any{return struct{E error}{E:err}}`
	graph := parseIR(t, source, pdg.NewBuilder())
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	for _, symbol := range graph.Symbols {
		if graph.Tables.Text(symbol.Name.Value) == "E" {
			t.Fatal("struct literal field name becomes a variable binding")
		}
	}
	found := false
	for _, edge := range graph.Edges {
		if graph.Tables.Text(graph.NodeAttributes(graph.Nodes[edge.Target-1]).Operator.Value) == "key-value" && edge.Position == 2 {
			found = true
		}
	}
	if !found {
		t.Fatal("struct literal loses its value operand")
	}
}

func TestErrorConversionsAreNotErrorProducingCalls(t *testing.T) {
	fn, fset, info := parseNilErrFunc(t, `package p;type E struct{};func f(){_ = error((*E)(nil))};func(*E)Error()string{return ""}`)
	if sites := findErrorCheckSites(fn, fset, info); len(sites) != 0 {
		t.Fatalf("conversion becomes error call: %+v", sites)
	}
}

func TestErrorUseQueriesLeaveCanonicalGraphUnchanged(t *testing.T) {
	graph := parseIR(t, `package p;func f()error{err:=produce();alias:=err;return alias};func produce()error{return nil}`, pdg.NewBuilder())
	before, err := json.Marshal(&graph)
	if err != nil {
		t.Fatal(err)
	}
	query, err := errorflow.New(&graph)
	if err != nil {
		t.Fatal(err)
	}
	for index, node := range graph.Nodes {
		switch node.Category {
		case pdg.Call:
			query.Value(pdg.Ref(index + 1))
		case pdg.Assignment, pdg.Declaration:
			query.Assignment(pdg.Ref(index+1), 1)
		}
	}
	after, err := json.Marshal(&graph)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("error-use query mutates borrowed PDG input")
	}
}

func TestNamedErrorReturnsConsumeTheReachingDefinition(t *testing.T) {
	for _, tc := range []struct {
		body    string
		checked bool
	}{
		{`err=produce();return`, true},
		{`err=produce();err=nil;return`, false},
	} {
		fn, fset, info := parseNilErrFunc(t, `package p;func f()(err error){`+tc.body+`};func produce()error{return nil}`)
		sites := findErrorCheckSites(fn, fset, info)
		if len(sites) != 1 || sites[0].Checked != tc.checked || sites[0].CheckUnknown {
			t.Fatalf("sites=%+v, want checked=%v", sites, tc.checked)
		}
	}
}
