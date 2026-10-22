package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

type assuranceFamily struct {
	name, body string
	writes     int
	holdout    bool
}

// Freeze these families before examining their analyzer results. Each case has
// one scalar write per source site and no effectful place evaluation.
func assuranceFamilies() []assuranceFamily {
	return []assuranceFamily{
		{"borrowed-pointer", "local := shared\nlocal.Count++", 1, false},
		{"private-pointer", "local := new(State)\nlocal.Count++", 0, false},
		{"borrowed-slice", "local := shared.Items\nlocal[0]++", 1, false},
		{"cloned-scalar-slice", "local := append([]int(nil), shared.Items...)\nlocal[0]++", 0, false},
		{"container-shared-element", "local := []*State{shared}\nlocal[0].Count++", 1, false},
		{"copied-container-shared-element", "local := append([]*State(nil), shared)\nlocal[0].Count++", 1, false},
		{"mixed-rebind", "local := new(State)\nif shared.Count >= 0 { local = shared }\nlocal.Count++", -1, false},
		{"indirect-rebind", "local := new(State)\nholder := &local\n*holder = shared\nlocal.Count++", 1, false},
		{"borrowed-helper-return", "local := assuranceBorrow(shared)\nlocal.Count++", 1, false},
		{"private-helper-return", "local := assuranceFresh()\nlocal.Count++", 0, false},
		{"inline-callback", "local := func(value *State) { value.Count++ }\nlocal(shared)", 1, false},
		{"restore-with-io", "local := shared\nlocal.Count++\nlocal.Count--\nprintln(\"unrelated IO\")", 2, false},
		{"holdout-reverse", "local := shared.Items\ntemp := local[0]\nlocal[0] = local[1]\nlocal[1] = temp", 2, true},
		{"holdout-swap-fields", "temp := shared.Count\nshared.Count = shared.Array[0]\nshared.Array[0] = temp", 2, true},
		{"holdout-detached-alias", "local := shared.Items\nshared.Items = nil\nlocal[0]++", 1, true},
		{"holdout-restoring-helper", "assuranceRestore(shared)", 2, true},
		{"same-value-control", "shared.Count = shared.Count", 1, false},
	}
}

func assuranceCorpus(t *testing.T) ([]specimen, map[string]assuranceFamily) {
	t.Helper()
	var cases []specimen
	families := map[string]assuranceFamily{}
	for index, family := range assuranceFamilies() {
		variants := []string{"base", "parentheses", "rename", "comments", "discard", "combined"}
		if family.name == "same-value-control" {
			variants = variants[:1]
		}
		for _, variant := range variants {
			body, err := variantBody(family.body, variant)
			if err != nil {
				t.Fatal(err)
			}
			name := fmt.Sprintf("Assure%d_%s", index, variant)
			cases = append(cases, specimen{Name: name, Base: fmt.Sprintf("Assure%d_base", index), Family: family.name, Variant: variant, Body: body})
			families[name] = family
		}
	}
	return cases, families
}

func assuranceProgram(cases []specimen) []byte {
	return append(program(cases), []byte("\nfunc assuranceBorrow(value *State) *State { return value }\nfunc assuranceFresh() *State { return new(State) }\nfunc assuranceRestore(value *State) {\nvalue.Count++\nvalue.Count--\n}\n")...)
}

// Reduce an extracted witness by deleting unreachable helper declarations. Go
// compilation and the runtime oracle must still confirm the same missing write.
func assuranceReducedProgram(t *testing.T, c specimen) []byte {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "input.go", assuranceProgram([]specimen{c}), 0)
	if err != nil {
		t.Fatal(err)
	}
	functions := map[string]*ast.FuncDecl{}
	for _, decl := range file.Decls {
		if function, ok := decl.(*ast.FuncDecl); ok {
			functions[function.Name.Name] = function
		}
	}
	needed := map[string]bool{}
	var visit func(string)
	visit = func(name string) {
		function := functions[name]
		if function == nil || needed[name] {
			return
		}
		needed[name] = true
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if id, ok := ast.Unparen(call.Fun).(*ast.Ident); ok {
					visit(id.Name)
				}
			}
			return true
		})
	}
	visit(c.Name)
	var declarations []ast.Decl
	for _, decl := range file.Decls {
		if function, ok := decl.(*ast.FuncDecl); ok && !needed[function.Name.Name] {
			continue
		}
		declarations = append(declarations, decl)
	}
	file.Decls = declarations
	var out bytes.Buffer
	if err := format.Node(&out, fset, file); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

