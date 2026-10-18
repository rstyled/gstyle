package bindingflow

import (
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func replaceValueAttrs(t *testing.T, q *Query, ref pdg.Ref, category pdg.Category, attributes pdg.Attributes) {
	t.Helper()
	attrs := make([]pdg.Attributes, len(q.graph.Nodes))
	for i, node := range q.graph.Nodes {
		attrs[i] = q.graph.NodeAttributes(node)
	}
	attrs[ref-1] = attributes
	q.graph.Nodes[ref-1].Category = category
	q.graph.Attributes = nil
	q.graph.AttributeTables = pdg.AttributeTables{}
	for i := range q.graph.Nodes {
		q.graph.Nodes[i].Attributes = pdg.Ref(i + 1)
	}
	if err := pdg.PackAttributes(q.graph, attrs); err != nil {
		t.Fatal(err)
	}
}
func testText(q *Query, text string) pdg.Text {
	q.graph.Tables.Strings = append(q.graph.Tables.Strings, text)
	return pdg.Text(len(q.graph.Tables.Strings))
}
func TestPathsComposePromotedAndExplicitFields(t *testing.T) {
	q := valueQuery(t, "read", "field", "field", "field", "read", "field")
	attrs := q.graph.NodeAttributes(q.graph.Nodes[4])
	attrs.Reads = []pdg.Access{{Symbol: 2, Position: 1}}
	replaceValueAttrs(t, q, 5, pdg.Read, attrs)
	q.fields[2] = []uint32{1}
	q.fields[3] = []uint32{2}
	q.fields[4] = []uint32{1, 2}
	q.fields[6] = []uint32{1, 2}
	valueEdge(q, 1, 2, 1)
	valueEdge(q, 2, 3, 1)
	valueEdge(q, 1, 4, 1)
	valueEdge(q, 5, 6, 1)
	explicit, promoted := q.Path(3, Facts{}), q.Path(4, Facts{})
	if !explicit.Same(promoted) || explicit.Root != 1 || !reflect.DeepEqual(explicit.Fields, []uint32{1, 2}) {
		t.Fatalf("explicit=%+v promoted=%+v", explicit, promoted)
	}
	if explicit.Same(q.Path(6, Facts{})) {
		t.Fatal("different binding roots are equated")
	}
	if !q.Path(2, Facts{}).PrefixOf(explicit) {
		t.Fatal("receiver prefix is lost")
	}
	promoted.Fields[0] = 9
	if q.Path(4, Facts{}).Fields[0] != 1 {
		t.Fatal("returned path mutates borrowed metadata")
	}
	delete(q.operands[3], 1)
	if q.Path(4, Facts{}).Root != 0 {
		t.Fatal("missing base uses a source fallback")
	}
}
func TestPathsRequireFieldMetadataAndIdentityInputs(t *testing.T) {
	q := valueQuery(t, "read", "conversion", "field")
	q.fields[3] = []uint32{2}
	valueEdge(q, 1, 2, 1)
	valueEdge(q, 2, 3, 1)
	facts := Facts{Nodes: map[pdg.Ref]Fact{2: {Conversion: IdentityConversion}}}
	if got := q.Path(3, facts); got.Root != 1 || !reflect.DeepEqual(got.Fields, []uint32{2}) {
		t.Fatalf("path=%+v", got)
	}
	delete(q.operands[1], 1)
	if q.Path(3, facts).Root != 0 {
		t.Fatal("missing conversion input invents storage")
	}
	valueEdge(q, 1, 2, 1)
	delete(q.fields, 3)
	if q.Path(3, facts).Root != 0 {
		t.Fatal("field without compiler path invents identity")
	}
}
func TestDependencyFactBoundaries(t *testing.T) {
	q := valueQuery(t, "read")
	for _, value := range []string{`null`, `true`, `{}`, `[-1]`, `[4294967296]`} {
		if _, err := numericFact(value); err == nil {
			t.Fatalf("invalid fact %s admitted", value)
		}
	}
	for _, positions := range [][]uint32{{0}, {1, 1}} {
		if _, err := operandPositions(positions); err == nil {
			t.Fatalf("invalid positions %v admitted", positions)
		}
	}
	field := pdg.Extension{Name: testText(q, "gstyle.go:field-path"), Value: testText(q, `[]`)}
	if err := q.dependencyFact(1, field); err == nil {
		t.Fatal("empty field selection admitted")
	}
	field.Value = testText(q, `[0,2]`)
	if err := q.dependencyFact(1, field); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(q.fields[1], []uint32{0, 2}) {
		t.Fatal("field indexes lost")
	}
	operand := pdg.Extension{Name: testText(q, "gstyle.go:value-operands"), Value: testText(q, `[]`)}
	if err := q.dependencyFact(1, operand); err != nil {
		t.Fatal(err)
	}
	if _, known := q.required[1]; !known {
		t.Fatal("empty input inventory becomes missing")
	}
}
