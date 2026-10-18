package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestParameterZeroRequiresResolvedEvidence(t *testing.T) {
	pool := pdg.NewBuilder()
	q := &Query{graph: &pdg.Graph{Tables: pool.Tables}}
	boolean := pool.SingletonExtension("gstyle.go.scalar-normal:resolved-type", `{"primitive":"bool"}`)
	pointer := pool.SingletonExtension("gstyle.memory:kind", `"pointer"`)
	for _, tc := range []struct {
		name              string
		extensions        []pdg.Extension
		boolean, nilValue bool
	}{
		{"resolved bool", boolean, true, false},
		{"resolved pointer", pointer, false, true},
		{"resolved interface", pool.SingletonExtension("gstyle.memory:kind", `"interface"`), false, true},
		{"numeric scalar", pool.SingletonExtension("gstyle.go.scalar-normal:resolved-type", `{"primitive":"isize"}`), false, false},
		{"spelling alone", nil, false, false},
		{"malformed kind", pool.SingletonExtension("gstyle.memory:kind", `null`), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parameter := pdg.Parameter{Type: pool.Known("bool"), Extensions: tc.extensions}
			value := q.ParameterZero(parameter)
			_, known := value.Bool()
			if known != tc.boolean || value.Nil != tc.nilValue {
				t.Fatalf("zero=%+v", value)
			}
		})
	}
	for _, extensions := range [][]pdg.Extension{
		pool.SingletonExtension("gstyle.go.scalar-normal:resolved-type", `null`),
		pool.SingletonExtension("gstyle.go.scalar-normal:resolved-type", `{"primitive":true}`),
		append(append([]pdg.Extension{}, boolean...), boolean...),
		append(append([]pdg.Extension{}, boolean...), pointer...),
	} {
		parameter := pdg.Parameter{Extensions: extensions}
		q.graph.Function.Outputs = []pdg.Parameter{parameter}
		if _, err := q.ValueFacts(); err == nil {
			t.Fatal("invalid output type evidence accepted")
		}
		if value := q.ParameterZero(parameter); value.Nil || value.Scalar != nil {
			t.Fatalf("invalid evidence proves %+v", value)
		}
	}
	var missing *Query
	if value := missing.ParameterZero(pdg.Parameter{Extensions: boolean}); value.Nil || value.Scalar != nil {
		t.Fatal("nil query proves a zero value")
	}
}