type assuranceWitness struct {
	Target   string `json:"target"`
	Source   int    `json:"source_line"`
	CaseLine int    `json:"case_line"`
}

type assuranceObservation struct {
	Values map[string]int     `json:"values"`
	Length int                `json:"length"`
	Writes []assuranceWitness `json:"writes"`
}

// This runtime code never imports Gstyle or consults its facts. Pointer identity
// decides whether an executed integer assignment touches a caller cell.
const assuranceTrace = `package repro
import("runtime";"strings")
type AssuranceWitness struct { Target string ` + "`json:\"target\"`" + `; Source int ` + "`json:\"source_line\"`" + `; CaseLine int ` + "`json:\"case_line\"`" + ` }
type AssuranceObservation struct { Values map[string]int ` + "`json:\"values\"`" + `; Length int ` + "`json:\"length\"`" + `; Writes []AssuranceWitness ` + "`json:\"writes\"`" + ` }
var assuranceCells map[*int]string
var assuranceWrites []AssuranceWitness
var assuranceActive string
func assuranceRecord(place *int, line int) {
 target,ok:=assuranceCells[place];if !ok{return}
 pcs:=make([]uintptr,32); n:=runtime.Callers(2,pcs); frames:=runtime.CallersFrames(pcs[:n])
 for { frame,more:=frames.Next(); if strings.HasSuffix(frame.Function,"."+assuranceActive) {
  assuranceWrites=append(assuranceWrites,AssuranceWitness{target,line,frame.Line});return
 };if !more {panic("active case missing from runtime stack")} }
}
func AssuranceObserve(value *State, name string, run func()) AssuranceObservation {
 assuranceCells=map[*int]string{&value.Count:"Count",&value.Items[0]:"Items[0]",&value.Items[1]:"Items[1]",&value.Array[0]:"Array[0]"}
 assuranceWrites=nil;assuranceActive=name;run()
 values:=map[string]int{};for cell,name:=range assuranceCells {values[name]=*cell}
 return AssuranceObservation{values,len(value.Items),assuranceWrites}
}
`

func assuranceOracle(cases []specimen) []byte {
	var out strings.Builder
	out.WriteString("package main\nimport(\"encoding/json\";\"os\";repro \"example.com/quality-hunt\")\nfunc main(){out:=map[string]repro.AssuranceObservation{}\n")
	for _, c := range cases {
		for _, seed := range []int{-1, 0, 7} {
			fmt.Fprintf(&out, "{value:=&repro.State{Count:%d,Items:[]int{%d,%d},Array:[1]int{%d}};out[%q]=repro.AssuranceObserve(value,%q,func(){repro.%s(value)})}\n", seed, seed, seed+1, seed+2, fmt.Sprintf("%s/%d", c.Name, seed), c.Name, c.Name)
		}
	}
	out.WriteString("if err:=json.NewEncoder(os.Stdout).Encode(out);err!=nil{panic(err)}}\n")
	return []byte(out.String())
}

type assuranceInsertion struct {
	offset int
	text   string
}

