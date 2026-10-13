package gopatterns

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestDeclarationOrderFixGroupsTypesAndPreservesComments(t *testing.T) {
	src := []byte(`package example

type Service struct{}

// helper stays a helper.
func helper() {} // helper tail

// private service method.
func (s *Service) run() {}

// constructor service.
func NewService() *Service { return &Service{} }

// public service method.
func (s *Service) Run() {}

type Worker[T any] struct{}

func (w *Worker[T]) start() {}
func NewWorker[T any]() *Worker[T] { return &Worker[T]{} }
func (w *Worker[T]) Start() {}
func init() {}
func init() {}
`)
	spec := &patterns.FixSpec{Kind: patterns.DeclarationOrder, File: "example.go"}
	fixed, err := applyProjectedFix(t, spec, src)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Service", "NewService", "Run", "run", "Worker", "NewWorker", "Start", "start", "helper", "init", "init"}
	if got := declarationNames(t, fixed); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v\n%s", got, want, fixed)
	}
	for _, comment := range []string{"// helper stays a helper.\nfunc helper", "{} // helper tail", "// constructor service.\nfunc NewService", "// private service method.\nfunc (s *Service) run"} {
		if !bytes.Contains(fixed, []byte(comment)) {
			t.Errorf("comment lost: %s\n%s", comment, fixed)
		}
	}
	again, err := applyProjectedFix(t, spec, fixed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fixed, again) {
		t.Fatal("fix is not idempotent")
	}
}

func TestDeclarationOrderKeepsDirectivesAndCgo(t *testing.T) {
	for _, src := range []string{
		"package example\nfunc helper() {}\n//go:noinline\nfunc NewThing() *Thing { return nil }\ntype Thing struct{}\n",
		"package example\n/* preamble */\nimport \"C\"\nfunc helper() {}\ntype Thing struct{}\nfunc (t Thing) Run() {}\n",
	} {
		out, err := applyProjectedFix(t, &patterns.FixSpec{Kind: patterns.DeclarationOrder, File: "example.go"}, []byte(src))
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != src {
			t.Fatal("unsafe declaration moved")
		}
	}
}

func TestDeclarationOrderParseError(t *testing.T) {
	_, err := applyProjectedFix(t, &patterns.FixSpec{Kind: patterns.DeclarationOrder, File: "example.go"}, []byte("package example\nfunc ("))
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func declarationNames(t *testing.T, src []byte) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "example.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			names = append(names, d.Name.Name)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					names = append(names, ts.Name.Name)
				}
			}
		}
	}
	return names
}

func TestDeclarationOrderLeavesScalarFactoriesAsFunctions(t *testing.T) {
	src := []byte("package example\n\nfunc helper() {}\nfunc NewCount() int { return 1 }\n")
	out, err := applyProjectedFix(t, &patterns.FixSpec{Kind: patterns.DeclarationOrder, File: "example.go"}, src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src, out) {
		t.Fatalf("scalar function treated as constructor: %s", out)
	}
}

func TestDeclarationOrderKeepsSameLineDeclarations(t *testing.T) {
	src := []byte("package example\nfunc helper() {}; type Thing struct{}; func (t Thing) Run() {}\n")
	out, err := applyProjectedFix(t, &patterns.FixSpec{Kind: patterns.DeclarationOrder, File: "example.go"}, src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src, out) {
		t.Fatal("same-line declaration moved")
	}
}

func TestDeclarationOrderFileSelection(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		path   string
		inputs []string
		want   bool
	}{
		{"a.go", []string{"a.go"}, true},
		{"b.go", []string{"a.go"}, false},
		{"sub/b.go", []string{"sub"}, true},
		{"other/b.go", []string{"sub"}, false},
		{"b.go", nil, true},
	}
	for _, c := range cases {
		path := filepath.Join(root, c.path)
		if got := declarationPathSelected(root, path, c.inputs); got != c.want {
			t.Errorf("%s with %v: got %v", c.path, c.inputs, got)
		}
	}
}

func TestDeclarationOrderGroupedTypes(t *testing.T) {
	src := []byte(`package example

type (
 First struct{}
 Second struct{}
)
func (s Second) private() {}
func (f First) Public() {}
func NewSecond() Second { return Second{} }
func (s Second) Public() {}
func (f First) private() {}
func helper() {}
`)
	out, err := applyProjectedFix(t, &patterns.FixSpec{Kind: patterns.DeclarationOrder, File: "example.go"}, src)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"First", "Second", "Public", "private", "NewSecond", "Public", "private", "helper"}
	if got := declarationNames(t, out); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if len(declarationNames(t, src)) != len(declarationNames(t, out)) {
		t.Fatal("declaration count changed")
	}
}

func TestDeclarationOrderWithoutFinalNewline(t *testing.T) {
	src := []byte("package example\n\nfunc helper() {}\n\ntype Thing struct{}\n\nfunc (t Thing) private() {}\n\nfunc (t Thing) Public() {}")
	spec := &patterns.FixSpec{Kind: patterns.DeclarationOrder, File: "example.go"}
	out, err := applyProjectedFix(t, spec, src)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Thing", "Public", "private", "helper"}
	if got := declarationNames(t, out); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
