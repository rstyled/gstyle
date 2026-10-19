package goquality

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestUnitGlobalReadsRespectStorageBoundaries(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/units\n\ngo 1.26\n",
		"external/units.go": `package external
 type Unit struct{}
 func (Unit) Value() int { return 1 }
 func (*Unit) Pointer() {}
 type PointerAlias = *Unit
 func (PointerAlias) AliasPointer() {}
 var Exported Unit
 var Addressed Unit
 var Pointed Unit
 var AliasPointed Unit
 var Written Unit
 var Nonempty struct{ Count int }
 `,
		"units.go": `package units
 import "example.com/units/external"
 var local struct{}
 var shared struct{}
 func receive(value external.Unit) external.Unit { return value }
 func ValueMethod() int { return external.Exported.Value() }
 func Pass() external.Unit { return receive(external.Exported) }
 func Return() external.Unit { return external.Exported }
 func Local() struct{} { return local }
 func Address() *external.Unit { return &external.Addressed }
 func ReadAddressed() external.Unit { return external.Addressed }
 func PointerMethod() { external.Pointed.Pointer() }
 func ReadPointed() external.Unit { return external.Pointed }
 func AliasPointerMethod() { external.AliasPointed.AliasPointer() }
 func ReadAliasPointed() external.Unit { return external.AliasPointed }
 func Write() { external.Written = external.Unit{}; shared = struct{}{} }
 func ReadWritten() external.Unit { return external.Written }
 func Nonempty() int { return external.Nonempty.Count }
 `,
	}
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"units.go"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ValueMethod", "Pass", "Return", "Local"} {
		fact := findFunction(t, facts, name)
		for _, effect := range fact.Effects {
			if effect.Kind == quality.Global {
				t.Errorf("%s retains unit global dependence: %+v", name, effect)
			}
		}
	}
	for _, name := range []string{"Address", "ReadAddressed", "PointerMethod", "ReadPointed", "AliasPointerMethod", "ReadAliasPointed", "Write", "ReadWritten", "Nonempty"} {
		fact := findFunction(t, facts, name)
		found := false
		for _, effect := range fact.Effects {
			found = found || effect.Kind == quality.Global
		}
		if !found {
			t.Errorf("%s loses global storage evidence", name)
		}
	}
	value := findFunction(t, facts, "ValueMethod")
	if len(value.Calls) != 1 {
		t.Fatalf("by-value method call evidence lost: %+v", value.Calls)
	}
	write := findFunction(t, facts, "Write")
	if len(write.Effects) != 2 {
		t.Fatalf("unit global writes lost: %+v", write.Effects)
	}
	for _, effect := range write.Effects {
		if effect.Kind != quality.Global {
			t.Fatalf("unit write kind changes: %+v", effect)
		}
	}
}
