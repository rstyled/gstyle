package goquality

import (
	"context"
	"testing"
)

func TestAggregateBoundariesExcludeOnlyOwnedLocalWrites(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type A struct{Count int}
type B struct{Count int}
type Holder struct{Child *B}
type Embedded struct{*B}
var global B
func Borrowed(a *A,b *B){a.Count++;b.Count++}
func OwnedValue(a *A){var b B;b.Count++;a.Count++}
func OwnedPointer(a *A){b:=new(B);b.Count++;a.Count++}
func ValueParameter(a *A,b B){a.Count++;b.Count++}
func Aliased(a *A,b *B){alias:=b;alias.Count++;a.Count++}
func Nested(a *A,b *B){local:=Holder{Child:b};local.Child.Count++;a.Count++}
func Promoted(a *A,b *B){local:=Embedded{B:b};local.Count++;a.Count++}
func Global(a *A){global.Count++;a.Count++}
func Mixed(a *A,b *B,condition bool){local:=new(B);if condition{local=b};local.Count++;a.Count++}
func Header(a *A,b *B){local:=Holder{};local.Child=b;a.Count++}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		want int
	}{
		{"Borrowed", 2}, {"OwnedValue", 1}, {"OwnedPointer", 1}, {"ValueParameter", 1},
		{"Aliased", 2}, {"Nested", 2}, {"Promoted", 2}, {"Global", 2}, {"Mixed", 2}, {"Header", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := findFunction(t, facts, tc.name)
			if len(f.DDD) != tc.want {
				t.Fatalf("aggregate types: %+v, want %d", f.DDD, tc.want)
			}
		})
	}
}
