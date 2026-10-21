package gopatterns

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/resourceflow"
)

func TestResourceWorkerEvidence(t *testing.T) {
	pkg := loadTestPackage(t, `package p
import("context";"time")
func worker(ctx context.Context,t *time.Ticker){defer t.Stop();for{select{case <-ctx.Done():return;case <-t.C:}}}
func withoutCleanup(ctx context.Context,t *time.Ticker){for{select{case <-ctx.Done():return;case <-t.C:}}}
func never(ctx context.Context,t *time.Ticker){defer t.Stop();select{}}
func finiteWorker(ctx context.Context,t *time.Ticker){defer t.Stop()}
func finite(ctx context.Context){t:=time.NewTicker(time.Second);go finiteWorker(ctx,t)}
func start(ctx context.Context){t:=time.NewTicker(time.Second);go worker(ctx,t)}
func background(){t:=time.NewTicker(time.Second);go worker(context.Background(),t)}
func missing(ctx context.Context){t:=time.NewTicker(time.Second);go withoutCleanup(ctx,t)}
func blocked(ctx context.Context){t:=time.NewTicker(time.Second);go never(ctx,t)}
type owner struct{}
func(*owner)run(ctx context.Context,t *time.Ticker){defer t.Stop();for{select{case <-ctx.Done():return;case <-t.C:}}}
func method(ctx context.Context){t:=time.NewTicker(time.Second);o:=&owner{};go o.run(ctx,t)}
func expression(ctx context.Context){t:=time.NewTicker(time.Second);o:=&owner{};go (*owner).run(o,ctx,t)}
var other *time.Ticker
var choose bool
func wrongCleanup(ctx context.Context,t *time.Ticker,other *time.Ticker){defer other.Stop();for{select{case <-ctx.Done():return;case <-t.C:}}}
func wrong(ctx context.Context){t:=time.NewTicker(time.Second);go wrongCleanup(ctx,t,other)}
func conditionalCleanup(ctx context.Context,t *time.Ticker){if choose{defer t.Stop()};for{select{case <-ctx.Done():return;case <-t.C:}}}
func conditional(ctx context.Context){t:=time.NewTicker(time.Second);go conditionalCleanup(ctx,t)}
func otherContext(ctx context.Context,t *time.Ticker){other:=context.Background();defer t.Stop();for{select{case <-other.Done():return;case <-t.C:}}}
func otherDone(ctx context.Context){t:=time.NewTicker(time.Second);go otherContext(ctx,t)}
func rebound(ctx context.Context){t:=time.NewTicker(time.Second);t=other;go worker(ctx,t)}
`)
	root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		cleanup  bool
		shutdown string
	}{
		"start":       {true, "cancellation_receive; caller_cancellation_unverified"},
		"background":  {true, "non_cancellable_context"},
		"missing":     {false, "cancellation_receive; caller_cancellation_unverified"},
		"blocked":     {true, "unresolved"},
		"method":      {true, "cancellation_receive; caller_cancellation_unverified"},
		"expression":  {true, "cancellation_receive; caller_cancellation_unverified"},
		"wrong":       {false, "cancellation_receive; caller_cancellation_unverified"},
		"conditional": {true, "cancellation_receive; caller_cancellation_unverified"},
		"otherDone":   {true, "unresolved"},
		"rebound":     {true, "cancellation_receive; caller_cancellation_unverified"},
		"finite":      {true, "unresolved"},
	}
	for _, function := range extraction.Funcs {
		name := function.Name[strings.LastIndex(function.Name, ".")+1:]
		expected, selected := want[name]
		if !selected {
			continue
		}
		if len(function.ResourceLifecycles) != 1 {
			t.Errorf("%s missing acquisition: %+v", name, function.ResourceLifecycles)
			continue
		}
		fact := function.ResourceLifecycles[0]
		if fact.State != "worker" || fact.Worker == nil {
			t.Errorf("%s missing worker handoff: %+v", name, fact)
			continue
		}
		if fact.Worker.HandoffProven || fact.Worker.CleanupProvenOnExit != (name == "finite") || fact.Worker.CleanupDeferObserved != expected.cleanup || fact.Worker.Shutdown != expected.shutdown {
			t.Errorf("%s worker=%+v want %+v", name, fact.Worker, expected)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("missing worker cases: %v", want)
	}
}

func TestUnsupportedWorkersRemainUnresolved(t *testing.T) {
	pkg := loadTestPackage(t, `package p
import "time"
func anonymous()*time.Ticker{t:=time.NewTicker(time.Second);go func(){defer t.Stop()}();return t}
func consume(t ...*time.Ticker){defer t[0].Stop()}
func variadic()*time.Ticker{t:=time.NewTicker(time.Second);go consume(t);return t}
`)
	root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"anonymous": true, "variadic": true}
	for _, function := range extraction.Funcs {
		name := function.Name[strings.LastIndex(function.Name, ".")+1:]
		if !want[name] {
			continue
		}
		if len(function.ResourceLifecycles) != 1 {
			t.Fatalf("%s missing acquisition: %+v", name, function.ResourceLifecycles)
		}
		fact := function.ResourceLifecycles[0]
		if fact.State != "unresolved" || fact.Worker != nil {
			t.Errorf("%s unsupported worker classified: %+v", name, fact)
		}
		observation, err := resourceflow.Observe(&function.Pdg)
		if err != nil {
			t.Fatal(err)
		}
		if observation.HasFlow() || observation.FromEntry(resourceflow.Policy{}) {
			t.Errorf("%s partial graph used as cleanup proof", name)
		}
		if got := resourceflow.SummarizeFactories(map[string]*resourceflow.Query{name: observation}, nil); got[name].Fresh {
			t.Errorf("%s partial graph used as factory proof", name)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("missing unsupported worker cases: %v", want)
	}
}
