package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func TestResourceHandoffMustCoverEveryPath(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"formatting is not handoff", `fmt.Println(file)`, 1},
		{"reader is not ownership", `readOnly(file)`, 1},
		{"variadic close capable input", `consumeMany(file)`, 0},
		{"global handoff", `held=file`, 0},
		{"conditional global handoff", `if choose {held=file}`, 1},
		{"direct handoff", `consume(file)`, 0},
		{"conditional handoff", `if choose {consume(file)}`, 1},
		{"handoff or close", `if choose {consume(file)} else {file.Close()}`, 0},
		{"bounds before handoff", `_=values[index];consume(file)`, 1},
		{"handoff before later fault", `consume(file);_=values[index]`, 0},
		{"short circuit fault follows handoff", `_ = receive(file)>0 && values[index]>0`, 0},
		{"independent sibling after logical join", `consumeBool(receive(file)>0&&choose,values[index])`, 1},
		{"later call produces sibling fault", `consumeInts(receive(file),nextValues()[index])`, 0},
		{"earlier call requires sibling fault", `consumeInts(identity(values[index]),receive(file))`, 1},
		{"nested handoff versus sibling fault", `consumeInts(receive(file),values[index])`, 1},
		{"saved close covers nested handoff order", `defer file.Close();consumeInts(receive(file),values[index])`, 0},
		{"sibling fault before entry", `consumeWith(file,values[index])`, 1},
		{"saved close covers sibling fault", `defer file.Close();consumeWith(file,values[index])`, 0},
		{"nil callable", `var target func(fs.File);target(file)`, 1},
		{"printing is not handoff", `println(file)`, 1},
		{"deferred printing is not handoff", `defer println(file)`, 1},
		{"deferred handoff", `defer consume(file)`, 0},
		{"abort prevents deferred handoff", `defer consume(file);defer abort()`, 1},
		{"handoff before older abort", `defer abort();defer consume(file)`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import("io/fs";"os";"fmt";"io");var _=fmt.Println;var held fs.File;var choose bool;var values []int;var index int
 func caller(files fs.FS){file,err:=files.Open("file");if err!=nil{return};`+tc.body+`}
 func consumeBool(bool,int){};func nextValues()[]int{return nil};func identity(x int)int{return x};func receive(fs.File)int{return 0};func consumeInts(int,int){};func consumeMany(...fs.File){};func readOnly(io.Reader){};func consume(fs.File){};func consumeWith(fs.File,int){};func abort(){os.Exit(0)}`)
			summaries, err := packageResourceSummaries(pkg)
			if err != nil {
				t.Fatal(err)
			}
			fn := findFunc(pkg.Syntax[0], "caller")
			graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			ctx := extractCtx{fileSources: summaries.sources, fileFailures: summaries.failures, helpers: summaries.helpers, exits: summaries.exits}
			_, sites, err := fileObligationFacts(pkg, fn, ctx, &graph)
			if err != nil {
				t.Fatal(err)
			}
			if len(sites) != tc.want {
				t.Fatalf("obligations=%v want %d", sites, tc.want)
			}
		})
	}
}

func TestNativeFaultCanPreventOwnershipHandoff(t *testing.T) {
	for _, kind := range []string{"bounds", "nil callable", "deferred panic"} {
		handed := false
		func() {
			defer func() { _ = recover() }()
			consume := func() { handed = true }
			switch kind {
			case "bounds":
				var values []int
				_ = values[0]
				consume()
			case "nil callable":
				var target func()
				target()
				consume()
			case "deferred panic":
				defer consume()
				// Panic unwinding still invokes the earlier deferred handoff.
				panic(7)
			}
		}()
		if handed != (kind == "deferred panic") {
			t.Fatalf("%s: handoff=%v", kind, handed)
		}
	}
}

func TestHandoffRequiresCanonicalEntryAndInputFacts(t *testing.T) {
	for _, fact := range []string{"gstyle.go:call-entry", "gstyle.go:resource-inputs"} {
		graph := parseIR(t, `package p;func f(resource *T){consume(resource)};type T struct{};func(*T)Close(){};func consume(*T){}`, pdg.NewBuilder())
		policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol, TransferCalls: true}
		prove := func() bool {
			query, err := resourceflow.New(&graph)
			if err != nil {
				t.Fatal(err)
			}
			return query.FromEntry(policy)
		}
		if !prove() {
			t.Fatal("baseline handoff not recognized")
		}
		changed := false
		for _, node := range graph.Nodes {
			if node.Category != pdg.Call {
				continue
			}
			attrs := graph.NodeAttributes(node)
			if graph.Tables.Text(attrs.Callee.Value) != "example.consume" {
				continue
			}
			set := graph.Attributes[node.Attributes-1]
			graph.AttributeTables.Extensions[set.Extensions-1] = slices.DeleteFunc(slices.Clone(attrs.Extensions), func(extension pdg.Extension) bool {
				if graph.Tables.Text(extension.Name) == fact {
					changed = true
					return true
				}
				return false
			})
		}
		if !changed {
			t.Fatal("missing boundary fact to remove")
		}
		if prove() {
			t.Fatalf("missing %s replaced by source spelling", fact)
		}
	}
}

func TestNativeHandoffAbortProbe(t *testing.T) {
	mode := os.Getenv("GSTYLE_HANDOFF_ABORT")
	if mode == "" {
		return
	}
	handoff := func() { fmt.Println("handoff-reached") }
	if mode == "before" {
		defer handoff()
		defer os.Exit(3)
	} else {
		defer os.Exit(3)
		defer handoff()
	}
}
func TestNativeAbortCanPreventDeferredHandoff(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"before", "after"} {
		command := exec.Command(executable, "-test.run=^TestNativeHandoffAbortProbe$")
		command.Env = append(os.Environ(), "GSTYLE_HANDOFF_ABORT="+mode)
		out, err := command.CombinedOutput()
		failure, ok := err.(*exec.ExitError)
		if !ok || failure.ExitCode() != 3 {
			t.Fatalf("exit %v: %s", err, out)
		}
		if strings.Contains(string(out), "handoff-reached") != (mode == "after") {
			t.Fatalf("mode=%s: %s", mode, out)
		}
	}
}
func TestConstructedReturnRetainsResource(t *testing.T) {
	for _, value := range []string{`struct{File fs.File}{file}`, `struct{File fs.File}{File:file}`, `[]fs.File{file}`, `map[string]fs.File{"file":file}`} {
		t.Run(value, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import "io/fs";func caller(files fs.FS)(any,error){file,err:=files.Open("file");if err!=nil{return nil,err};return `+value+`,nil}`)
			fn := findFunc(pkg.Syntax[0], "caller")
			graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			released, _, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph)
			if err != nil {
				t.Fatal(err)
			}
			want := true
			if (len(released) > 0) != want {
				t.Fatalf("returned carrier=%v want %v", released, want)
			}
		})
	}
}

