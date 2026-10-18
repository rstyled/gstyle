package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"reflect"
	"testing"
)

func TestCallableSetsOwnCanonicalNames(t *testing.T) {
	names := []string{"p.b", "p.a", "p.b"}
	set := NewCallableSet(names...)
	names[0] = "changed"
	got := set.Names()
	if !reflect.DeepEqual(got, []string{"p.a", "p.b"}) {
		t.Fatal(got)
	}
	got[0] = "changed again"
	if !reflect.DeepEqual(set.Names(), []string{"p.a", "p.b"}) {
		t.Fatal("borrowed name storage escaped")
	}
	for _, names := range [][]string{nil, {""}, {"p.a", ""}, {string([]byte{255})}} {
		if NewCallableSet(names...).Known() {
			t.Fatal("invalid names establish targets")
		}
	}
}
func TestCallableFactsRejectMalformedAndConflictingEvidence(t *testing.T) {
	for _, raw := range []string{"null", `""`, `true`, `{}`, `["p.f"]`, `{`, `7`} {
		q := fragment(t, pdg.Computation, func(pool *pdg.Builder, attrs *pdg.Attributes) {
			attrs.Extensions = pool.SingletonExtension("gstyle.go:function-value", raw)
		})
		if _, err := q.ValueFacts(); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	for _, extra := range []string{`{"nil":true}`, `{"boolean":false}`, `{"conversion":"identity"}`} {
		q := fragment(t, pdg.Computation, func(pool *pdg.Builder, attrs *pdg.Attributes) {
			attrs.Extensions = append(pool.SingletonExtension("gstyle.go:function-value", `"p.f"`), pool.SingletonExtension("gstyle.go:value-fact", extra)...)
		})
		if _, err := q.ValueFacts(); err == nil {
			t.Fatalf("conflicting callable fact %s", extra)
		}
	}
	q := fragment(t, pdg.Computation, func(pool *pdg.Builder, attrs *pdg.Attributes) {
		attrs.Extensions = append(pool.SingletonExtension("gstyle.go:function-value", `"p.f"`), pool.SingletonExtension("gstyle.go:function-value", `"p.f"`)...)
	})
	if _, err := q.ValueFacts(); err == nil {
		t.Fatal("duplicate declaration fact accepted")
	}
}
