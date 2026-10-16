// Command evidence records source structure and creates an instrumented replay.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"strings"

	"golang.org/x/tools/go/cfg"
)

type span struct {
	Kind  string `json:"kind"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}
type block struct {
	ID         int32   `json:"id"`
	Kind       string  `json:"kind"`
	Live       bool    `json:"live"`
	Successors []int32 `json:"successors"`
	Nodes      []span  `json:"nodes"`
}
type evidence struct {
	AST    []span  `json:"ast"`
	CFG    []block `json:"cfg"`
	Replay string  `json:"replay"`
	Typed  bool    `json:"typed"`
}

// gstyle-allow(side_effect_density): CLI boundary reads arguments, writes evidence, and reports errors; helper IO is propagated here.
func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// gstyle-allow(side_effect_density): Evidence boundary reads selected source and writes JSON; these effects are its purpose.
func run(args []string, output io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("expected source path")
	}
	src, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, args[0], src, 0)
	if err != nil {
		return err
	}
	if _, err := new(types.Config).Check("fixture", fset, []*ast.File{f}, nil); err != nil {
		return err
	}
	fn, err := function(f)
	if err != nil {
		return err
	}
	result := evidence{AST: astSpans(fset, fn), CFG: flowBlocks(fset, fn), Replay: replay(fset, fn, src), Typed: true}
	return json.NewEncoder(output).Encode(result)
}

func function(f *ast.File) (*ast.FuncDecl, error) {
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if ok && fn.Name.Name == "branchAssignments" {
			return fn, nil
		}
	}
	return nil, fmt.Errorf("branchAssignments is missing")
}

func location(fset *token.FileSet, n ast.Node) span {
	return span{Kind: fmt.Sprintf("%T", n), Start: fset.Position(n.Pos()).Offset, End: fset.Position(n.End()).Offset}
}

func astSpans(fset *token.FileSet, fn *ast.FuncDecl) []span {
	out := []span{}
	ast.Inspect(fn, func(n ast.Node) bool {
		if n != nil {
			out = append(out, location(fset, n))
		}
		return true
	})
	return out
}
func flowBlocks(fset *token.FileSet, fn *ast.FuncDecl) []block {
	out := []block{}
	for _, b := range cfg.New(fn.Body, func(*ast.CallExpr) bool { return true }).Blocks {
		out = append(out, flowBlock(fset, b))
	}
	return out
}
func flowBlock(fset *token.FileSet, b *cfg.Block) block {
	successors := []int32{}
	for _, next := range b.Succs {
		successors = append(successors, next.Index)
	}
	nodes := []span{}
	for _, n := range b.Nodes {
		nodes = append(nodes, location(fset, n))
	}
	return block{ID: b.Index, Kind: b.Kind.String(), Live: b.Live, Successors: successors, Nodes: nodes}
}

// The original statements stay intact. Observations are inserted after writes,
// before the branch, and after the comparison. This is an instrumented replay.
func replay(fset *token.FileSet, fn *ast.FuncDecl, src []byte) string {
	start := fset.Position(fn.Pos()).Offset
	end := fset.Position(fn.End()).Offset
	inserts := map[int]string{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.DeclStmt:
			inserts[fset.Position(v.End()).Offset] = observation("initial", fset.Position(v.Pos()).Offset)
		case *ast.AssignStmt:
			inserts[fset.Position(v.End()).Offset] = observation("value", fset.Position(v.Pos()).Offset)
		case *ast.IfStmt:
			inserts[fset.Position(v.Pos()).Offset] = fmt.Sprintf("observe(\"branch\", %d, enabled, enabled);\n", fset.Position(v.Cond.Pos()).Offset)
		}
		return true
	})
	var body strings.Builder
	for i := start; i < end; i++ {
		body.WriteString(inserts[i])
		body.WriteByte(src[i])
	}
	return replayHeader + body.String() + replayFooter
}

func observation(kind string, start int) string {
	return fmt.Sprintf("; observe(%q, %d, boxed, boxed == nil);\n", kind, start)
}

const replayHeader = `package main
import ("encoding/json"; "fmt"; "os")
type Parcel struct{ Code int }
type Event struct { Kind string ` + "`json:\"kind\"`" + `; Start int ` + "`json:\"start\"`" + `; Type string ` + "`json:\"type\"`" + `; Value any ` + "`json:\"value\"`" + `; Result bool ` + "`json:\"result\"`" + ` }
var events []Event
func observe(kind string, start int, value any, result bool) { events = append(events, Event{kind, start, fmt.Sprintf("%T", value), value, result}) }
`
const replayFooter = `
func main() {
 runs := map[string][]Event{}
 for _, enabled := range []bool{true, false} {
  events = nil
  branchAssignments(enabled)
  runs[fmt.Sprint(enabled)] = events
 }
 if err := json.NewEncoder(os.Stdout).Encode(runs); err != nil { panic(err) }
}
`
