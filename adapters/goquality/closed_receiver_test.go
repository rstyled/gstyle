package goquality

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestClosedReceiverStorageBoundaries(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "encoding/json"
type cursor struct{ n int }
func(p *cursor) ClosedCursor(){p.n++}
func cursorRun(){p:=new(cursor);p.ClosedCursor()}
type problem struct{ message string }
func(p *problem) ClosedError(){p.message="error"}
func errorRun(){p:=&problem{};p.ClosedError()}
type returned struct{ n int }
func(p *returned) Returned(){p.n++}
func returnedRun()any{p:=new(returned);return p}
type holder struct{ n int }
func(p *holder) PointerHolder(){p.n++}
func exposeHolder()any{p:=new(*holder);json.Unmarshal([]byte("{}"),p);return *p}
type global struct{ n int }
var globalValue=new(global)
func(p *global) InferredGlobal(){p.n++}
type fresh struct{ n int }
func(p *fresh) FreshEscape(){p.n++}
func freshRun()any{return &fresh{}}
type parameter struct{ n int }
func(p *parameter) PublicParameter(){p.n++}
func ExposeParameter(p *parameter){}
type result struct{ n int }
func(p *result) PublicResult(){p.n++}
func ExposeResult()*result{return new(result)}
type methodValue struct{ n int }
func(p *methodValue) MethodValue(){p.n++}
func methodRun()func(){p:=new(methodValue);return p.MethodValue}
type captured struct{ n int }
func(p *captured) Captured(){p.n++}
func captureRun()func(){p:=new(captured);return func(){p.Captured()}}
type concurrent struct{ n int }
func(p *concurrent) Concurrent(){p.n++}
func concurrentRun(){p:=new(concurrent);go p.Concurrent()}
type deferred struct{ n int }
func(p *deferred) Deferred(){p.n++}
func deferredRun(){p:=new(deferred);defer p.Deferred()}
type addressed struct{ n int }
var escapedInt *int
func(p *addressed) FieldAddress(){p.n++}
func addressRun(){p:=new(addressed);escapedInt=&p.n}
type arrayView struct{ n int; values [2]int }
var escapedSlice []int
func(p *arrayView) ArrayView(){p.n++}
func viewRun(){p:=new(arrayView);escapedSlice=p.values[:]}
type base struct{}
var escapedBase *base
func(p *base) Expose(){escapedBase=p}
type promoted struct{base;n int}
func(p *promoted) Promoted(){p.n++}
func promotedRun(){p:=new(promoted);p.Expose()}
type scalar int
var escapedScalar *scalar
func(p *scalar) Save(){escapedScalar=p}
type scalarField struct{ value scalar; n int }
func(p *scalarField) ScalarPointerMethod(){p.n++}
func scalarRun(){p:=new(scalarField);p.value.Save()}
type aliasScalar int
type aliasPointer = *aliasScalar
var escapedAlias aliasPointer
func(p aliasPointer) SaveAlias(){escapedAlias=p}
type aliasField struct{ value aliasScalar; n int }
func(p *aliasField) AliasedPointerMethod(){p.n++}
func aliasRun(){p:=new(aliasField);p.value.SaveAlias()}
type sharedBase struct{ count int }
type embeddedPointer struct{*sharedBase}
func(p *embeddedPointer) PromotedWrite(){p.count++}
func promotedWriteRun(shared *sharedBase){p:=&embeddedPointer{shared};p.PromotedWrite()}
type aliasTarget struct{ n int }
type publicAlias = aliasTarget
func(p *aliasTarget) TypeAlias(){p.n++}
func aliasTargetRun(){p:=new(aliasTarget);p.TypeAlias()}
type borrowed struct{ n int; values map[string]int }
func(p *borrowed) BorrowedMap(){p.values["key"]=1}
func borrowedRun(values map[string]int){p:=new(borrowed);p.values=values;p.BorrowedMap()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		want quality.Provenance
	}{
		{"ClosedCursor", quality.Local}, {"ClosedError", quality.Local},
		{"Returned", quality.External}, {"PointerHolder", quality.External}, {"InferredGlobal", quality.External}, {"FreshEscape", quality.External},
		{"PublicParameter", quality.External}, {"PublicResult", quality.External}, {"MethodValue", quality.External},
		{"Captured", quality.External}, {"Concurrent", quality.External}, {"Deferred", quality.External},
		{"FieldAddress", quality.External}, {"ArrayView", quality.External}, {"Promoted", quality.External},
		{"ScalarPointerMethod", quality.External}, {"AliasedPointerMethod", quality.External}, {"PromotedWrite", quality.External}, {"TypeAlias", quality.External}, {"BorrowedMap", quality.External},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fact := findFunction(t, facts, tc.name)
			if len(fact.Mutations) != 1 || fact.Mutations[0].Provenance != tc.want {
				t.Fatalf("mutations=%+v, want one %s write", fact.Mutations, tc.want)
			}
		})
	}
}

func TestClosedReceiverTestVariantEscape(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type testCursor struct{n int}
func(p *testCursor) TestVariantWrite(){p.n++}
func run(){p:=new(testCursor);p.TestVariantWrite()}
`))
	if err := os.WriteFile(filepath.Join(root, "input_test.go"), []byte(`package repro
var escapedTest any
func init(){p:=new(testCursor);escapedTest=p}
`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tests := range []bool{false, true} {
		facts, err := (Analyzer{Root: root, Tests: tests}).Extract(context.Background(), []string{"input.go"})
		if err != nil {
			t.Fatal(err)
		}
		want := quality.Local
		if tests {
			want = quality.External
		}
		fact := findFunction(t, facts, "TestVariantWrite")
		if len(fact.Mutations) != 1 || fact.Mutations[0].Provenance != want {
			t.Fatalf("tests=%t mutations=%+v, want %s", tests, fact.Mutations, want)
		}
	}
}
