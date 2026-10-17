package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"slices"
	"testing"
)

func TestResourceAliasStateFollowsCanonicalPaths(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"original overwrite with saved alias", `alias:=file;file=other;alias.Close()`, 0},
		{"original overwrite loses ownership", `file=other;file.Close()`, 1},
		{"saved original receiver before overwrite", `defer file.Close();file=other;_=file`, 0},
		{"tuple moves original", `file,other=other,file;other.Close()`, 0},
		{"last repeated destination retains alias", `alias:=other;alias,alias=other,file;alias.Close()`, 0},
		{"last repeated destination loses alias", `alias:=other;alias,alias=file,other;alias.Close()`, 1},
		{"saved alias across panic", `alias:=file;defer alias.Close();alias=other;panic("fault")`, 0},
		{"copy close", `alias:=file;alias.Close()`, 0},
		{"copy chain", `first:=file;second:=first;second.Close()`, 0},
		{"var copy", `var alias= file;alias.Close()`, 0},
		{"discarded copy", `alias:=file;_=alias`, 1},
		{"overwritten copy", `alias:=file;alias=other;alias.Close()`, 1},
		{"branch copies", `var alias fs.File;if choose {alias=file}else{alias=file};alias.Close()`, 0},
		{"partial branch copy", `alias:=other;if choose {alias=file};alias.Close()`, 1},
		{"branch close or alias", `alias:=other;if choose {alias=file}else{file.Close()};alias.Close()`, 0},
		{"saved alias receiver", `alias:=file;defer alias.Close();alias=other;_=alias`, 0},
		{"saved other receiver", `alias:=other;defer alias.Close();alias=file;_=alias`, 1},
		{"alias handoff", `alias:=file;consume(alias)`, 0},
		{"conditional alias handoff", `alias:=other;if choose {alias=file};consume(alias)`, 1},
		{"global alias handoff", `alias:=file;held=alias`, 0},
		{"tuple swap loses alias", `alias:=file;alias,other=other,alias;alias.Close()`, 1},
		{"tuple swap retains alias", `alias:=file;alias,other=other,alias;other.Close()`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import "io/fs";var choose bool;var held fs.File;func consume(fs.File){};func caller(files fs.FS,other fs.File){file,err:=files.Open("file");if err!=nil{return};`+tc.body+`}`)
			fn := findFunc(pkg.Syntax[0], "caller")
			graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			_, sites, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph)
			if err != nil {
				t.Fatal(err)
			}
			if len(sites) != tc.want {
				t.Fatalf("obligations=%v want %d", sites, tc.want)
			}
		})
	}
}

func TestResourceAliasNeedsCanonicalCopyOperand(t *testing.T) {
	graph := parseIR(t, `package p;func f(resource *T){alias:=resource;alias.Close()};type T struct{};func(*T)Close(){}`, pdg.NewBuilder())
	policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol}
	prove := func() bool {
		q, err := resourceflow.New(&graph)
		if err != nil {
			t.Fatal(err)
		}
		return q.FromEntry(policy)
	}
	if !prove() {
		t.Fatal("baseline alias close not recognized")
	}
	changed := false
	graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool {
		target := graph.Nodes[edge.Target-1]
		if edge.Kind == pdg.Data && (target.Category == pdg.Declaration || target.Category == pdg.Assignment) && (graph.Tables.Text(edge.Subkind) == "binding-value" || graph.Tables.Text(edge.Subkind) == "source-value") {
			changed = true
			return true
		}
		return false
	})
	if !changed {
		t.Fatal("missing copy operand to remove")
	}
	if prove() {
		t.Fatal("missing canonical copy replaced by source alias spelling")
	}
}

func TestResourceAliasBeforeAcquisitionRetainsOldValue(t *testing.T) {
	pkg := loadTestPackage(t, `package p;import "io/fs";func caller(files fs.FS,other fs.File){file:=other;alias:=file;file,err:=files.Open("file");if err!=nil{return};alias.Close()}`)
	fn := findFunc(pkg.Syntax[0], "caller")
	graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	_, sites, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 1 {
		t.Fatalf("old alias discharges new acquisition: %v", sites)
	}
}

func TestResourceAliasReturnUsesCurrentValue(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{
		{`alias:=file;return alias,nil`, true},
		{`alias:=file;file=other;return alias,nil`, true},
		{`alias:=file;alias=other;return alias,nil`, false},
		{`alias:=file;result=alias;return`, true},
	} {
		pkg := loadTestPackage(t, `package p;import "io/fs";func caller(files fs.FS,other fs.File)(result fs.File,err error){file,err:=files.Open("file");if err!=nil{return nil,err};`+tc.body+`}`)
		fn := findFunc(pkg.Syntax[0], "caller")
		graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
		if err != nil {
			t.Fatal(err)
		}
		released, _, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph)
		if err != nil {
			t.Fatal(err)
		}
		if (len(released) > 0) != tc.want {
			t.Fatalf("%s: transfer=%v want %v", tc.body, released, tc.want)
		}
	}
}

func TestNativeAliasSnapshotsAndSwap(t *testing.T) {
	first, second := &handoffNativeFile{}, &handoffNativeFile{}
	alias := first
	func() { defer alias.Close(); alias = second }()
	if !first.closed || second.closed {
		t.Fatalf("saved receiver closes first=%v second=%v", first.closed, second.closed)
	}
	first, second = &handoffNativeFile{}, &handoffNativeFile{}
	alias, other := first, second
	alias, other = other, alias
	other.Close()
	if !first.closed || second.closed {
		t.Fatalf("parallel swap closes first=%v second=%v", first.closed, second.closed)
	}
}

func TestAliasCommitOrderUsesCanonicalSlots(t *testing.T) {
	graph := parseIR(t, `package p;func f(resource,other *T){alias:=other;alias,alias=other,resource;alias.Close()};type T struct{};func(*T)Close(){}`, pdg.NewBuilder())
	policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol}
	prove := func() bool {
		q, err := resourceflow.New(&graph)
		if err != nil {
			t.Fatal(err)
		}
		return q.FromEntry(policy)
	}
	if !prove() {
		t.Fatal("baseline last slot not retained")
	}
	changed := false
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if len(attrs.Writes) != 2 {
			continue
		}
		writes := slices.Clone(attrs.Writes)
		slices.Reverse(writes)
		set := graph.Attributes[node.Attributes-1]
		graph.AttributeTables.Accesses[set.Writes-1] = writes
		changed = true
	}
	if !changed {
		t.Fatal("missing repeated destination writes")
	}
	if !prove() {
		t.Fatal("access table order replaces canonical slot order")
	}
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if len(attrs.Writes) != 2 {
			continue
		}
		writes := slices.Clone(attrs.Writes)
		for i := range writes {
			if writes[i].Position == 1 {
				writes[i].Position = 0
			}
		}
		set := graph.Attributes[node.Attributes-1]
		graph.AttributeTables.Accesses[set.Writes-1] = writes
	}
	if prove() {
		t.Fatal("unordered write cannot rule out a final overwrite")
	}
}

func TestUnknownCallWriteCannotPreserveOwnedAlias(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := parseIR(t, `package p;func f(resource *T){alias:=resource;noop();alias.Close()};type T struct{};func(*T)Close(){};func noop(){}`, pool)
	policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol, Exits: callflow.ExitSummaries{"example.noop": {Ordinary: callflow.Returns, DeferredPanic: callflow.Returns}}}
	prove := func() bool {
		q, err := resourceflow.New(&graph)
		if err != nil {
			t.Fatal(err)
		}
		return q.FromEntry(policy)
	}
	if !prove() {
		t.Fatal("baseline known-return call prevents alias close")
	}
	reason := pool.Text("unknown binding write")
	graph.Tables = pool.Tables
	changed := false
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if node.Category != pdg.Call || graph.Tables.Text(attrs.Callee.Value) != "example.noop" {
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
	if prove() {
		t.Fatal("unknown call write preserves exact alias")
	}
}
