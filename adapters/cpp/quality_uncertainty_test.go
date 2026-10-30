package cpp

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/quality"
)

func TestCPPFilePathUncertaintyHasSourceEvidence(t *testing.T) {
	extraction, err := Load(context.Background(), "testdata/filepath.graphson.json")
	if err != nil {
		t.Fatal(err)
	}
	result := extraction.Report().Quality.Report.Functions[0]
	unknown, locale := 0, 0
	for _, effect := range result.Effects {
		if effect.Kind == quality.UnknownEffect {
			unknown++
		}
		if effect.Line == 129 && effect.Kind == quality.GlobalRead && effect.Detail == "cpp::std.isalpha:int(int)" {
			locale++
		}
		if effect.Detail == "unresolved C++ object lifetime: Text" {
			t.Fatal("a type qualifier is treated as a runtime object")
		}
	}
	if unknown != 27 || locale != 1 || result.Complete {
		t.Fatalf("qualified ctype or remaining missing types are misclassified: unknown=%d locale=%d complete=%v", unknown, locale, result.Complete)
	}
	graph := extraction.Functions[0].Graph
	for _, node := range graph.Nodes {
		if graph.Tables.Text(node.ID) == "30064781641" && graph.NodeAttributes(node).Callee.Status != pdg.Unknown {
			t.Fatal("an API effect model invents a canonical call target")
		}
	}
}

func TestCPPPredicateTypesPropagateWithoutInventingOverloadEffects(t *testing.T) {
	for _, typ := range []string{"int", "Widget", "ANY"} {
		t.Run(typ, func(t *testing.T) {
			p := &projection{types: map[*vertex]string{}}
			operand := &vertex{Label: "IDENTIFIER", Props: map[string]any{"TYPE_FULL_NAME": typ}}
			literal := &vertex{Label: "LITERAL", Props: map[string]any{"TYPE_FULL_NAME": "int"}}
			comparison := &vertex{Label: "CALL", Props: map[string]any{"NAME": "<operator>.equals"}, Children: []*vertex{operand, literal}}
			conjunction := &vertex{Label: "CALL", Props: map[string]any{"NAME": "<operator>.logicalAnd"}, Children: []*vertex{comparison, comparison}}
			if got := p.expressionType(conjunction); (got == "bool") != (typ == "int") {
				t.Fatalf("operand %s produces %s", typ, got)
			}
			if p.opaqueOperator(conjunction) != (typ != "int") {
				t.Fatal("predicate propagation hides an unproved overloaded operator")
			}
		})
	}
	p := &projection{types: map[*vertex]string{}}
	size := &vertex{Label: "CALL", Props: map[string]any{"NAME": "<operator>.sizeOf"}}
	comparison := &vertex{Label: "CALL", Props: map[string]any{"NAME": "<operator>.equals"}, Children: []*vertex{size, size}}
	if p.expressionType(comparison) != "bool" || p.opaqueOperator(comparison) {
		t.Fatal("sizeof results are treated as possible class objects")
	}
}

func TestCPPCtypeModelRequiresQualifiedSingleArgumentCall(t *testing.T) {
	for _, test := range []struct {
		code, dispatch string
		arguments      int
		modeled        bool
	}{
		{"std::isalpha(value)", "STATIC_DISPATCH", 1, true},
		{" ::std::isalpha (value)", "STATIC_DISPATCH", 1, true},
		{"std::isalpha(value, locale)", "STATIC_DISPATCH", 2, false},
		{"isalpha(value)", "STATIC_DISPATCH", 1, false},
		{"custom::isalpha(value)", "STATIC_DISPATCH", 1, false},
		{"std::isalpha(value)", "DYNAMIC_DISPATCH", 1, false},
		{"std::isalpha", "STATIC_DISPATCH", 1, false},
	} {
		t.Run(test.code+test.dispatch, func(t *testing.T) {
			node := &vertex{Label: "CALL", Props: map[string]any{"NAME": "isalpha", "CODE": test.code, "DISPATCH_TYPE": test.dispatch}}
			for index := range test.arguments {
				node.Children = append(node.Children, &vertex{Props: map[string]any{"ARGUMENT_INDEX": json.Number(strconv.Itoa(index + 1))}})
			}
			p := &projection{doc: &document{}}
			if (p.modeledCtype(node) != "") != test.modeled {
				t.Fatal("the ctype model accepts an ambiguous call")
			}
			p.doc.Callables = map[string][]*vertex{"std.isalpha:int(int)": {{Label: "METHOD", Props: map[string]any{"IS_EXTERNAL": false}}}}
			if p.modeledCtype(node) != "" {
				t.Fatal("the foreign model overrides a source implementation")
			}
		})
	}
}

func TestCPPTypeQualifierRequiresOnlyScopeUses(t *testing.T) {
	data, err := os.ReadFile("testdata/filepath.graphson.json")
	if err != nil {
		t.Fatal(err)
	}
	var wire wireGraph
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	doc, err := indexGraph(context.Background(), wire)
	if err != nil {
		t.Fatal(err)
	}
	p := newProjection(doc, doc.Methods[0])
	for _, node := range doc.Nodes {
		if node.Label != "LOCAL" || node.text("NAME") != "Text" {
			continue
		}
		if !p.typeQualifierLocal(node) {
			t.Fatal("recorded scope-only qualifier is treated as a local object")
		}
		doc.Referrers[node] = append(doc.Referrers[node], &vertex{Label: "IDENTIFIER"})
		if p.typeQualifierLocal(node) {
			t.Fatal("a value use is hidden as a scope qualifier")
		}
		node.Props["CODE"] = "Text Text"
		if p.typeQualifierLocal(node) {
			t.Fatal("a source declaration is treated as a phantom local")
		}
		return
	}
	t.Fatal("the reduced export loses Text::npos")
}

func TestCPPSourceCallReturnTypeFeedsPredicates(t *testing.T) {
	p := newProjection(&document{}, nil)
	node := &vertex{Label: "CALL", Props: map[string]any{"NAME": "query", "TYPE_FULL_NAME": "ANY"}}
	p.refs[node] = 1
	p.attrs = []pdg.Attributes{{Callee: p.pool.Known("query:bool()")}}
	target := &vertex{Label: "METHOD", Children: []*vertex{{Label: "METHOD_RETURN", Props: map[string]any{"TYPE_FULL_NAME": "bool"}}}}
	node.Out = map[string][]*vertex{"CALL": {target}}
	if p.expressionType(node) != "bool" {
		t.Fatal("a recorded return type is lost at the call site")
	}
	p.types = map[*vertex]string{}
	node.Out["CALL"] = append(node.Out["CALL"], target)
	if p.expressionType(node) != "ANY" {
		t.Fatal("ambiguous targets get an invented return type")
	}
}
