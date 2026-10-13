package gopatterns

import (
	"encoding/json"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestScalarInterfaceUsesResolvedTypes(t *testing.T) {
	graph := parseIR(t, `package example; type Count = uint32; type Signed int16; func f(x Count,y Signed,z int) Count { return x }`, pdg.NewBuilder())
	want := []string{"u32", "i16", "isize"}
	for i, input := range graph.Function.Inputs {
		if got := resolvedPrimitive(t, graph, input.Extensions); got != want[i] {
			t.Fatalf("input %d=%s; want %s", i, got, want[i])
		}
		if got := resolvedPrimitive(t, graph, graph.Symbols[input.Symbol-1].Extensions); got != want[i] {
			t.Fatalf("symbol %d=%s; want %s", i, got, want[i])
		}
	}
	if graph.Tables.Text(graph.Function.Inputs[0].Type.Value) == "u32" {
		t.Fatal("resolved evidence replaces original type spelling")
	}
	if got := resolvedPrimitive(t, graph, graph.Function.Outputs[0].Extensions); got != "u32" {
		t.Fatal(got)
	}
}

func resolvedPrimitive(t *testing.T, graph pdg.Graph, extensions []pdg.Extension) string {
	t.Helper()
	for _, extension := range extensions {
		if graph.Tables.Text(extension.Name) != resolvedTypeExtension {
			continue
		}
		var value struct {
			Primitive string `json:"primitive"`
		}
		if err := json.Unmarshal([]byte(graph.Tables.Text(extension.Value)), &value); err != nil {
			t.Fatal(err)
		}
		return value.Primitive
	}
	t.Fatal("missing resolved type evidence")
	return ""
}
