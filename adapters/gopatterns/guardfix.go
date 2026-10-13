package gopatterns

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/scanner"
	"go/token"
)

// GuardFix describes one applied guard-clause rewrite.
type GuardFix struct {
	// Line is the original if-statement line.
	Line int
	// Kind is always "guard_clause".
	Kind string
}

// FixLine implements Fix.
func (g GuardFix) FixLine() int { return g.Line }

// FixKind implements Fix.
func (g GuardFix) FixKind() string { return g.Kind }

// textEdit replaces src[start:end] with replacement.
type textEdit struct {
	start, end  int
	replacement []byte
}

// FixInvertedGuards rewrites inverted conditionals in f into guard clauses.
// It returns the rewritten source (gofmt-clean) and the fixes applied.
// The transform is behavior-preserving: ifStmt of the form
// `if [init;] cond { happy } else { return }` becomes
// `[init;] if !cond { return }; happy`.
// If no inverted guards are found, it returns src unchanged.
func FixInvertedGuards(fset *token.FileSet, f *ast.File, src []byte) ([]byte, []GuardFix, error) {
	var edits []textEdit
	var fixes []GuardFix

	ast.Inspect(f, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		if !isInvertedGuard(ifStmt) {
			return true
		}
		edit, fix, ok := guardFileEdit(fset, f, ifStmt, src)
		if !ok {
			return true
		}
		edits = append(edits, edit)
		fixes = append(fixes, fix)
		// Don't descend into the rewritten statement.
		return false
	})

	if len(edits) == 0 {
		return src, nil, nil
	}

	out := applyEdits(src, edits)
	formatted, err := format.Source(out)
	if err != nil {
		return nil, nil, fmt.Errorf("gofmt after guard fix: %w", err)
	}
	return formatted, fixes, nil
}

// applyEdits applies edits to src. Edits must not overlap; they are applied
// in ascending order, copying unchanged spans between them.
func applyEdits(src []byte, edits []textEdit) []byte {
	// Sort ascending by start.
	for i := 0; i < len(edits); i++ {
		for j := i + 1; j < len(edits); j++ {
			if edits[j].start < edits[i].start {
				edits[i], edits[j] = edits[j], edits[i]
			}
		}
	}
	var out bytes.Buffer
	out.Grow(len(src))
	pos := 0
	for _, e := range edits {
		out.Write(src[pos:e.start])
		out.Write(e.replacement)
		pos = e.end
	}
	out.Write(src[pos:])
	return out.Bytes()
}

// guardEdit computes the text edit that turns ifStmt into a guard clause.
// It returns false when the transform cannot be proven safe.
func guardEdit(fset *token.FileSet, ifStmt *ast.IfStmt, src []byte) (textEdit, GuardFix, bool) {
	// Else must be a plain block: an else-if chain or missing else is not
	// a guard-clause candidate the transform can handle.
	elseBlock, ok := ifStmt.Else.(*ast.BlockStmt)
	if !ok {
		return textEdit{}, GuardFix{}, false
	}

	// Invert the condition via De Morgan's laws.
	negated := negate(ifStmt.Cond)
	var condBuf bytes.Buffer
	if err := format.Node(&condBuf, fset, negated); err != nil {
		return textEdit{}, GuardFix{}, false
	}

	// Safety: if the condition contained comments, format.Node may have
	// dropped them. Bail out rather than silently deleting comments.
	if guardHeaderHasComments(ifStmt, fset, src) {
		return textEdit{}, GuardFix{}, false
	}

	// Source slices.
	stmtStart := fset.PositionFor(ifStmt.Pos(), false).Offset
	stmtEnd := fset.PositionFor(ifStmt.End(), false).Offset
	elseStart := fset.PositionFor(elseBlock.Pos(), false).Offset
	elseEnd := fset.PositionFor(elseBlock.End(), false).Offset
	bodyLbrace := fset.PositionFor(ifStmt.Body.Lbrace, false).Offset
	bodyRbrace := fset.PositionFor(ifStmt.Body.Rbrace, false).Offset

	// The else block text (including braces) becomes the new guard body.
	guardBody := src[elseStart:elseEnd]
	// The if-body inner statements (without braces) flow after.
	happyPath := bytes.TrimSpace(src[bodyLbrace+1 : bodyRbrace])

	var b bytes.Buffer
	// Hoist the init statement, if present.
	if ifStmt.Init != nil {
		if !safeInit(ifStmt.Init) {
			return textEdit{}, GuardFix{}, false
		}
		initStart := fset.PositionFor(ifStmt.Init.Pos(), false).Offset
		initEnd := fset.PositionFor(ifStmt.Init.End(), false).Offset
		b.Write(bytes.TrimSpace(src[initStart:initEnd]))
		b.WriteByte('\n')
	}
	b.WriteString("if ")
	b.Write(bytes.TrimSpace(condBuf.Bytes()))
	b.WriteByte(' ')
	b.Write(bytes.TrimSpace(guardBody))
	b.WriteByte('\n')
	b.Write(happyPath)

	line := fset.PositionFor(ifStmt.Pos(), false).Line
	return textEdit{start: stmtStart, end: stmtEnd, replacement: b.Bytes()},
		GuardFix{Line: line, Kind: "guard_clause"},
		true
}

