package callflow

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestFunctionDeclarationFactRequiresOneNonemptyString(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables}
	name := pool.Text("gstyle.go:function-value")
	for _, data := range []string{"null", `""`, "true", "{}", `["p.f"]`, "{", "7"} {
		if value, err := declarationIdentity(graph, []pdg.Extension{{Name: name, Value: pool.Text(data)}}); err == nil {
			t.Fatalf("invalid identity=%q from %s", value, data)
		}
	}
	for _, first := range []string{`""`, `"p.f"`} {
		extensions := []pdg.Extension{{Name: name, Value: pool.Text(first)}, {Name: name, Value: pool.Text(`"p.f"`)}}
		if _, err := declarationIdentity(graph, extensions); err == nil {
			t.Fatal("duplicate inventory accepted")
		}
	}
	if value, err := declarationIdentity(graph, []pdg.Extension{{Name: name, Value: pool.Text(`"p.f"`)}}); err != nil || value != "p.f" {
		t.Fatalf("known declaration=%q,%v", value, err)
	}
}
