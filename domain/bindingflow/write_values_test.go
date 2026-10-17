package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestZeroValueNeedsOneNilableTypeFact(t *testing.T) {
	for _, tc := range []struct {
		facts []string
		want  bool
	}{
		{[]string{`"interface"`}, true}, {[]string{`"pointer"`}, true}, {[]string{`"slice"`}, true},
		{nil, false}, {[]string{`"aggregate"`}, false}, {[]string{`null`}, false},
		{[]string{`"interface"`, `"aggregate"`}, false}, {[]string{`"interface"`, `"interface"`}, false},
	} {
		pool := pdg.NewBuilder()
		var extensions []pdg.Extension
		for _, value := range tc.facts {
			extensions = append(extensions, pdg.Extension{Name: pool.Text("gstyle.memory:kind"), Value: pool.Text(value)})
		}
		query := &Query{graph: &pdg.Graph{Tables: pool.Tables, Symbols: []pdg.Symbol{{Extensions: extensions}}}}
		if got := query.ZeroValue(1).Nil; got != tc.want {
			t.Fatalf("facts=%v nil=%v want %v", tc.facts, got, tc.want)
		}
	}
}
