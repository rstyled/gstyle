package runtimeguard

import (
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestExtensionFirstMatchAndBoundaries(t *testing.T) {
	pool := pdg.NewBuilder()
	pool.Tables.Sources = &pdg.SourceContext{}
	graph := &pdg.Graph{Tables: pool.Tables, Profile: 77}
	key := pool.Text("gstyle.runtime:integer")
	other := pdg.Extension{Name: pool.Text("gstyle.runtime:other"), Value: pool.Text("gstyle.runtime:integer")}
	first := pdg.Extension{Name: key, Value: pool.Text("first")}
	last := pdg.Extension{Name: key, Value: pool.Text("last")}
	empty := pdg.Extension{Name: key, Value: pool.Text("")}
	pool.Tables.Strings = append(pool.Tables.Strings, "gstyle.runtime:integer")
	duplicateKey := pdg.Text(len(pool.Tables.Strings))
	cases := []struct {
		label, name string
		rows        []pdg.Extension
		want        string
	}{
		{"first", "integer", []pdg.Extension{first, other}, "first"},
		{"last", "integer", []pdg.Extension{other, last}, "last"},
		{"duplicate", "integer", []pdg.Extension{first, last}, "first"},
		{"equal decoded keys", "integer", []pdg.Extension{{Name: duplicateKey, Value: first.Value}, last}, "first"},
		{"empty decoded duplicate", "integer", []pdg.Extension{{Name: duplicateKey, Value: empty.Value}, last}, ""},
		{"empty first", "integer", []pdg.Extension{empty, last}, ""},
		{"absent first value", "integer", []pdg.Extension{{Name: key}, last}, ""},
		{"oversized first value", "integer", []pdg.Extension{{Name: key, Value: pdg.Text(^uint32(0))}, last}, ""},
		{"no match", "integer", []pdg.Extension{other}, ""},
		{"missing names", "integer", []pdg.Extension{{Value: first.Value}, {Name: pdg.Text(^uint32(0)), Value: last.Value}, first}, "first"},
		{"namespace", "integer", []pdg.Extension{{Name: pool.Text("integer"), Value: last.Value}, first}, "first"},
		{"case", "Integer", []pdg.Extension{first}, ""},
		{"exact key", "integer", []pdg.Extension{{Name: pool.Text("gstyle.runtime:integer-more"), Value: last.Value}, first}, "first"},
		{"empty key", "", []pdg.Extension{{Name: pool.Text("gstyle.runtime:"), Value: last.Value}}, "last"},
		{"byte key", "λ\x00key", []pdg.Extension{{Name: pool.Text("gstyle.runtime:λ\x00key"), Value: pool.Text("\x00\xffpayload")}}, "\x00\xffpayload"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			rows := slices.Clone(tc.rows)
			strings := slices.Clone(pool.Tables.Strings)
			table := *pool.Tables
			before := *graph
			if got := extension(graph, tc.rows, tc.name); got != tc.want {
				t.Fatalf("got=%q want=%q", got, tc.want)
			}
			if !slices.Equal(tc.rows, rows) || !slices.Equal(pool.Tables.Strings, strings) ||
				!reflect.DeepEqual(*pool.Tables, table) || !reflect.DeepEqual(*graph, before) {
				t.Fatal("lookup changes its inputs")
			}
		})
	}
}

func TestExtensionEmptyRowsNeedNoGraph(t *testing.T) {
	for _, rows := range [][]pdg.Extension{nil, {}, make([]pdg.Extension, 0, 3)} {
		if got := extension(nil, rows, "anything"); got != "" {
			t.Fatalf("empty lookup=%q", got)
		}
	}
	for _, strings := range [][]string{nil, {}, make([]string, 0, 3)} {
		graph := &pdg.Graph{Tables: &pdg.Tables{Strings: strings}}
		if got := extension(graph, []pdg.Extension{{Name: 1, Value: 2}}, "integer"); got != "" {
			t.Fatalf("missing strings become a value: %q", got)
		}
	}
}

func TestFactUsesActualNodeAndExtensionRows(t *testing.T) {
	pool := pdg.NewBuilder()
	key := pool.Text("gstyle.runtime:integer")
	input := []pdg.Attributes{
		{Construct: 11, Extensions: []pdg.Extension{{Name: key, Value: pool.Text("first")}, {Name: key, Value: pool.Text("later")}}},
		{Construct: 22, Extensions: pool.SingletonExtension("gstyle.runtime:integer", "second")},
	}
	graph := &pdg.Graph{Tables: pool.Tables, Profile: 81, Nodes: []pdg.Node{{Attributes: 2}, {Attributes: 1}, {}}}
	if err := pdg.PackAttributes(graph, input); err != nil {
		t.Fatal(err)
	}
	view := &graphView{graph: graph, kinds: []string{"retained"}, tracked: []bool{true}}
	beforeView := *view
	nodes := slices.Clone(graph.Nodes)
	records := slices.Clone(graph.Attributes)
	rows := make([][]pdg.Extension, len(graph.AttributeTables.Extensions))
	for i := range rows {
		rows[i] = slices.Clone(graph.AttributeTables.Extensions[i])
	}
	for _, tc := range []struct {
		ref  pdg.Ref
		want string
	}{{0, ""}, {1, "second"}, {2, "first"}, {3, ""}} {
		if got := view.fact(tc.ref, "integer"); got != tc.want {
			t.Fatalf("ref=%d got=%q want=%q", tc.ref, got, tc.want)
		}
	}
	if got := view.attrs(1); got.Construct != 22 {
		t.Fatalf("wrong first node metadata: %+v", got)
	}
	if got := view.attrs(2); got.Construct != 11 {
		t.Fatalf("wrong last present node metadata: %+v", got)
	}
	if !reflect.DeepEqual(*view, beforeView) || !slices.Equal(graph.Nodes, nodes) ||
		!slices.Equal(graph.Attributes, records) || !reflect.DeepEqual(graph.AttributeTables.Extensions, rows) || graph.Profile != 81 {
		t.Fatal("fact or attribute access changes its inputs")
	}
}

func TestFactAbsentAttributesNeedNoTable(t *testing.T) {
	var empty *graphView
	if got := empty.attrs(0); !reflect.DeepEqual(got, pdg.Attributes{}) {
		t.Fatalf("zero attrs=%+v", got)
	}
	view := &graphView{}
	if got := view.fact(0, "integer"); got != "" {
		t.Fatalf("zero fact=%q", got)
	}
	for _, ref := range []pdg.Ref{0, 1, pdg.Ref(^uint32(0))} {
		view.graph = &pdg.Graph{Nodes: []pdg.Node{{Attributes: ref}}}
		if got := view.fact(1, "integer"); got != "" {
			t.Fatalf("absent ref=%d fact=%q", ref, got)
		}
	}
}
