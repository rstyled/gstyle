package goquality

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestBinaryTypedBufferOwnership(t *testing.T) {
	var source strings.Builder
	source.WriteString("package repro\nimport(\"encoding/binary\";\"slices\")\n")
	methods := []string{"PutUint16", "PutUint32", "PutUint64", "AppendUint16", "AppendUint32", "AppendUint64"}
	for _, order := range []string{"LittleEndian", "BigEndian"} {
		for _, method := range methods {
			fmt.Fprintf(&source, "func Local%s%s(){ buffer:=make([]byte,8); binary.%s.%s(buffer,1) }\n", order, method, order, method)
			fmt.Fprintf(&source, "func Shared%s%s(buffer []byte){ binary.%s.%s(buffer,1) }\n", order, method, order, method)
		}
		fmt.Fprintf(&source, "func Read%s(buffer []byte) uint64 {return binary.%s.Uint64(buffer)}\n", order, order)
	}
	source.WriteString(`func LocalArray(){var buffer [8]byte;binary.LittleEndian.PutUint64(buffer[:],1)}
func SelfAppend(){buffer:=make([]byte,0,8);buffer=binary.LittleEndian.AppendUint64(buffer,1)}
func SelfDelete(){values:=[]int{1,2};values=slices.Delete(values,0,1)}
func SelfReplace(){values:=[]int{1,2};values=slices.Replace(values,0,1,3)}
func SharedSelfAppend(input []byte){buffer:=input;buffer=binary.LittleEndian.AppendUint64(buffer,1)}
func SharedSelfDelete(input []int){values:=input;values=slices.Delete(values,0,1)}
func SharedSelfReplace(input []int){values:=input;values=slices.Replace(values,0,1,3)}
func Interface(order binary.ByteOrder,buffer []byte){order.PutUint64(buffer,1)}
func AppendInterface(order binary.AppendByteOrder,buffer []byte){order.AppendUint64(buffer,1)}
func Converted(buffer []byte){var order binary.ByteOrder=binary.LittleEndian;order.PutUint64(buffer,1)}
`)
	root := qualitySourceModule(t, []byte(source.String()))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, order := range []string{"LittleEndian", "BigEndian"} {
		for _, method := range methods {
			callee := "encoding/binary." + strings.ToLower(order[:1]) + order[1:] + "." + method
			for _, storage := range []string{"Local", "Shared"} {
				name := storage + order + method
				f := findFunction(t, facts, name)
				call := f.Calls[len(f.Calls)-1]
				want := quality.Local
				if storage == "Shared" {
					want = quality.External
				}
				if call.Callee != callee || call.Dynamic || call.MutationTarget != want {
					t.Fatalf("%s: %+v", name, call)
				}
				result := resultNamed(t, report, name)
				mutated := false
				for _, effect := range result.Effects {
					mutated = mutated || effect.Kind == quality.MutationEffect
				}
				if mutated != (storage == "Shared") {
					t.Fatalf("%s mutation effects: %+v", name, result.Effects)
				}
			}
		}
		read := findFunction(t, facts, "Read"+order)
		if len(read.Calls) != 1 || read.Calls[0].MutationTarget != "" || quality.ClassifyCallee(read.Calls[0].Callee) != quality.None {
			t.Fatalf("read query: %+v", read.Calls)
		}
	}
	localArray := findFunction(t, facts, "LocalArray")
	if localArray.Calls[0].MutationTarget != quality.Local {
		t.Fatalf("local array slice loses ownership: %+v", localArray.Calls)
	}
	for _, name := range []string{"SelfAppend", "SelfDelete", "SelfReplace"} {
		f := findFunction(t, facts, name)
		call := f.Calls[len(f.Calls)-1]
		if call.MutationTarget != quality.Local {
			t.Fatalf("owned self result loses ownership %s: %+v", name, call)
		}
	}
	for _, name := range []string{"SharedSelfAppend", "SharedSelfDelete", "SharedSelfReplace"} {
		f := findFunction(t, facts, name)
		call := f.Calls[len(f.Calls)-1]
		if call.MutationTarget != quality.Unknown {
			t.Fatalf("shared self result becomes local %s: %+v", name, call)
		}
	}
	override := quality.DefaultConfig()
	for _, callee := range []string{"encoding/binary.littleEndian.AppendUint64", "slices.Delete", "slices.Replace"} {
		override.Prefixes = append(override.Prefixes, quality.Prefix{Path: callee, Kind: quality.IO})
	}
	changed, err := quality.Evaluate(facts, override)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"SelfAppend", "SelfDelete", "SelfReplace"} {
		got := resultNamed(t, changed, name)
		io := false
		for _, effect := range got.Effects {
			io = io || effect.Kind == quality.IO
		}
		if !io {
			t.Fatalf("self-result ownership hides IO override %s: %+v", name, got)
		}
	}
	for _, tc := range []struct{ name, callee string }{
		{"Interface", "encoding/binary.ByteOrder.PutUint64"},
		{"AppendInterface", "encoding/binary.AppendByteOrder.AppendUint64"},
		{"Converted", "encoding/binary.ByteOrder.PutUint64"},
	} {
		f := findFunction(t, facts, tc.name)
		call := f.Calls[len(f.Calls)-1]
		if call.Callee != tc.callee || !call.Dynamic || call.MutationTarget != "" {
			t.Fatalf("interface identity %s: %+v", tc.name, call)
		}
		result := resultNamed(t, report, tc.name)
		unknown := false
		for _, effect := range result.Effects {
			unknown = unknown || effect.Kind == quality.UnknownEffect
		}
		if !unknown || result.Complete || result.UnclassifiedCalls != 1 {
			t.Fatalf("interface call hides dispatch: %+v", result)
		}
	}
}

func TestBinaryMethodExpressionUsesBufferAfterReceiver(t *testing.T) {
	source := `package binary
 type littleEndian struct{}
 func(littleEndian)PutUint64(b []byte,v uint64){}
 func(littleEndian)AppendUint64(b []byte,v uint64)[]byte{return b}
 func caller(buffer []byte){littleEndian.PutUint64(littleEndian{},buffer,1);buffer=littleEndian.AppendUint64(littleEndian{},buffer,1)}
 `
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "binary.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Uses: map[*ast.Ident]types.Object{}, Defs: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	pkg, err := new(types.Config).Check("encoding/binary", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[len(file.Decls)-1].(*ast.FuncDecl)
	parameter := info.Defs[fn.Type.Params.List[0].Names[0]]
	x := extractor{pkg: &packages.Package{Types: pkg, TypesInfo: info, Fset: fset}, parameters: map[types.Object]bool{parameter: true}}
	var calls []*ast.CallExpr
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			calls = append(calls, call)
		}
		return true
	})
	for _, call := range calls {
		object := x.calledObject(unwrappedCallee(call.Fun)).(*types.Func)
		if x.containerMutationTarget(object, call) != quality.External {
			t.Fatalf("method receiver mistaken for buffer: %+v", call)
		}
	}
	if !x.containerSelfAssignment(parameter, calls[1]) {
		t.Fatal("explicit receiver prevents self-buffer identity")
	}
}
