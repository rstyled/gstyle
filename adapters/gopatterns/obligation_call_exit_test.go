package gopatterns

import (
	"errors"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"io"
	"testing"
)

func resourceCallFixture(t *testing.T, body string) (*resourceflow.Query, resourceflow.Policy) {
	t.Helper()
	pkg := loadTestPackage(t, `package p
 import("runtime";"os")
 var _=runtime.Goexit;var _=os.Exit;var choose bool
 type T struct{};func(*T)Close(){}
 func f(resource *T){`+body+`}
 func normal(){}
 func fails(){panic("bad")}
 func stops(){runtime.Goexit()}
 func aborts(){os.Exit(0)}
 func branch(){if choose {fails()}}
 func unsafeBranch(){if choose {aborts()}}
 func loop(){for {}}
 func recursive(){recursive()}
 func unknown()
 `)
	summaries, err := packageResourceSummaries(pkg)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := buildGraph(pkg, findFunc(pkg.Syntax[0], "f"), "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	query, err := resourceflow.New(&graph)
	if err != nil {
		t.Fatal(err)
	}
	return query, resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol, Exits: summaries.exits, Helpers: releaseHelperIdentities(summaries.helpers)}
}
func TestResourceReleaseUsesLocalCallExitSummaries(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       bool
	}{
		{"normal without close", `normal()`, false},
		{"deferred normal without close", `defer normal()`, false},
		{"deferred panic without close", `defer fails()`, false},
		{"normal before close", `normal();resource.Close()`, true},
		{"deferred normal", `defer resource.Close();defer normal()`, true},
		{"panic unwinds", `defer resource.Close();fails()`, true},
		{"panic skips later close", `fails();resource.Close()`, false},
		{"Goexit unwinds", `defer resource.Close();stops()`, true},
		{"Goexit skips later close", `stops();resource.Close()`, false},
		{"abort skips defer", `defer resource.Close();aborts()`, false},
		{"deferred panic", `defer resource.Close();defer fails()`, true},
		{"deferred Goexit", `defer resource.Close();defer stops()`, true},
		{"deferred abort", `defer resource.Close();defer aborts()`, false},
		{"close before deferred abort", `defer aborts();defer resource.Close()`, true},
		{"ordinary mixed return panic", `defer resource.Close();branch()`, true},
		{"deferred mixed return panic", `defer resource.Close();defer branch()`, true},
		{"possible abort", `defer resource.Close();unsafeBranch()`, false},
		{"deferred possible abort", `defer resource.Close();defer unsafeBranch()`, false},
		{"infinite helper", `defer resource.Close();loop()`, false},
		{"recursive helper", `defer resource.Close();recursive()`, false},
		{"unknown helper", `defer resource.Close();unknown()`, false},
		{"deferred unknown helper", `defer resource.Close();defer unknown()`, false},
		{"stored target", `target:=fails;defer resource.Close();target()`, true},
		{"saved target", `target:=normal;defer resource.Close();defer target();target=aborts;_=target`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			query, policy := resourceCallFixture(t, tc.body)
			if got := query.FromEntry(policy); got != tc.want {
				t.Fatalf("released=%v want %v", got, tc.want)
			}
		})
	}
}

type panicBeforeClose struct{ closed bool }

func (*panicBeforeClose) Write([]byte) (int, error) { panic("write callback") }
func (file *panicBeforeClose) Close() error         { file.closed = true; return nil }
func TestOpaqueWriterCanSkipJoinedClose(t *testing.T) {
	file := &panicBeforeClose{}
	func() {
		defer func() {
			if got := recover(); got != "write callback" {
				t.Fatalf("panic=%v", got)
			}
		}()
		_, err := io.WriteString(file, "rules")
		_ = errors.Join(err, file.Close())
	}()
	if file.closed {
		t.Fatal("close unexpectedly executed after a callback panic")
	}
}

func TestResourceReleaseRejectsMissingOrInvalidExitOutcomes(t *testing.T) {
	query, policy := resourceCallFixture(t, `defer resource.Close();normal()`)
	if !query.FromEntry(policy) {
		t.Fatal("baseline normal call does not reach cleanup")
	}
	for _, outcome := range []callflow.Outcomes{0, 1 << 12, callflow.RecoversInherited, callflow.Returns | callflow.UnknownExit} {
		policy.Exits["testp.normal"] = callflow.ExitSummary{Ordinary: outcome}
		if query.FromEntry(policy) {
			t.Fatalf("invalid/incomplete exit outcomes prove cleanup: %v", outcome)
		}
	}
}

func TestResourceReleaseCallTargetStaysCanonical(t *testing.T) {
	pkg := loadTestPackage(t, `package p
 type T struct{};func(*T)Close(){}
 func f(resource *T){defer resource.Close();fails()}
 func fails(){panic(7)}
 `)
	summaries, err := packageResourceSummaries(pkg)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := buildGraph(pkg, findFunc(pkg.Syntax[0], "f"), "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol, Exits: summaries.exits}
	prove := func() bool {
		q, err := resourceflow.New(&graph)
		if err != nil {
			t.Fatal(err)
		}
		return q.FromEntry(policy)
	}
	if !prove() {
		t.Fatal("baseline local panic does not unwind cleanup")
	}
	changed := false
	for _, node := range graph.Nodes {
		if node.Category != pdg.Call {
			continue
		}
		attrs := &graph.Attributes[node.Attributes-1]
		if graph.Tables.Text(attrs.Callee.Value) == "testp.fails" {
			attrs.Callee.Status = pdg.Unknown
			changed = true
		}
	}
	if !changed {
		t.Fatal("missing target to change")
	}
	if prove() {
		t.Fatal("source spelling overrides missing canonical target")
	}
}

func TestExitSummaryDoesNotProveReturnStatePreserved(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{
		{`return resource`, true},
		{`normal();return resource`, false},
		{`if choose {normal()};return resource`, false},
		{`if false {normal()};return resource`, true},
		{`defer normal();return resource`, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;type T struct{};func f(resource *T,choose bool)*T{`+tc.body+`};func normal(){}`)
			summaries, err := packageResourceSummaries(pkg)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := buildGraph(pkg, findFunc(pkg.Syntax[0], "f"), "p.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			query, err := resourceflow.New(&graph)
			if err != nil {
				t.Fatal(err)
			}
			policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol, TransferReturn: true, Exits: summaries.exits}
			if got := query.FromEntry(policy); got != tc.want {
				t.Fatalf("transfer=%v want %v", got, tc.want)
			}
		})
	}
}

// The consumer receives a proved summary contract here; source recovery/order
// analysis is tested separately and still reports its unresolved evidence.
func TestResourceDeferredCallUsesPanicContext(t *testing.T) {
	query, policy := resourceCallFixture(t, `defer resource.Close();defer normal();panic(7)`)
	policy.Exits["testp.normal"] = callflow.ExitSummary{Ordinary: callflow.Aborts, DeferredPanic: callflow.Returns}
	if !query.FromEntry(policy) {
		t.Fatal("deferred call ignores inherited panic context")
	}
	policy.Exits["testp.normal"] = callflow.ExitSummary{Ordinary: callflow.Returns, DeferredPanic: callflow.Aborts}
	if query.FromEntry(policy) {
		t.Fatal("ordinary return contract replaces deferred panic outcome")
	}
}
