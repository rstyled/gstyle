package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strconv"
	"strings"
)

const sourceHeader = "package generated\nfunc observe(value int64) {}\n"

// parseSource admits source only when the independent tree renderer reproduces
// every byte. This excludes extra statements, changed wrappers and shadowing.
func parseSource(source string) ([]observation, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "input.go", source, 0)
	if err != nil {
		return nil, err
	}
	if !supportedSourceHeader(source, len(file.Decls)) {
		return nil, fmt.Errorf("expected the generated package header and at least one branch function")
	}
	cases, err := sourceDeclarations(file.Decls[1:])
	if err != nil {
		return nil, err
	}
	var rendered strings.Builder
	rendered.WriteString(sourceHeader)
	for _, c := range cases {
		rendered.WriteString(c.Source)
	}
	if rendered.String() != source {
		return nil, fmt.Errorf("source does not match the supported canonical branch grammar")
	}
	return cases, nil
}

func supportedSourceHeader(source string, declarations int) bool {
	return strings.HasPrefix(source, sourceHeader) && declarations >= 2
}

func sourceDeclarations(declarations []ast.Decl) ([]observation, error) {
	var cases []observation
	names := make(map[string]bool)
	for _, declaration := range declarations {
		c, err := sourceCase(declaration)
		if err != nil {
			return nil, err
		}
		if names[c.Name] || !ast.IsExported(c.Name) {
			return nil, fmt.Errorf("source function %s must be exported and unique", c.Name)
		}
		names[c.Name] = true
		cases = append(cases, c)
	}
	return cases, nil
}

func sourceCase(declaration ast.Decl) (observation, error) {
	function, ok := declaration.(*ast.FuncDecl)
	if !ok || function.Body == nil || len(function.Body.List) != 5 {
		return observation{}, fmt.Errorf("expected a canonical branch function")
	}
	p, err := sourceTree(function.Body.List[2])
	if err != nil {
		return observation{}, fmt.Errorf("%s: %w", function.Name.Name, err)
	}
	name := function.Name.Name
	return observation{Name: name, Suite: "source", Program: p, Source: p.source(name)}, nil
}

func sourceTree(statement ast.Stmt) (program, error) {
	switch statement := statement.(type) {
	case *ast.AssignStmt:
		return sourceResult(statement)
	case *ast.IfStmt:
		return sourceBranch(statement)
	default:
		return program{}, fmt.Errorf("unsupported branch statement %T", statement)
	}
}

func sourceResult(statement *ast.AssignStmt) (program, error) {
	if statement.Tok != token.ASSIGN || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
		return program{}, fmt.Errorf("expected one result assignment")
	}
	if !sourceIdentifier(statement.Lhs[0], "result") {
		return program{}, fmt.Errorf("only result can be assigned")
	}
	if sourceIdentifier(statement.Rhs[0], "x") {
		return program{Value: expression{Input: true}}, nil
	}
	value, err := sourceInteger(statement.Rhs[0])
	return program{Value: expression{Value: value}}, err
}

func sourceBranch(statement *ast.IfStmt) (program, error) {
	if statement.Init != nil {
		return program{}, fmt.Errorf("branch initializers are unsupported")
	}
	condition, err := sourceCondition(statement.Cond)
	if err != nil {
		return program{}, err
	}
	yes, err := sourceBlock(statement.Body)
	if err != nil {
		return program{}, err
	}
	no, err := sourceBlock(statement.Else)
	if err != nil {
		return program{}, err
	}
	return program{Test: &condition, Yes: &yes, No: &no}, nil
}

func sourceBlock(statement ast.Stmt) (program, error) {
	block, ok := statement.(*ast.BlockStmt)
	if !ok || len(block.List) != 1 {
		return program{}, fmt.Errorf("expected exactly one statement in each branch")
	}
	return sourceTree(block.List[0])
}

func sourceCondition(expr ast.Expr) (condition, error) {
	binary, ok := expr.(*ast.BinaryExpr)
	if !ok || !sourceIdentifier(binary.X, "x") {
		return condition{}, fmt.Errorf("expected an input-to-literal comparison")
	}
	operators := []token.Token{token.LSS, token.LEQ, token.GTR, token.GEQ, token.EQL, token.NEQ}
	if !slices.Contains(operators, binary.Op) {
		return condition{}, fmt.Errorf("unsupported comparison %s", binary.Op)
	}
	bound, err := sourceInteger(binary.Y)
	return condition{Operator: binary.Op.String(), Bound: bound}, err
}

func sourceIdentifier(expr ast.Expr, name string) bool {
	identifier, ok := expr.(*ast.Ident)
	return ok && identifier.Name == name
}

func sourceInteger(expr ast.Expr) (int64, error) {
	sign := ""
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.SUB {
		sign, expr = "-", unary.X
	}
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.INT {
		return 0, fmt.Errorf("expected a signed decimal int64 literal")
	}
	return strconv.ParseInt(sign+literal.Value, 10, 64)
}

func suppliedCorpus(path string) ([]observation, string, []int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", nil, err
	}
	cases, err := parseSource(string(data))
	if err != nil {
		return nil, "", nil, err
	}
	var tests []condition
	for _, c := range cases {
		tests = append(tests, c.Program.conditions()...)
	}
	inputs := representativeInputs([]suite{{Tests: tests}})
	return cases, string(data), inputs, nil
}

func (p program) conditions() []condition {
	if p.Test == nil {
		return nil
	}
	return append(append([]condition{*p.Test}, p.Yes.conditions()...), p.No.conditions()...)
}