func assuranceInstrument(ctx context.Context, root string, source []byte) ([]byte, error) {
	loaded, err := packages.Load(&packages.Config{Context: ctx, Dir: root, Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedFiles}, "file="+filepath.Join(root, "input.go"))
	if err != nil {
		return nil, err
	}
	if len(loaded) != 1 || len(loaded[0].Errors) != 0 {
		return nil, fmt.Errorf("instrumentation type check failed: %v", loaded)
	}
	pkg := loaded[0]
	var inserts []assuranceInsertion
	var invalid error
	for _, file := range pkg.Syntax {
		if filepath.Base(pkg.Fset.Position(file.Pos()).Filename) != "input.go" {
			continue
		}
		astutil.Apply(file, func(cursor *astutil.Cursor) bool {
			place := assurancePlace(cursor.Node())
			if place == nil {
				return true
			}
			value := pkg.TypesInfo.Types[place]
			if !types.Identical(value.Type, types.Typ[types.Int]) || !value.Addressable() {
				return true // Reference/header assignments are outside the cell claim.
			}
			if _, ok := cursor.Parent().(*ast.BlockStmt); !ok || !assuranceSimplePlace(place) {
				invalid = fmt.Errorf("unsupported integer write at %s", pkg.Fset.Position(place.Pos()))
				return false
			}
			start := pkg.Fset.Position(cursor.Node().Pos())
			first, last := pkg.Fset.Position(place.Pos()).Offset, pkg.Fset.Position(place.End()).Offset
			text := fmt.Sprintf("assuranceRecord(&(%s),%d);", source[first:last], start.Line)
			inserts = append(inserts, assuranceInsertion{start.Offset, text})
			return true
		}, nil)
	}
	if invalid != nil {
		return nil, invalid
	}
	sort.Slice(inserts, func(i, j int) bool { return inserts[i].offset > inserts[j].offset })
	result := bytes.Clone(source)
	for _, insert := range inserts {
		result = append(result[:insert.offset], append([]byte(insert.text), result[insert.offset:]...)...)
	}
	return result, nil
}

func assurancePlace(node ast.Node) ast.Expr {
	switch node := node.(type) {
	case *ast.IncDecStmt:
		return node.X
	case *ast.AssignStmt:
		if node.Tok != token.DEFINE && len(node.Lhs) == 1 {
			return node.Lhs[0]
		}
	}
	return nil
}

func assuranceSimplePlace(value ast.Expr) bool {
	switch value := ast.Unparen(value).(type) {
	case *ast.Ident:
		return true
	case *ast.StarExpr:
		return assuranceSimplePlace(value.X)
	case *ast.SelectorExpr:
		return assuranceSimplePlace(value.X)
	case *ast.IndexExpr:
		index, ok := ast.Unparen(value.Index).(*ast.BasicLit)
		return ok && index.Kind == token.INT && assuranceSimplePlace(value.X)
	}
	return false
}

func assuranceExecute(t *testing.T, root string) map[string]assuranceObservation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45e9)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "run", "./oracle")
	command.Dir = root
	var stderr bytes.Buffer
	command.Stderr = &stderr
	data, err := command.Output()
	if err != nil {
		t.Fatalf("runtime setup/execution failed: %v: %s", err, stderr.String())
	}
	var observations map[string]assuranceObservation
	if err := json.Unmarshal(data, &observations); err != nil {
		t.Fatal(err)
	}
	return observations
}

func assuranceWrite(t *testing.T, root, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func assuranceCheckObservation(t *testing.T, name string, plain, traced assuranceObservation, family assuranceFamily, seed int) {
	t.Helper()
	if !reflect.DeepEqual(plain.Values, traced.Values) || plain.Length != traced.Length {
		t.Fatalf("instrumentation changes final state for %s", name)
	}
	writes := family.writes
	if writes == -1 {
		writes = 0
		if seed >= 0 {
			writes = 1
		}
	}
	if len(traced.Writes) != writes {
		t.Fatalf("oracle expectation failed for %s: got %d writes, want %d", name, len(traced.Writes), writes)
	}
	if family.name == "restore-with-io" || family.name == "holdout-restoring-helper" || family.name == "same-value-control" {
		if plain.Values["Count"] != seed {
			t.Fatalf("%s no longer demonstrates a write with unchanged final value", name)
		}
	}
}