type handoffNativeFile struct {
	fs.File
	closed bool
}

func (file *handoffNativeFile) Close() error { file.closed = true; return nil }
func TestNativeDiscardedAliasDoesNotRelease(t *testing.T) {
	file := &handoffNativeFile{}
	func() { alias := file; _ = alias }()
	if file.closed {
		t.Fatal("discarded local alias releases resource")
	}
}

func TestResourceDependentScalarIsNotOwnershipReturn(t *testing.T) {
	pkg := loadTestPackage(t, `package p;import "io/fs";func caller(files fs.FS)(bool,error){file,err:=files.Open("file");if err!=nil{return false,err};return file==nil,nil}`)
	fn := findFunc(pkg.Syntax[0], "caller")
	graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	released, sites, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(released) != 0 || len(sites) != 1 {
		t.Fatalf("scalar dependency transfers ownership: released=%v sites=%v", released, sites)
	}
}

func TestUnhashableCarrierCanFaultBeforeReturn(t *testing.T) {
	pkg := loadTestPackage(t, `package p;import "io/fs";var key any;func caller(files fs.FS)(any,error){file,err:=files.Open("file");if err!=nil{return nil,err};return map[any]fs.File{key:file},nil}`)
	fn := findFunc(pkg.Syntax[0], "caller")
	graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	released, sites, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(released) != 0 || len(sites) != 1 {
		t.Fatalf("possible hash fault bypassed: release=%v sites=%v", released, sites)
	}
}

func TestHandoffInputInventoryOwnsResults(t *testing.T) {
	graph := parseIR(t, `package p;func f(resource *T){consume(resource)};type T struct{};func(*T)Close(){};func consume(*T){}`, pdg.NewBuilder())
	calls, err := callflow.NewCallSites(&graph)
	if err != nil {
		t.Fatal(err)
	}
	checked := false
	for i, node := range graph.Nodes {
		ref := pdg.Ref(i + 1)
		if node.Category != pdg.Call || !calls.EntersFunction(ref) {
			continue
		}
		positions, known := calls.ResourceInputs(ref)
		if !known || len(positions) != 1 || positions[0] != 1 {
			t.Fatalf("inventory=%v %v", positions, known)
		}
		positions[0] = 99
		again, _ := calls.ResourceInputs(ref)
		if again[0] != 1 {
			t.Fatal("returned positions mutate call boundary")
		}
		checked = true
	}
	if !checked {
		t.Fatal("missing recipient call")
	}
}

func TestUnresolvedGlobalStoreCannotProveHandoff(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := parseIR(t, `package p;var held *T;func f(resource *T){held=resource};type T struct{};func(*T)Close(){}`, pool)
	policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol, TransferCalls: true}
	prove := func() bool {
		query, err := resourceflow.New(&graph)
		if err != nil {
			t.Fatal(err)
		}
		return query.FromEntry(policy)
	}
	if !prove() {
		t.Fatal("baseline global handoff not recognized")
	}
	reason := pool.Text("uncertain store target")
	graph.Tables = pool.Tables
	changed := false
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		writes := slices.Clone(attrs.Writes)
		for i := range writes {
			symbol := writes[i].Symbol
			if symbol == 0 || graph.Symbols[symbol-1].Role != pdg.GlobalRole {
				continue
			}
			writes[i].UnresolvedReason = reason
			changed = true
		}
		if len(writes) == 0 {
			continue
		}
		set := graph.Attributes[node.Attributes-1]
		graph.AttributeTables.Accesses[set.Writes-1] = writes
	}
	if !changed {
		t.Fatal("missing global store")
	}
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	if prove() {
		t.Fatal("unresolved store proves global handoff")
	}
}
