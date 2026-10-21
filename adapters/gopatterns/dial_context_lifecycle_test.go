package gopatterns

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestDialContextCleanupAndOwnership(t *testing.T) {
	pkg := loadTestPackage(t, `package p
import("context";stdnet "net")
func missing(ctx context.Context,d *stdnet.Dialer){c,err:=d.DialContext(ctx,"tcp","example.com:80");if err!=nil{return};_=c}
func deferred(ctx context.Context,d *stdnet.Dialer){c,err:=d.DialContext(ctx,"tcp","example.com:80");if err!=nil{return};defer c.Close()}
func immediate(ctx context.Context,d *stdnet.Dialer){c,err:=d.DialContext(ctx,"tcp","example.com:80");if err!=nil{return};c.Close()}
func returned(ctx context.Context,d *stdnet.Dialer)(stdnet.Conn,error){c,err:=d.DialContext(ctx,"tcp","example.com:80");if err!=nil{return nil,err};return c,nil}
func factory(ctx context.Context,d *stdnet.Dialer)(stdnet.Conn,error){return d.DialContext(ctx,"tcp","example.com:80")}
func wrappedMissing(ctx context.Context,d *stdnet.Dialer){c,err:=factory(ctx,d);if err!=nil{return};_=c}
func wrappedClosed(ctx context.Context,d *stdnet.Dialer){c,err:=factory(ctx,d);if err!=nil{return};defer c.Close()}
func expression(ctx context.Context,d *stdnet.Dialer){c,err:=(*stdnet.Dialer).DialContext(d,ctx,"tcp","example.com:80");if err!=nil{return};_=c}
func wrong(ctx context.Context,d *stdnet.Dialer,other stdnet.Conn){c,err:=d.DialContext(ctx,"tcp","example.com:80");if err!=nil{return};other.Close();_=c}
func conditional(ctx context.Context,d *stdnet.Dialer,closeIt bool){c,err:=d.DialContext(ctx,"tcp","example.com:80");if err!=nil{return};if closeIt{defer c.Close()}}
func cancelled(ctx context.Context,d *stdnet.Dialer){child,cancel:=context.WithCancel(ctx);c,err:=d.DialContext(child,"tcp","example.com:80");cancel();if err!=nil{return};_=c}
type fake struct{}
func(*fake)DialContext(context.Context,string,string)(stdnet.Conn,error){return nil,nil}
func unrelated(ctx context.Context,d *fake){c,err:=d.DialContext(ctx,"tcp","example.com:80");if err!=nil{return};_=c}
`)
	root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]patterns.ResourceState{
		"missing": {patterns.ResourceUnresolved}, "deferred": {patterns.ResourceReleased},
		"immediate": {patterns.ResourceReleased}, "returned": {patterns.ResourceReturned},
		"factory": {patterns.ResourceReturned}, "wrappedMissing": {patterns.ResourceUnresolved},
		"wrappedClosed": {patterns.ResourceReleased}, "expression": {patterns.ResourceUnresolved},
		"wrong": {patterns.ResourceUnresolved}, "conditional": {patterns.ResourceUnresolved},
		"cancelled": {patterns.ResourceUnresolved}, "unrelated": nil,
	}
	for _, function := range extraction.Funcs {
		name := function.Name[strings.LastIndex(function.Name, ".")+1:]
		expected, selected := want[name]
		if !selected {
			continue
		}
		if len(function.ResourceLifecycles) != len(expected) {
			t.Errorf("%s: got lifecycle facts %+v, want states %v", name, function.ResourceLifecycles, expected)
			continue
		}
		count := 0
		for i, fact := range function.ResourceLifecycles {
			if fact.State != expected[i] || fact.Resource != "connection" {
				t.Errorf("%s: got %+v, want connection state %s", name, fact, expected[i])
			}
			if name != "wrappedMissing" && name != "wrappedClosed" && fact.AcquireCall != "net.Dialer.DialContext" {
				t.Errorf("%s: incorrect acquisition identity: %s", name, fact.AcquireCall)
			}
			if expected[i] == patterns.ResourceUnresolved {
				count++
			}
		}
		facts := &patterns.MiningFacts{ID: function.Name, Pdg: patterns.MiningView(&function.Pdg), ResourceLifecycles: function.ResourceLifecycles}
		if violations := patterns.FindObligationViolations([]*patterns.MiningFacts{facts}); len(violations) != count {
			t.Errorf("%s: got %d warnings, want %d: %+v", name, len(violations), count, violations)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("unchecked cases: %v", want)
	}
}