// guardHeaderHasComments protects comments that a rewritten header would drop.
func guardHeaderHasComments(n *ast.IfStmt, fset *token.FileSet, src []byte) bool {
	start := fset.PositionFor(n.Pos(), false).Offset
	end := fset.PositionFor(n.Body.Lbrace, false).Offset
	if start < 0 || end > len(src) || start >= end {
		return false
	}
	return conditionHasComments(src[start:end])
}

func conditionHasComments(segment []byte) bool {
	var lexer scanner.Scanner
	file := token.NewFileSet().AddFile("condition", -1, len(segment))
	lexer.Init(file, segment, nil, scanner.ScanComments)
	for {
		_, kind, _ := lexer.Scan()
		if kind == token.COMMENT {
			return true
		}
		if kind == token.EOF {
			return false
		}
	}
}

// safeInit reports whether the init statement can be hoisted before the if.
// Only simple non-declaring statements are safe; `:=` declarations risk
// shadowing an outer variable, so they are rejected.
func safeInit(init ast.Stmt) bool {
	switch s := init.(type) {
	case *ast.AssignStmt:
		return s.Tok != token.DEFINE
	case *ast.ExprStmt, *ast.IncDecStmt, *ast.SendStmt:
		return true
	default:
		return false
	}
}

// negate returns the AST for !expr, applying De Morgan's laws and
// comparison inversions so the result reads naturally.
func negate(expr ast.Expr) ast.Expr {
	if neg, ok := negateUnary(expr); ok {
		return neg
	}
	if neg, ok := negateBinary(expr); ok {
		return neg
	}
	return &ast.UnaryExpr{Op: token.NOT, X: expr}
}

// negateUnary handles !x -> x.
func negateUnary(expr ast.Expr) (ast.Expr, bool) {
	e, ok := expr.(*ast.UnaryExpr)
	if ok && e.Op == token.NOT {
		return e.X, true
	}
	return nil, false
}

// negateBinary handles comparisons, &&, ||, and parenthesized exprs.
func negateBinary(expr ast.Expr) (ast.Expr, bool) {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		if inv, ok := invertBinary(e.Op); ok {
			return &ast.BinaryExpr{X: e.X, Op: inv, Y: e.Y}, true
		}
		if e.Op == token.LAND {
			return &ast.BinaryExpr{X: negate(e.X), Op: token.LOR, Y: negate(e.Y)}, true
		}
		if e.Op == token.LOR {
			return &ast.BinaryExpr{X: negate(e.X), Op: token.LAND, Y: negate(e.Y)}, true
		}
	case *ast.ParenExpr:
		return &ast.ParenExpr{X: negate(e.X)}, true
	}
	return nil, false
}

// invertBinary maps a comparison to its negation.
func invertBinary(op token.Token) (token.Token, bool) {
	// Grouped by inversion pairs to keep the switch small.
	switch op {
	case token.EQL, token.NEQ:
		return invertEquality(op), true
	case token.LSS, token.GEQ, token.LEQ, token.GTR:
		return invertOrdering(op), true
	}
	return 0, false
}

func invertEquality(op token.Token) token.Token {
	if op == token.EQL {
		return token.NEQ
	}
	return token.EQL
}

func invertOrdering(op token.Token) token.Token {
	switch op {
	case token.LSS:
		return token.GEQ
	case token.GEQ:
		return token.LSS
	case token.LEQ:
		return token.GTR
	default: // token.GTR
		return token.LEQ
	}
}

// A rewritten else-if must stay within its original conditional branch.
func guardFileEdit(fset *token.FileSet, file *ast.File, target *ast.IfStmt, src []byte) (textEdit, GuardFix, bool) {
	edit, fix, ok := guardEdit(fset, target, src)
	if !ok {
		return edit, fix, false
	}
	if elseIfInFile(file, target) {
		edit.replacement = append([]byte("{\n"), edit.replacement...)
		edit.replacement = append(edit.replacement, []byte("\n}")...)
	}
	return edit, fix, true
}

func elseIfInFile(file *ast.File, target *ast.IfStmt) bool {
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		if statement, ok := node.(*ast.IfStmt); ok && statement.Else == target {
			found = true
		}
		return !found
	})
	return found
}
