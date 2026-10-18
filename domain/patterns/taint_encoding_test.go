package patterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestDecimalEncodingRequiresOneExactPrimitiveFact(t *testing.T) {
	for _, tc := range []struct {
		name  string
		facts []string
		want  bool
	}{
		{"missing", nil, false},
		{"integer", []string{`"int"`}, true},
		{"string", []string{`"string"`}, false},
		{"named", []string{`"example.Number"`}, false},
		{"malformed", []string{`int`}, false},
		{"duplicate", []string{`"int"`, `"int"`}, false},
		{"conflicting", []string{`"int"`, `"string"`}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := pdg.NewBuilder()
			var extensions []pdg.Extension
			for _, fact := range tc.facts {
				extensions = append(extensions, pdg.Extension{Name: pool.Text("gstyle.value:basic"), Value: pool.Text(fact)})
			}
			graph := &pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Attributes: 1}}, Attributes: []pdg.AttributeSet{{Extensions: 1}}, AttributeTables: pdg.AttributeTables{Extensions: [][]pdg.Extension{extensions}}}
			if got := basicInteger(graph, 1); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
