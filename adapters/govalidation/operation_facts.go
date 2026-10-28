package govalidation

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (a *inventory) operationFacts(g *pdg.Graph, node pdg.Node, source ast.Node) error {
	if err := sourceKind(g, node, source); err != nil {
		return err
	}
	if err := sourceSemanticFacts(g, node, source); err != nil {
		return err
	}
	attrs := g.NodeAttributes(node)
	if operator := sourceOperator(source); operator != "" {
		if err := knownValue(g, attrs.Operator, operator, "operator"); err != nil {
			return err
		}
	}
	if err := sourceWriteForm(g, attrs, source); err != nil {
		return err
	}
	return a.sourceLiteral(g, node, source)
}

func sourceSemanticFacts(g *pdg.Graph, node pdg.Node, source ast.Node) error {
	switch source.(type) {
	case *ast.BlockStmt:
		return implicitReturnFacts(g, node)
	case *ast.IfStmt:
		return controlFacts(g, node, "if")
	case *ast.BinaryExpr:
		if node.Category == pdg.Control {
			return controlFacts(g, node, "short-circuit")
		}
	case *ast.ReturnStmt:
		return normalReturnFacts(g, node)
	}
	return nil
}

func controlFacts(g *pdg.Graph, node pdg.Node, construct string) error {
	attrs := g.NodeAttributes(node)
	valid := slices.EqualFunc(attrs.Outcomes, []string{"true", "false"}, func(actual pdg.Text, expected string) bool {
		return g.Tables.Text(actual) == expected
	})
	if g.Tables.Text(attrs.Construct) != construct || !valid {
		return fmt.Errorf("predicate construct or outcomes disagree with source")
	}
	return nil
}

func normalReturnFacts(g *pdg.Graph, node pdg.Node) error {
	if node.Category != pdg.Return {
		return nil
	}
	if g.NodeAttributes(node).ExitKind != pdg.NormalExit {
		return fmt.Errorf("source return must record a normal exit")
	}
	return nil
}

func (a *inventory) sourceLiteral(g *pdg.Graph, node pdg.Node, source ast.Node) error {
	if node.Category != pdg.Computation {
		return nil
	}
	attrs := g.NodeAttributes(node)
	switch source := source.(type) {
	case *ast.BasicLit:
		return knownValue(g, attrs.Literal, source.Value, "literal")
	case *ast.Ident:
		constant, ok := a.info.Uses[source].(*types.Const)
		if !ok {
			return nil
		}
		return knownValue(g, attrs.Literal, constant.Val().ExactString(), "constant value")
	default:
		return nil
	}
}

func sourceWriteForm(g *pdg.Graph, attrs pdg.Attributes, source ast.Node) error {
	write := ""
	switch source := source.(type) {
	case *ast.AssignStmt:
		write = source.Tok.String()
	case *ast.IncDecStmt:
		write = source.Tok.String()
	}
	if write != "" && g.Tables.Text(attrs.WriteForm) != write {
		return fmt.Errorf("write form disagrees with source")
	}
	return nil
}

func sourceKind(g *pdg.Graph, node pdg.Node, source ast.Node) error {
	if node.Synthetic && node.Category != pdg.Merge {
		if node.OriginalKind.Status != pdg.NotApplicable {
			return fmt.Errorf("lowered operation falsely claims a source AST kind")
		}
		return nil
	}
	return knownValue(g, node.OriginalKind, fmt.Sprintf("%T", source), "original kind")
}

func knownValue(g *pdg.Graph, fact pdg.Fact, expected, field string) error {
	if fact.Status != pdg.Known || g.Tables.Text(fact.Value) != expected {
		return fmt.Errorf("%s disagrees with typed source", field)
	}
	return nil
}

func sourceOperator(source ast.Node) string {
	switch source := source.(type) {
	case *ast.BasicLit:
		return "literal"
	case *ast.UnaryExpr:
		return source.Op.String()
	case *ast.BinaryExpr:
		return source.Op.String()
	case *ast.ParenExpr:
		return "parentheses"
	default:
		return ""
	}
}
