package cpp

import (
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
)

type sourcePatterns struct {
	guards       []patterns.GuardClauseHit
	booleans     []patterns.ConstantBoolReturnHit
	emptyCatches []patterns.CPPEmptyCatchHit
}

func (p *sourcePatterns) guard(node *vertex) {
	if isElseIf(node) {
		return
	}
	then, otherwise := ifBranches(node)
	if singleReturn(otherwise) == nil {
		return
	}
	count := statementCount(then)
	if count < 3 {
		return
	}
	p.guards = append(p.guards, patterns.GuardClauseHit{Line: node.number("LINE_NUMBER"), BodyStmts: count})
}

func (p *sourcePatterns) boolean(node *vertex) {
	then, otherwise := ifBranches(node)
	value, known := literalBool(returnedExpression(singleReturn(then)))
	if !known {
		return
	}
	otherReturn := singleReturn(otherwise)
	if otherwise == nil {
		otherReturn = nextReturn(node)
	}
	other, known := literalBool(returnedExpression(otherReturn))
	if !known || value != other {
		return
	}
	end := max(node.number("LINE_NUMBER_END"), otherReturn.number("LINE_NUMBER"))
	p.booleans = append(p.booleans, patterns.ConstantBoolReturnHit{Line: node.number("LINE_NUMBER"), EndLine: end, Value: value})
}

func detectSourcePatterns(method *vertex, nodes []*vertex) sourcePatterns {
	var result sourcePatterns
	boolean := booleanMethod(method)
	for _, node := range nodes {
		if node.number("LINE_NUMBER") < 1 {
			continue
		}
		result.guard(node)
		if boolean {
			result.boolean(node)
		}
		if emptyCatch(node) {
			result.emptyCatches = append(result.emptyCatches, patterns.CPPEmptyCatchHit{NodeID: strconv.FormatUint(node.ID, 10), Line: node.number("LINE_NUMBER")})
		}
	}
	return result
}

func booleanMethod(method *vertex) bool {
	for _, child := range method.Children {
		if child.Label == "METHOD_RETURN" && child.text("TYPE_FULL_NAME") == "bool" {
			return true
		}
	}
	return false
}

func ordinaryIf(node *vertex) bool {
	if !isIf(node) {
		return false
	}
	return runtimeIfCode(node.text("CODE"))
}

func runtimeIfCode(code string) bool {
	code = strings.ReplaceAll(strings.ReplaceAll(code, "\\\r\n", ""), "\\\n", "")
	rest, found := strings.CutPrefix(strings.TrimSpace(code), "if")
	if !found {
		return false
	}
	rest, valid := trimCPPTrivia(rest)
	if !valid {
		return false
	}
	return strings.HasPrefix(rest, "(")
}

func trimCPPTrivia(code string) (string, bool) {
	for {
		code = strings.TrimLeft(code, " \t\r\n\v\f")
		switch {
		case strings.HasPrefix(code, "//"):
			_, code, _ = strings.Cut(code, "\n")
		case strings.HasPrefix(code, "/*"):
			_, rest, closed := strings.Cut(code[2:], "*/")
			if !closed {
				return "", false
			}
			code = rest
		default:
			return code, true
		}
	}
}

func simpleIf(node *vertex) bool {
	if !ordinaryIf(node) {
		return false
	}
	if len(node.Children) < 2 || len(node.Children) > 3 {
		return false
	}
	condition := node.Out["CONDITION"]
	return len(condition) == 1 && node.Children[0] == condition[0]
}

func ifBranches(node *vertex) (*vertex, *vertex) {
	if !simpleIf(node) {
		return nil, nil
	}
	then := node.Children[1]
	if then.Label != "BLOCK" {
		return nil, nil
	}
	if len(node.Children) == 2 {
		return then, nil
	}
	otherwise := node.Children[2]
	if otherwise.Label != "BLOCK" {
		return nil, nil
	}
	return then, otherwise
}

func singleReturn(block *vertex) *vertex {
	if block == nil || block.Label != "BLOCK" {
		return nil
	}
	if len(block.Children) != 1 {
		return nil
	}
	if block.Children[0].Label != "RETURN" {
		return nil
	}
	return block.Children[0]
}

func statementCount(block *vertex) int {
	if block == nil {
		return 0
	}
	count := 0
	for _, child := range block.Children {
		switch child.Label {
		case "LOCAL", "METHOD", "TYPE_DECL":
			continue
		}
		count++
	}
	return count
}

func isElseBranch(node *vertex) bool {
	parent := node.Parent
	return parent != nil && isIf(parent) && len(parent.Children) == 3 && parent.Children[2] == node
}

func isElseIf(node *vertex) bool {
	if isElseBranch(node) {
		return true
	}
	if node.Parent == nil {
		return false
	}
	return isElseBranch(node.Parent) && elseIfBranch(node.Parent) == node
}

func returnedExpression(node *vertex) *vertex {
	if node == nil || node.Label != "RETURN" {
		return nil
	}
	if len(node.Children) != 1 {
		return nil
	}
	return node.Children[0]
}

func literalBool(node *vertex) (bool, bool) {
	if node == nil || node.Label != "LITERAL" {
		return false, false
	}
	if node.text("TYPE_FULL_NAME") != "bool" {
		return false, false
	}
	switch strings.TrimSpace(node.text("CODE")) {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}

func nextReturn(node *vertex) *vertex {
	if node.Parent == nil || node.Parent.Label != "BLOCK" {
		return nil
	}
	children := node.Parent.Children
	for i := 0; i+1 < len(children); i++ {
		if children[i] == node && children[i+1].Label == "RETURN" {
			return children[i+1]
		}
	}
	return nil
}

func emptyCatch(node *vertex) bool {
	if node.Label != "CONTROL_STRUCTURE" {
		return false
	}
	if node.text("CONTROL_STRUCTURE_TYPE") != "CATCH" {
		return false
	}
	for _, child := range node.Children {
		if child.Label != "BLOCK" {
			continue
		}
		if strings.Join(strings.Fields(child.text("CODE")), "") != "{}" {
			return false
		}
		return statementCount(child) == 0
	}
	return false
}
